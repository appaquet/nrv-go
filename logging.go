package nrv

import (
	golog "log"
	"fmt"
)

type Logger interface {
	Trace(msg string, v ...interface{})
	Debug(msg string, v ...interface{})
	Info(msg string, v ...interface{})
	Warning(msg string, v ...interface{})
	Error(msg string, v ...interface{})
	Fatal(msg string, v ...interface{})
}

type GoLogger struct {

}

func (gl GoLogger) Trace(msg string, v ...interface{}) {
	golog.Printf("TRACE > %s", fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Debug(msg string, v ...interface{}) {
	golog.Printf("DEBUG > %s", fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Info(msg string, v ...interface{}) {
	golog.Printf("INFO > %s", fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Warning(msg string, v ...interface{}) {
	golog.Printf("WARN > %s", fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Error(msg string, v ...interface{}) {
	golog.Printf("ERROR > %s", fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Fatal(msg string, v ...interface{}) {
	golog.Fatalf("FATAL > %s", fmt.Sprintf(msg, v...))
}
