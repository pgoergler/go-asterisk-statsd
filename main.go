package main

import (
	"io/ioutil"
	"os"

	"flag"
	"regexp"
	"time"

	"github.com/pgoergler/go-asterisk-statsd/asterisk/ami"
	"github.com/pgoergler/go-asterisk-statsd/logging"
	"github.com/pgoergler/go-asterisk-statsd/statsd-ami"

	"github.com/quipo/statsd"
)

type eventHandler func(*statsd.StatsdClient, *ami.Event, map[string]string)

func main() {

	logging.Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr, "asterisk-monitor")

	asteriskInfo := flag.String("asterisk", "", "asterisk connection info. format: user:password@host:port")
	statsdInfo := flag.String("statsd", "", "statsd connection info. format: host:port/prefix")
	flag.Parse()

	statsdEnabled := true
	regexStatsd, _ := regexp.Compile("^(.+?)(:(.*?))?(/(.*?))?$")
	if !regexStatsd.MatchString(*statsdInfo) {
		logging.Error.Println("statsd disabled")
		statsdEnabled = false
	}

	var statsdclient *statsd.StatsdClient
	if statsdEnabled {

		matches := regexStatsd.FindAllStringSubmatch(*statsdInfo, -1)
		statsdHost := matches[0][1] + ":" + matches[0][3]
		statsdPrefix := matches[0][5]

		statsdclient = statsd.NewStatsdClient(statsdHost, statsdPrefix)
		err := statsdclient.CreateSocket()
		if nil != err {
			logging.Error.Println(err)
			os.Exit(1)
		}
	}

	regexAsterisk, _ := regexp.Compile("^(.*?):(.*?)@(.*?)$")
	if !regexAsterisk.MatchString(*asteriskInfo) {
		logging.Error.Println("could not parse asterisk connection info <" + (*asteriskInfo) + ">")
		return
	}

	matches := regexAsterisk.FindAllStringSubmatch((*asteriskInfo), -1)
	asteriskUsername := matches[0][1]
	asteriskPassword := matches[0][2]
	asteriskAddress := matches[0][3]

	ami := ami.New(asteriskAddress, asteriskUsername, asteriskPassword)

	//ami.RegisterDefaultHandler(eventDefaultHandler)

	ami.RegisterHandler("Newchannel", statsdami.NewHandler(statsdclient, statsdami.EventNewChannelHandler))
	ami.RegisterHandler("Newstate", statsdami.NewHandler(statsdclient, statsdami.EventNewStateHandler))
	ami.RegisterHandler("SoftHangupRequest", statsdami.NewHandler(statsdclient, statsdami.EventSoftHangupHandler))
	ami.RegisterHandler("Hangup", statsdami.NewHandler(statsdclient, statsdami.EventHangupHandler))

	for {
		ami.StopKeepAlive()

		if err := ami.Connect(map[string]string{"Events": "call,command"}); err != nil {
			logging.Error.Println(err)
		} else {
			logging.Info.Println("Connected to", asteriskAddress)
			ami.KeepAlive(time.Second * 1)

			ami.Run()
			logging.Info.Println("Connection lost")
			ami.StopKeepAlive()
		}

		// reconnect after 100ms
		<-time.After(time.Millisecond * 100)
	}
}

func eventDefaultHandler(message *ami.Event) {
}
