package asterisk

import (
	"regexp"
	"time"
)

// Call define a call structure
type Call struct {
	Source      string
	Destination string
	UniqueID    string
	Channel     string
	Context     string

	AccountCode string
	CreatedAt   time.Time
	RingingAt   time.Time
	AnsweredAt  time.Time
	HangupAt    time.Time

	ActiveDuration int64
	TotalDuration  int64

	HangupCause    string
	HangupCauseTxt string
}

// NewCall create a Call instance
func NewCall(source string, destination string, uniqueID string, channel string, context string) *Call {
	call := Call{
		Source:         source,
		Destination:    destination,
		UniqueID:       uniqueID,
		Channel:        channel,
		Context:        context,
		CreatedAt:      time.Now(),
		ActiveDuration: 0,
		TotalDuration:  0,
	}

	return &call
}

// GetTrunkName return the trunk of the call (remove SIP/ and channel id)
func (c *Call) GetTrunkName() string {
	r, err := regexp.Compile("^(SIP|.*?)/(.*)\\-[0-9a-f]+$")
	if err == nil {
		if r.MatchString(c.Channel) {
			values := r.FindAllStringSubmatch(c.Channel, -1)
			return values[0][2]
		}
	}
	return c.Channel
}

// Answered mark the Call as answered
func (c *Call) Answered() {
	c.AnsweredAt = time.Now()
}

// Ringing mark the Call as Ringing
func (c *Call) Ringing() {
	c.RingingAt = time.Now()
}

// HangingUp mark the Call as HangingUp
func (c *Call) HangingUp(cause string) {
	if c.HangupAt.IsZero() {
		c.HangupAt = time.Now()
	}

	if c.HangupCause == "" {
		c.HangupCause = cause
	}

	if c.AnsweredAt.IsZero() {
		c.ActiveDuration = 0
	} else {
		c.ActiveDuration = c.HangupAt.Sub(c.AnsweredAt).Nanoseconds() / int64(1000000)
	}
}

// Hangup mark the Call as hangup
func (c *Call) Hangup(cause string, causeTxt string) {
	if c.HangupAt.IsZero() {
		c.HangupAt = time.Now()
	}

	if c.HangupCause == "" {
		c.HangupCause = cause
	}
	c.HangupCauseTxt = causeTxt
	c.TotalDuration = time.Now().Sub(c.CreatedAt).Nanoseconds() / int64(1000000)
}

// Disposition return the disposition
func (c *Call) Disposition() string {
	switch c.HangupCause {
	case "16": // normal clearing
		{
			if c.ActiveDuration == 0 {
				return "NOANSWER"
			}
			return "ANSWERED"
		}
	case "17":
		return "BUSY"
	case "19":
		{
			if c.RingingAt.IsZero() {
				return "FAILED"
			}
			return "NOANSWER"
		}
	}
	return "FAILED"
}
