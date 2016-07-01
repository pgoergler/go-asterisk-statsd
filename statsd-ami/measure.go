package statsdami

import (
	"sync"
)

// Measure a proxy object to handle statsd measurements
type Measure struct {
	name string
	tags map[string]string

	client *statsd.Statsd
}

// MeasureGauge build statsd aspect from name and tags
func MeasureGauge(measure string, tags map[string]string) string {
	aspect := measure
	for k, v := range tags {
		aspect += "," + k + "=" + v
	}

	RegisterGauge(aspect)
	return aspect
}

var gaugeMutex = new(sync.RWMutex)
var gaugesCounter = make(map[string]bool)

// ResetAllGauges clean gaugesCounter
func ResetAllGauges() {
	gaugeMutex.Lock()
	defer gaugeMutex.Unlock()
	gaugesCounter = make(map[string]bool)
}

// shouldReset return true if a Gauge should be rested in statsd server
func shouldResetGauge(aspect string) bool {
	gaugeMutex.RLock()
	defer gaugeMutex.RUnlock()
	value, ok := gaugesCounter[aspect]
	if ok {
		return !value
	}
	return registerGauge(aspect)
}

func registerGauge(aspect string) bool {
	gaugesCounter[aspect] = true
	return true
}

// NewMeasure build statsd aspect from name and tags
func NewMeasure(client *statsd.Statsd, name string, tags map[string]string) *Measure {
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
