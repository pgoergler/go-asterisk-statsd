package statsdami

import (
	"sync"
)

// Measure build statsd aspect from name and tags
func Measure(measure string, tags map[string]string) string {
	aspect := measure
	for k, v := range tags {
		aspect += "," + k + "=" + v
	}
	return aspect
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

// ShouldReset return true if a Gauge should be rested in statsd server
func ShouldReset(aspect string) bool {
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
