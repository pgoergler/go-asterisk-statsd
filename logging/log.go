package logging

import (
	"io"
	"io/ioutil"
	"log"
	"log/syslog"
)

var (
	// Trace log.Logger ptr
	Trace = log.New(ioutil.Discard, "TRACE: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Debug log.Logger ptr
	Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Info log.Logger ptr
	Info = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Warning log.Logger ptr
	Warning = log.New(ioutil.Discard, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Error log.Logger ptr
	Error = log.New(ioutil.Discard, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Dump log.Logger ptr
	Dump = log.New(ioutil.Discard, "", log.Ldate|log.Ltime|log.Lshortfile)
)

// Init logger with a writer
func Init(logger *log.Logger, writer io.Writer) {
	logger.SetOutput(writer)
}

// InitWithSyslog enable syslog for that logger
func InitWithSyslog(logger *log.Logger, writer io.Writer, syslogTag string) {
	sysloger, err := syslog.New(syslog.LOG_NOTICE, syslogTag)
	if err == nil {
		writer = io.MultiWriter(sysloger, writer)
	}

	Init(logger, writer)
}
