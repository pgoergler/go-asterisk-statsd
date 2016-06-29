package logging

import (
	"io"
	"log"
	"log/syslog"
)

var (
	// Trace log.Logger ptr
	Trace *log.Logger

	// Info log.Logger ptr
	Info *log.Logger

	// Warning log.Logger ptr
	Warning *log.Logger

	// Error log.Logger ptr
	Error *log.Logger
)

// Init initialize loggers
func Init(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer,
	syslogTag string) {

	sysloger, err := syslog.New(syslog.LOG_NOTICE, syslogTag)
	if err == nil {
		infoHandle = io.MultiWriter(sysloger, infoHandle)
		warningHandle = io.MultiWriter(sysloger, warningHandle)
		errorHandle = io.MultiWriter(sysloger, errorHandle)
	}

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}
