package asterisk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCall(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")

	assert.Equal("source", call.Source, "Source not correctly set")
	assert.Equal("destination", call.Destination, "Destination not correctly set")
	assert.Equal("uniqueId", call.UniqueID, "UniqueID not correctly set")
	assert.Equal("SIP/Trunk-channel-1234deadbeef", call.Channel, "Channel not correctly set")
	assert.Equal("context", call.Context, "Context not correctly set")
	assert.False(call.CreatedAt.IsZero(), "CreatedAt not correctly set")
}

func TestGetTrunkName(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	assert.Equal("Trunk-channel", call.GetTrunkName(), "Trunk not extracted")

	call = NewCall("source", "destination", "uniqueId", "SIP/channel", "context")
	assert.Equal("SIP/channel", call.GetTrunkName(), "Trunk not extracted")

}

func TestAnswered(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	assert.True(call.AnsweredAt.IsZero(), "AnsweredAt already set")

	call.Answered()
	assert.False(call.AnsweredAt.IsZero(), "AnsweredAt not correctly set")
}

func TestRinging(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	assert.True(call.RingingAt.IsZero(), "RingingAt already set")

	call.Ringing()
	assert.False(call.RingingAt.IsZero(), "RingingAt already set")
}

func TestHangingUp(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	assert.True(call.HangupAt.IsZero(), "HangupAt already set")

	call.HangingUp("")
	assert.False(call.HangupAt.IsZero(), "HangupAt not correctly set")

	hangupAt := call.HangupAt.Nanosecond()
	assert.Equal("", call.HangupCause, "HangupCause not correctly set")

	call.HangingUp("16")
	assert.Equal("16", call.HangupCause, "HangupCause not correctly set")
	assert.Equal(hangupAt, call.HangupAt.Nanosecond(), "HangupAt must not be overrided")

	call.HangingUp("-1")
	assert.Equal("16", call.HangupCause, "HangupCause must not be overrided")
	assert.Equal(hangupAt, call.HangupAt.Nanosecond(), "HangupAt must not be overrided")

	assert.Equal(int64(0), call.ActiveDuration, "ActiveDuration not correctly set")
	assert.Equal(int64(0), call.TotalDuration, "TotalDuration not correctly set")
}

func TestHangupAlone(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	assert.True(call.HangupAt.IsZero(), "HangupAt already set")

	<-time.After(time.Millisecond * 10)
	call.Hangup("", "unknown")

	assert.False(call.HangupAt.IsZero(), "HangupAt not correctly set")
	assert.Equal("", call.HangupCause, "HangupCause not correctly set")
	assert.Equal("unknown", call.HangupCauseTxt, "HangupCauseTxt not correctly set")

	hangupAt := call.HangupAt.Nanosecond()

	call.Hangup("16", "Normal")
	assert.Equal("16", call.HangupCause, "HangupCause not correctly set")
	assert.Equal("Normal", call.HangupCauseTxt, "HangupCauseTxt must be overrided")
	assert.Equal(hangupAt, call.HangupAt.Nanosecond(), "HangupAt must not be overrided")

	call.Hangup("-1", "XXX")
	assert.Equal("16", call.HangupCause, "HangupCause must not be overrided")
	assert.Equal("XXX", call.HangupCauseTxt, "HangupCauseTxt must be overrided")
	assert.Equal(hangupAt, call.HangupAt.Nanosecond(), "HangupAt must not be overrided")

	assert.Equal(int64(0), call.ActiveDuration, "ActiveDuration not correctly set")
	assert.NotEqual(int64(0), call.TotalDuration, "TotalDuration not correctly set")

}

func TestHangup(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	call.Ringing()
	<-time.After(time.Millisecond * 10)

	call.Answered()
	<-time.After(time.Millisecond * 50)

	call.HangingUp("16")
	<-time.After(time.Millisecond * 40)
	call.Hangup("-1", "Normal")

	assert.Equal("16", call.HangupCause, "HangupCause not correctly set")
	assert.Equal("Normal", call.HangupCauseTxt, "HangupCauseTxt must be overrided")
	assert.InDelta(int64(50), call.ActiveDuration, 10, "ActiveDuration not correctly set, expected ~50ms +-10ms")
	assert.InDelta(int64(100), call.TotalDuration, 10, "TotalDuration not correctly set, expected ~100ms +-10ms")
}

func TestDispositionAnswered(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	call.Ringing()
	<-time.After(time.Millisecond * 10)

	call.Answered()
	<-time.After(time.Millisecond * 50)

	call.HangingUp("16")
	<-time.After(time.Millisecond * 40)
	call.Hangup("-1", "Normal")

	assert.Equal("ANSWERED", call.Disposition(), "Wrong Disposition()")
}

func TestDispositionRinging(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	call.Ringing()
	<-time.After(time.Millisecond * 50)
	call.Hangup("16", "Normal")

	assert.Equal("NOANSWER", call.Disposition(), "Wrong Disposition()")

	call = NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	call.Ringing()
	<-time.After(time.Millisecond * 50)
	call.Hangup("19", "Normal")

	assert.Equal("NOANSWER", call.Disposition(), "Wrong Disposition()")

}

func TestDispositionBusy(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	call.Ringing()
	<-time.After(time.Millisecond * 50)
	call.Hangup("17", "Normal")

	assert.Equal("BUSY", call.Disposition(), "Wrong Disposition()")
}

func TestDispositionFailed(t *testing.T) {
	assert := assert.New(t)
	call := NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	call.Hangup("19", "Normal")

	assert.Equal("FAILED", call.Disposition(), "Wrong Disposition()")

	call = NewCall("source", "destination", "uniqueId", "SIP/Trunk-channel-1234deadbeef", "context")
	call.Hangup("-1", "Normal")

	assert.Equal("FAILED", call.Disposition(), "Wrong Disposition()")
}
