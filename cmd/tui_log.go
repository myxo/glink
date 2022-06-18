package main

import (
	"fmt"
	"github.com/juju/loggo"
)

type TuiLogger struct {
	Messages chan loggo.Entry
}

func NewTuiLogger() *TuiLogger {
	return &TuiLogger{Messages: make(chan loggo.Entry, 10)}
}

func (l *TuiLogger) Write(entry loggo.Entry) {
	l.Messages <- entry
}

func (l *TuiLogger) Errorf(format string, args ...interface{}) {
	entry := loggo.Entry{Level: loggo.ERROR, Message: fmt.Sprintf(format, args...)}
	l.Messages <- entry
}

func (l *TuiLogger) Warnf(format string, args ...interface{}) {
	entry := loggo.Entry{Level: loggo.WARNING, Message: fmt.Sprintf(format, args...)}
	l.Messages <- entry
}

func (l *TuiLogger) Infof(format string, args ...interface{}) {
	entry := loggo.Entry{Level: loggo.INFO, Message: fmt.Sprintf(format, args...)}
	l.Messages <- entry
}

func (l *TuiLogger) Debugf(format string, args ...interface{}) {
	entry := loggo.Entry{Level: loggo.DEBUG, Message: fmt.Sprintf(format, args...)}
	l.Messages <- entry
}

func (l *TuiLogger) Tracef(format string, args ...interface{}) {
	entry := loggo.Entry{Level: loggo.TRACE, Message: fmt.Sprintf(format, args...)}
	l.Messages <- entry
}

func (l *TuiLogger) Error(msg string) {
	entry := loggo.Entry{Level: loggo.ERROR, Message: msg}
	l.Messages <- entry
}

func (l *TuiLogger) Warn(msg string) {
	entry := loggo.Entry{Level: loggo.WARNING, Message: msg}
	l.Messages <- entry
}

func (l *TuiLogger) Info(msg string) {
	entry := loggo.Entry{Level: loggo.INFO, Message: msg}
	l.Messages <- entry
}

func (l *TuiLogger) Debug(msg string) {
	entry := loggo.Entry{Level: loggo.DEBUG, Message: msg}
	l.Messages <- entry
}

func (l *TuiLogger) Trace(msg string) {
	entry := loggo.Entry{Level: loggo.TRACE, Message: msg}
	l.Messages <- entry
}
