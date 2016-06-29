package statsdami

import (
	"github.com/pgoergler/go-asterisk-statsd/asterisk"
	"github.com/pgoergler/go-asterisk-statsd/asterisk/ami"
	"github.com/pgoergler/go-asterisk-statsd/logging"

	"sync"

	"github.com/quipo/statsd"
)

type statsdEventHandler func(*statsd.StatsdClient, *asterisk.Call, *ami.Event, map[string]string)

var callsMutex = new(sync.RWMutex)
var calls = make(map[string]*asterisk.Call)

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

	err := client.GaugeDelta(MeasureGauge("concurrent", tags), +1)
	if err != nil {
		logging.Error.Println(err)
	}

	err = client.Incr(Measure("calls", tags), +1)
	if err != nil {
		logging.Error.Println(err)
	}

	err = client.GaugeDelta(MeasureGauge("concurrent", map[string]string{"trunk": "All"}), +1)
	if err != nil {
		logging.Error.Println(err)
	}

	err = client.Incr(Measure("calls", map[string]string{"trunk": "All"}), +1)
	if err != nil {
		logging.Error.Println(err)
	}

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

	err := client.GaugeDelta(MeasureGauge("concurrent", tags), -1)
	if err != nil {
		logging.Error.Println(err)
	}

	err = client.GaugeDelta(MeasureGauge("concurrent", map[string]string{"trunk": "All"}), -1)
	if err != nil {
		logging.Error.Println(err)
	}

	callTags := make(map[string]string)
	for k, v := range tags {
		callTags[k] = v
	}

	cause := call.HangupCause
	if cause == "" {
		cause = "-"
	}

	causeTxt := call.HangupCauseTxt
	if causeTxt == "" {
		causeTxt = "-"
	}

	callTags["cause"] = cause
	callTags["cause_txt"] = causeTxt
	callTags["disposition"] = call.Disposition()

	err = client.Timing(Measure("active_duration", callTags), call.ActiveDuration)
	if err != nil {
		logging.Error.Println(err)
	}

	err = client.Timing(Measure("total_duration", callTags), call.TotalDuration)
	if err != nil {
		logging.Error.Println(err)
	}

	callTags["trunk"] = "All"
	err = client.Timing(Measure("active_duration", callTags), call.ActiveDuration)
	if err != nil {
		logging.Error.Println(err)
	}

	err = client.Timing(Measure("total_duration", callTags), call.TotalDuration)
	if err != nil {
		logging.Error.Println(err)
	}

}
