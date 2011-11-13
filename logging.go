package nrv

import (
	golog "log"
	"fmt"
	"time"
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
	first	int64
}

func NewLogger() Logger {
	return GoLogger{time.Nanoseconds()}
}

func (gl GoLogger) Trace(msg string, v ...interface{}) {
	golog.Printf("TRACE > %d %s", (time.Nanoseconds()-gl.first)/100000, fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Debug(msg string, v ...interface{}) {
	golog.Printf("DEBUG > %d %s", (time.Nanoseconds()-gl.first)/100000, fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Info(msg string, v ...interface{}) {
	golog.Printf("INFO > %d %s", (time.Nanoseconds()-gl.first)/100000, fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Warning(msg string, v ...interface{}) {
	golog.Printf("WARN > %d %s", (time.Nanoseconds()-gl.first)/100000, fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Error(msg string, v ...interface{}) {
	golog.Printf("ERROR > %d %s", (time.Nanoseconds()-gl.first)/100000, fmt.Sprintf(msg, v...))
}

func (gl GoLogger) Fatal(msg string, v ...interface{}) {
	golog.Fatalf("FATAL > %d %s", (gl.first-time.Nanoseconds())/100000, fmt.Sprintf(msg, v...))
}
