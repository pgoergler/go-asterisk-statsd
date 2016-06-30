package statsdami

import (
	"log"
	"sync"

	"github.com/pgoergler/go-asterisk-statsd/logging"
	"github.com/quipo/statsd"
)

// Measure a proxy object to handle statsd measurements
type Measure struct {
	name string
	tags map[string]string

	client *statsd.StatsdClient
}

var gaugeMutex = new(sync.RWMutex)
var gaugesCounter = make(map[string]bool)

// DumpGauges All gauges
func DumpGauges(logger *log.Logger) {
	gaugeMutex.Lock()
	defer gaugeMutex.Unlock()
	logger.Println(len(calls), " gauge")
	for k := range gaugesCounter {
		logger.Printf("%s,", k)
	}
	logger.Println("")
}

//GetGaugeCount return gauge count
func GetGaugeCount() int {
	gaugeMutex.Lock()
	defer gaugeMutex.Unlock()
	return len(gaugesCounter)
}

//GetAllGauges return all gauges
func GetAllGauges() []string {
	gaugeMutex.Lock()
	defer gaugeMutex.Unlock()
	keys := make([]string, 0, len(gaugesCounter))
	for k := range gaugesCounter {
		keys = append(keys, k)
	}
	return keys
}

// ResetAllGauges clean gaugesCounter
func ResetAllGauges() {
	gaugeMutex.Lock()
	defer gaugeMutex.Unlock()
	gaugesCounter = make(map[string]bool)
}

// shouldReset return true if a Gauge should be rested in statsd server
func shouldResetGauge(aspect string) bool {
	gaugeMutex.Lock()
	defer gaugeMutex.Unlock()

	_, ok := gaugesCounter[aspect]
	if ok {
		return false
	}
	return RegisterGauge(aspect)
}

// RegisterGauge add a gauge in gaugesCounter
func RegisterGauge(aspect string) bool {
	gaugeMutex.Lock()
	defer gaugeMutex.Unlock()

	gaugesCounter[aspect] = true
	return true
}

// NewMeasure build statsd aspect from name and tags
func NewMeasure(client *statsd.StatsdClient, name string, tags map[string]string) *Measure {
	m := &Measure{
		name:   name,
		tags:   make(map[string]string),
		client: client,
	}

	for k, v := range tags {
		m.tags[k] = v
	}
	return m
}

// Tag add a tag to a Measure
func (m *Measure) Tag(name string, value string) *Measure {
	m.tags[name] = value
	return m
}

// GetAspect return the statsd aspect of the Measure
func (m *Measure) GetAspect() string {
	aspect := m.name
	for k, v := range m.tags {
		aspect += "," + k + "=" + v
	}
	return aspect
}

// IncrementCounter a Counter
func (m *Measure) IncrementCounter() {
	err := (*m.client).Incr(m.GetAspect(), 1)
	if err != nil {
		logging.Error.Println(err)
	}
}

// IncrementGauge a Gauge
func (m *Measure) IncrementGauge() {
	aspect := m.GetAspect()
	if shouldResetGauge(aspect) {
		err := (*m.client).Gauge(aspect, 0)
		if err != nil {
			logging.Error.Println(err)
		}
	}

	err := (*m.client).GaugeDelta(aspect, 1)
	if err != nil {
		logging.Error.Println(err)
	}
}

// DecrementGauge a Gauge
func (m *Measure) DecrementGauge() {
	aspect := m.GetAspect()
	if shouldResetGauge(aspect) {
		err := (*m.client).Gauge(aspect, 0)
		if err != nil {
			logging.Error.Println(err)
		}
	}

	err := (*m.client).GaugeDelta(aspect, -1)
	if err != nil {
		logging.Error.Println(err)
	}
}

// Timing warp Statsd.Timing
func (m *Measure) Timing(delta int64) {
	err := (*m.client).Timing(m.GetAspect(), delta)
	if err != nil {
		logging.Error.Println(err)
	}
}
