package main

import (
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
