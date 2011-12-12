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
	level   int
}

func NewLogger(level int) Logger {
	return GoLogger{time.Nanoseconds(), level}
}

func (gl GoLogger) Trace(msg string, v ...interface{}) {
	if gl.level >= 5 {
		golog.Printf("TRACE > %d %s", (time.Nanoseconds()-gl.first)/1000000, fmt.Sprintf(msg, v...))
	}
}

func (gl GoLogger) Debug(msg string, v ...interface{}) {
	if gl.level >= 4 {
		golog.Printf("DEBUG > %d %s", (time.Nanoseconds()-gl.first)/1000000, fmt.Sprintf(msg, v...))
	}
}

func (gl GoLogger) Info(msg string, v ...interface{}) {
	if gl.level >= 3 {
		golog.Printf("INFO > %d %s", (time.Nanoseconds()-gl.first)/1000000, fmt.Sprintf(msg, v...))
	}
}

func (gl GoLogger) Warning(msg string, v ...interface{}) {
	if gl.level >= 2 {
		golog.Printf("WARN > %d %s", (time.Nanoseconds()-gl.first)/1000000, fmt.Sprintf(msg, v...))
	}
}

func (gl GoLogger) Error(msg string, v ...interface{}) {
	if gl.level >= 1 {
		golog.Printf("ERROR > %d %s", (time.Nanoseconds()-gl.first)/1000000, fmt.Sprintf(msg, v...))
	}
}

func (gl GoLogger) Fatal(msg string, v ...interface{}) {
	golog.Fatalf("FATAL > %d %s", (gl.first-time.Nanoseconds())/1000000, fmt.Sprintf(msg, v...))
}
