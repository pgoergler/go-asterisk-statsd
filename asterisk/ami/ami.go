package ami

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"time"

	"github.com/pgoergler/go-asterisk-statsd/logging"
	"github.com/pgoergler/go-asterisk-statsd/uuid"
)

var errNotEvent = errors.New("Not Event")

// ErrNotAMI raised when not response expected protocol AMI
var ErrNotAMI = errors.New("Server not AMI interface")

// Params for the actions
type Params map[string]string

// Action is an alias for map[string]string
type Action map[string]string

// Response is an alias for map[string]string
type Response struct {
	ID     string
	Status string
	Params map[string]string
}

// Event is an alias for map[string]string
type Event struct {
	//Identification of event Event: xxxx
	ID string

	Privilege []string

	// Params  of arguments received
	Params map[string]string
}

type eventHandlerFunc func(*Event)

// Client a connection to Asterisk Manager Interface
type Client struct {
	address  string
	username string
	password string

	conn        *textproto.Conn
	connRaw     io.ReadWriteCloser
	useTLS      bool
	unsecureTLS bool
	tlsConfig   *tls.Config

	// chanActions      chan Action
	responses map[string]chan *Response

	// Events for client parse
	Events chan *Event

	// Error Raise on logic
	Error chan error

	//NetError a network error
	NetError chan error

	mutexAsyncAction *sync.RWMutex
	mutexObject      *sync.RWMutex

	// network wait for a new connection
	waitNewConnection chan struct{}

	defaultHandler    eventHandlerFunc
	handlers          map[string]eventHandlerFunc
	keepAliveExitChan chan bool
}

// UseTLS option which enable tls connection for client
func UseTLS(c *Client) {
	c.useTLS = true
}

// UseTLSConfig return an option to set TLS configuration
func UseTLSConfig(config *tls.Config) func(*Client) {
	return func(c *Client) {
		c.tlsConfig = config
		c.useTLS = true
	}
}

// Dump memory objects
func Dump(client *Client, logger *log.Logger) {
	logger.Println(len(client.responses), " responses")
	for k := range client.responses {
		logger.Println("response id:" + k)
	}

}

// New create a Client
func New(address string, user string, password string, options ...func(*Client)) (client *Client) {
	client = &Client{
		address:           address,
		username:          user,
		password:          password,
		mutexAsyncAction:  new(sync.RWMutex),
		mutexObject:       new(sync.RWMutex),
		waitNewConnection: make(chan struct{}),
		responses:         make(map[string]chan *Response),
		Events:            nil,
		Error:             make(chan error, 1),
		NetError:          make(chan error, 1),
		useTLS:            false,
		unsecureTLS:       false,
		tlsConfig:         new(tls.Config),
		handlers:          make(map[string]eventHandlerFunc),
	}

	for _, op := range options {
		op(client)
	}
	return client
}

// SetEventChannel set a channel to send Event received
func (client *Client) SetEventChannel(channel chan *Event) {
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()
	client.Events = channel
}

// RegisterDefaultHandler register a default handler for all events
func (client *Client) RegisterDefaultHandler(f eventHandlerFunc) error {
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()
	if client.defaultHandler != nil {
		return errors.New("DefaultHandler already registered")
	}
	client.defaultHandler = f
	return nil
}

// UnregisterDefaultHandler unregister the default handler if exists
func (client *Client) UnregisterDefaultHandler(f eventHandlerFunc) error {
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()
	if client.defaultHandler == nil {
		return errors.New("DefaultHandler not registered")
	}
	client.defaultHandler = nil
	return nil
}

// RegisterHandler register an handler for a specific event
func (client *Client) RegisterHandler(eventID string, f eventHandlerFunc) error {
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()
	if client.handlers[eventID] != nil {
		return errors.New("Handler already registered")
	}
	client.handlers[eventID] = f
	return nil
}

// UnregisterHandler unregister an handler for a specific event
func (client *Client) UnregisterHandler(eventID string) error {
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()
	if client.handlers[eventID] == nil {
		return errors.New("Handler not registered")
	}
	client.handlers[eventID] = nil
	return nil
}

// Connect create a new connection and authenticate
func (client *Client) Connect(parameters map[string]string) (err error) {
	if client.useTLS {
		client.tlsConfig.InsecureSkipVerify = client.unsecureTLS
		client.connRaw, err = tls.Dial("tcp", client.address, client.tlsConfig)
	} else {
		client.connRaw, err = net.Dial("tcp", client.address)
	}

	if err != nil {
		return err
	}

	client.conn = textproto.NewConn(client.connRaw)
	label, err := client.conn.ReadLine()
	if err != nil {
		return err
	}

	if strings.Contains(label, "Asterisk Call Manager") != true {
		return ErrNotAMI
	}

	err = client.login(parameters)
	if err != nil {
		return err
	}

	return nil
}

// KeepAlive periodicaly send "Ping" action to AMI server
func (client *Client) KeepAlive(interval time.Duration) {
	go func(client *Client, interval time.Duration) {
		client.mutexObject.Lock()
		client.keepAliveExitChan = make(chan bool, 1)
		client.mutexObject.Unlock()

		for {
			select {
			case <-client.keepAliveExitChan:
				return
			case <-time.After(interval):
				{
					if _, err := client.Action("Ping", nil); err != nil {
						client.Close()
					}
				}
			}
		}
	}(client, interval)
}

// StopKeepAlive StopKeepAlive gorouting
func (client *Client) StopKeepAlive() {
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()
	if client.keepAliveExitChan != nil {
		client.keepAliveExitChan <- true
	}
}

func (client *Client) login(parameters map[string]string) error {
	params := Params{"Username": client.username, "Secret": client.password}
	if parameters != nil {
		for k, v := range parameters {
			params[k] = v
		}
	}

	_, err := client.AsyncAction("Login", params)
	if err != nil {
		return err
	}

	data, err := client.conn.ReadMIMEHeader()
	if err != nil {
		return err
	}

	response, err := newResponse(&data)
	if err == nil {
		if (*response).Status == "Error" {
			return errors.New((*response).Params["Message"])
		}
		return nil
	}
	return err
}

// Run process socket waiting events and responses
func (client *Client) Run() (err error) {
	for {
		data, err := client.conn.ReadMIMEHeader()
		if err != nil {
			return err
		}

		if response, err := newResponse(&data); err == nil {
			client.notifyResponse(response)
			continue
		}

		if ev, err := newEvent(&data); err != nil {
			if err != errNotEvent {
				fmt.Println(err)
				client.Error <- err
			}
		} else {
			if client.Events != nil {
				client.Events <- ev
			}
			event := ev.ID

			if handler, found := client.handlers[event]; found {
				handler(ev)
			} else if client.defaultHandler != nil {
				client.defaultHandler(ev)
			}
			continue
		}
	}
}

// AsyncAction return chan for wait response of action with parameter *ActionID* this can be helpful for
// massive actions,
func (client *Client) AsyncAction(action string, params Params) (<-chan *Response, error) {
	var output string
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()

	output = fmt.Sprintf("Action: %s\r\n", strings.TrimSpace(action))
	if params == nil {
		params = Params{}
	}
	if _, ok := params["ActionID"]; !ok {
		params["ActionID"] = "go_ami:" + uuid.NewV4()
	}

	if _, ok := client.responses[params["ActionID"]]; !ok {
		client.responses[params["ActionID"]] = make(chan *Response, 1)
	}
	for k, v := range params {
		output = output + fmt.Sprintf("%s: %s\r\n", k, strings.TrimSpace(v))
	}
	if err := client.conn.PrintfLine("%s", output); err != nil {
		return nil, err
	}

	return client.responses[params["ActionID"]], nil
}

// Action send synchronously with params
func (client *Client) Action(action string, params Params) (*Response, error) {
	resp, err := client.AsyncAction(action, params)
	if err != nil {
		return nil, err
	}
	response := <-resp

	return response, nil
}

// GetPendingActionsCount return nb responses unproceed
func (client *Client) GetPendingActionsCount() int {
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()

	return len(client.responses)
}

// Close the connection to AMI
func (client *Client) Close() {
	logging.Trace.Println("logoff")
	client.Action("Logoff", nil)
	logging.Trace.Println("locking")
	client.mutexObject.Lock()
	defer client.mutexObject.Unlock()
	(client.connRaw).Close()
	logging.Trace.Println("closed")
}

func (client *Client) notifyResponse(response *Response) {
	go func() {
		client.mutexAsyncAction.RLock()
		chanResponse, found := client.responses[response.ID]
		client.mutexAsyncAction.RUnlock()

		if found {
			chanResponse <- response
			client.mutexAsyncAction.Lock()
			close(client.responses[response.ID])
			delete(client.responses, response.ID)
			client.mutexAsyncAction.Unlock()
		}
	}()
}

func newResponse(data *textproto.MIMEHeader) (*Response, error) {
	if data.Get("Response") == "" {
		return nil, errors.New("Not Response")
	}
	response := &Response{"", "", make(map[string]string)}
	for k, v := range *data {
		if k == "Response" {
			continue
		}
		response.Params[k] = v[0]
	}
	response.ID = data.Get("Actionid")
	response.Status = data.Get("Response")
	return response, nil
}

func newEvent(data *textproto.MIMEHeader) (*Event, error) {
	if data.Get("Event") == "" {
		return nil, errNotEvent
	}
	ev := &Event{data.Get("Event"), strings.Split(data.Get("Privilege"), ","), make(map[string]string)}
	for k, v := range *data {
		if k == "Event" || k == "Privilege" {
			continue
		}
		ev.Params[k] = v[0]
	}
	return ev, nil
}
