package statsdami

import (
	"github.com/pgoergler/go-asterisk-statsd/asterisk"
	"github.com/pgoergler/go-asterisk-statsd/asterisk/ami"
	"github.com/pgoergler/go-asterisk-statsd/logging"

	"log"
	"sync"

	"github.com/quipo/statsd"
)

type statsdEventHandler func(*statsd.StatsdClient, *asterisk.Call, *ami.Event, map[string]string)

var callsMutex = new(sync.RWMutex)
var calls = make(map[string]*asterisk.Call)

// GetPendingCallsCount return number of pending calls (not deleted)
func GetPendingCallsCount() int {
	callsMutex.Lock()
	defer callsMutex.Unlock()
	return len(calls)
}

// Dump pending calls
func Dump(logger *log.Logger) {
	callsMutex.Lock()
	defer callsMutex.Unlock()
	logger.Println(len(calls), " pending calls")
	for k, v := range calls {
		logger.Printf("%s => %v\n", k, v)
	}
}

func watch(call *asterisk.Call) {
	callsMutex.Lock()
	defer callsMutex.Unlock()
	calls[call.UniqueID] = call
}

func unwatch(call *asterisk.Call) {
	callsMutex.Lock()
	defer callsMutex.Unlock()
	delete(calls, call.UniqueID)
}

func isWatched(uniqueID string) (*asterisk.Call, bool) {
	callsMutex.Lock()
	defer callsMutex.Unlock()
	value, found := calls[uniqueID]
	return value, found
}

func mapGetter(params map[string]string) func(string, string) string {
	return func(key string, defaultValue string) string {
		value, ok := params[key]
		if !ok {
			return defaultValue
		}
		return value
	}
}

// NewHandler call handler with extra paramters
//  handler(*statsd.StatsdClient, *asterisk.Call, *ami.Event, map[string]string)
func NewHandler(client *statsd.StatsdClient, handler statsdEventHandler) func(*ami.Event) {
	return func(message *ami.Event) {
		get := mapGetter(message.Params)

		uniqueID := get("Uniqueid", "")
		if uniqueID == "" {
			logging.Error.Println("no uniqueID found in", message)
			return
		}

		call, found := isWatched(uniqueID)
		if !found {
			// call not watched
			// is it a new call or discard?
			if message.ID != "Newchannel" {
				return
			}

			call = asterisk.NewCall(
				get("CallerIDNum", "anonymous"),
				get("Exten", "s"),
				uniqueID,
				get("Channel", "not_set"),
				get("Context", "not_set"))

			watch(call)
		}

		handler(client, call, message, map[string]string{"trunk": call.GetTrunkName()})

		if message.ID == "Hangup" {
			unwatch(call)
		}
	}
}

func eventDefaultHandler(client *statsd.StatsdClient,
	call *asterisk.Call, message *ami.Event, tags map[string]string) {
}

//EventNewChannelHandler handle new call
func EventNewChannelHandler(client *statsd.StatsdClient,
	call *asterisk.Call, message *ami.Event, tags map[string]string) {

	if client == nil {
		return
	}
	NewMeasure(client, "concurrent", tags).IncrementGauge()
	NewMeasure(client, "calls", tags).IncrementCounter()

	NewMeasure(client, "concurrent", tags).Tag("trunk", "All").IncrementGauge()
	NewMeasure(client, "calls", tags).Tag("trunk", "All").IncrementCounter()
}

// EventNewStateHandler handle Call state changed
func EventNewStateHandler(client *statsd.StatsdClient,
	call *asterisk.Call, message *ami.Event, tags map[string]string) {

	get := mapGetter(message.Params)

	state := get("Channelstatedesc", "")
	switch state {
	case "Ring", "Ringing":
		call.Ringing()
	case "Up":
		call.Answered()
	default:
		logging.Error.Println("Unknown state ", state, " event:", message)
	}
}

// EventNewAccountCodeHandler handle Call AccountCode changed
func EventNewAccountCodeHandler(client *statsd.StatsdClient,
	call *asterisk.Call, message *ami.Event, tags map[string]string) {

	get := mapGetter(message.Params)
	call.AccountCode = get("AccountCode", "")
}

// EventSoftHangupHandler handle Call soft hangup
func EventSoftHangupHandler(client *statsd.StatsdClient,
	call *asterisk.Call, message *ami.Event, tags map[string]string) {

	get := mapGetter(message.Params)
	call.HangingUp(get("Cause", ""))
}

// EventHangupHandler handle Call soft hangup
func EventHangupHandler(client *statsd.StatsdClient,
	call *asterisk.Call, message *ami.Event, tags map[string]string) {

	get := mapGetter(message.Params)
	call.Hangup(get("Cause", ""), get("Cause-Txt", ""))

	if client == nil {
		return
	}

	cause := call.HangupCause
	if cause == "" {
		cause = "-"
	}

	causeTxt := call.HangupCauseTxt
	if causeTxt == "" {
		causeTxt = "-"
	}

	NewMeasure(client, "concurrent", tags).DecrementGauge()
	NewMeasure(client, "concurrent", tags).Tag("trunk", "All").DecrementGauge()

	NewMeasure(client, "active_duration", tags).
		Tag("cause", cause).
		Tag("cause_txt", causeTxt).
		Tag("disposition", call.Disposition()).
		Timing(call.ActiveDuration)

	NewMeasure(client, "active_duration", tags).
		Tag("cause", cause).
		Tag("cause_txt", causeTxt).
		Tag("disposition", call.Disposition()).
		Tag("trunk", "All").
		Timing(call.ActiveDuration)

	NewMeasure(client, "total_duration", tags).
		Tag("cause", cause).
		Tag("cause_txt", causeTxt).
		Tag("disposition", call.Disposition()).
		Timing(call.TotalDuration)

	NewMeasure(client, "total_duration", tags).
		Tag("cause", cause).
		Tag("cause_txt", causeTxt).
		Tag("disposition", call.Disposition()).
		Tag("trunk", "All").
		Timing(call.TotalDuration)

}
