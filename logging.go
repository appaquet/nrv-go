package nrv

import (
	"fmt"
	golog "log"
	"time"
)

type Logger interface {
	GetLevel() uint8
	SetLevel(level uint8)
	Trace(metric string) loggerTrace
	Debug(msg string, v ...interface{})
	Info(msg string, v ...interface{})
	Warning(msg string, v ...interface{})
	Error(msg string, v ...interface{})
	Fatal(msg string, v ...interface{})
}

type loggerTrace interface {
	End()
}

// helper to use defer 
func EndTrace(t loggerTrace) {
	t.End()
}

// Wrapper for Go logging system
type GoLogger struct {
	first time.Time
	level uint8
}

func NewLogger(level uint8) Logger {
	return &GoLogger{time.Now(), level}
}

func (gl *GoLogger) GetLevel() uint8 {
	return gl.level
}

func (gl *GoLogger) SetLevel(level uint8) {
	gl.level = level
}

func (gl *GoLogger) Trace(metric string) loggerTrace {
	context := &glTrace{gl, metric, time.Now()}
	if gl.level >= 10 {
		golog.Printf(fmt.Sprintf("TRACE START %s", metric))
	}
	return context
}

func (gl *GoLogger) Debug(msg string, v ...interface{}) {
	if gl.level >= 4 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		golog.Printf("%dms > DEBUG > %s", (time.Now().Sub(gl.first))/1000000, msg)
	}
}

func (gl *GoLogger) Info(msg string, v ...interface{}) {
	if gl.level >= 3 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		golog.Printf("%dms > INFO > %s", (time.Now().Sub(gl.first))/1000000, msg)
	}
}

func (gl *GoLogger) Warning(msg string, v ...interface{}) {
	if gl.level >= 2 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		golog.Printf("%dms > WARN > %s", (time.Now().Sub(gl.first))/1000000, msg)
	}
}

func (gl *GoLogger) Error(msg string, v ...interface{}) {
	if gl.level >= 1 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		golog.Printf("%dms > ERROR > %s", (time.Now().Sub(gl.first))/1000000, msg)
	}
}

func (gl *GoLogger) Fatal(msg string, v ...interface{}) {
	golog.Fatalf("%dms > FATAL > %s", (gl.first.Sub(time.Now()))/1000000, fmt.Sprintf(msg, v...))
}

type glTrace struct {
	logger Logger
	metric string
	start  time.Time
}

func (glt *glTrace) End() {
	glt.logger.Debug(fmt.Sprintf("TRACE END (%d ms)", glt.metric, (time.Now().Sub(glt.start))/1000000))
}

// Request logger
type RequestLogger struct {
	FirstLine *logLine
	Level     uint8

	curStackLine []*logLine
	lastPop      *logLine

	nextHandler CallHandler
	prevHandler CallHandler
}

func NewRequestLogger(level uint8) Logger {
	return &RequestLogger{Level: level}
}

func (rl *RequestLogger) InitHandler(binding *Binding) {
}

func (rl *RequestLogger) GetLevel() uint8 {
	return rl.Level
}

func (rl *RequestLogger) SetLevel(level uint8) {
	rl.Level = level
}

func (rl *RequestLogger) SetNextHandler(handler CallHandler) {
	rl.nextHandler = handler
}

func (rl *RequestLogger) SetPreviousHandler(handler CallHandler) {
	rl.prevHandler = handler
}

func (rl *RequestLogger) HandleRequestSend(request *Request) *Request {
	if request.Logger == nil {
		request.Logger = &RequestLogger{
			Level: Log.GetLevel(),
		}
	}

	// start a trace if it's not a reply
	if request.Message.DestinationRdv == 0 {
		request.sendTrace = request.Trace("req_send " + request.Binding.service.Name + ":" + request.Path)

	} else if request.InitRequest != nil {
		// it's a reply
		request.Logger = request.InitRequest.Logger

	}

	return rl.nextHandler.HandleRequestSend(request)
}

func (rl *RequestLogger) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	if request.Logger == nil {
		request.Logger = &RequestLogger{
			Level: Log.GetLevel(),
		}
	}

	// if we have received a response for a request, we end the tracing
	if request.InitRequest != nil && request.InitRequest.sendTrace != nil && request.InitRequest.Logger != request.Logger {
		sendTrace := request.InitRequest.sendTrace
		sendTrace.End()
		sendTrace.(*logLine).Attach(request.Logger.(*RequestLogger))
	}

	return rl.prevHandler.HandleRequestReceive(request)
}

func (c *RequestLogger) String() string {
	if c.FirstLine != nil {
		return c.FirstLine.stringDepth(0)
	}

	return ""
}

func (c *RequestLogger) addLine(line *logLine, newStack bool) {
	if line.Type <= c.Level {
		if c.curStackLine == nil {
			emptyLine := newLogLine(0, "")
			c.curStackLine = []*logLine{emptyLine}
			c.FirstLine = emptyLine
		}

		lastIndex := len(c.curStackLine) - 1
		if newStack {
			last := c.curStackLine[lastIndex]
			c.curStackLine = append(c.curStackLine, line)

			if last.Child == nil {
				last.Child = line
			} else {
				c.lastPop.Next = line
			}
		} else {
			last := c.curStackLine[lastIndex]
			last.Next = line
			c.curStackLine[lastIndex] = line
		}
	}
}

func (c *RequestLogger) stackPop() {
	// TODO: should unstack to the right line
	c.lastPop = c.curStackLine[len(c.curStackLine)-1]
	c.curStackLine = c.curStackLine[:len(c.curStackLine)-1]
}

func (c *RequestLogger) Trace(metric string) loggerTrace {
	l := newLogLine(10, metric)

	if c.Level >= 10 {
		Log.Debug("TRACE START %s", metric)
		l.endCb = func() {
			el := newLogLine(10, metric)
			el.ElapsTime = el.StartTime.Sub(l.StartTime)
			c.addLine(el, false)
			c.stackPop()
			Log.Debug("TRACE END %s (%d ms)", metric, el.ElapsTime/1000000)
		}

		c.addLine(l, true)
	} else {
		l.endCb = func() {}
	}

	return l
}

func (c *RequestLogger) Debug(msg string, v ...interface{}) {
	if c.Level >= 4 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		Log.Debug(msg)
		c.addLine(newLogLine(4, msg), false)
	}
}

func (c *RequestLogger) Info(msg string, v ...interface{}) {
	if c.Level >= 3 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		Log.Info(msg)
		c.addLine(newLogLine(3, msg), false)
	}
}

func (c *RequestLogger) Warning(msg string, v ...interface{}) {
	if c.Level >= 2 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		Log.Warning(msg)
		c.addLine(newLogLine(2, msg), false)
	}
}

func (c *RequestLogger) Error(msg string, v ...interface{}) {
	if c.Level >= 1 {
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}
		Log.Error(msg)
		c.addLine(newLogLine(1, msg), false)
	}
}

func (c *RequestLogger) Fatal(msg string, v ...interface{}) {
	Log.Fatal(fmt.Sprintf(msg, v...))
}

type logLine struct {
	Type uint8
	Msg  string

	StartTime time.Time
	ElapsTime time.Duration
	endCb     interface{}

	Next  *logLine
	Child *logLine
}

func newLogLine(typ uint8, msg string, v ...interface{}) *logLine {
	return &logLine{
		StartTime: time.Now(),
		Type:      typ,
		Msg:       msg,
	}
}

func (ll *logLine) End() {
	if cb, ok := ll.endCb.(func()); ok {
		cb()
		ll.endCb = nil
	}
}

func (ll *logLine) Attach(c *RequestLogger) {
	ll.Child = c.FirstLine
}

func (ll *logLine) typeString() string {
	switch ll.Type {
	case 0:
		return "EMPTY"
	case 1:
		return "ERROR"
	case 2:
		return "WARN"
	case 3:
		return "INFO"
	case 4:
		return "DEBUG"
	case 10:
		return "TRACE"
	}

	return "UNKNOWN"
}

func (ll *logLine) stringDepth(depth int) string {
	ret := ""
	if ll.Type <= 10 && ll.Type > 0 {
		for i := 0; i < depth; i++ {
			ret += " "
		}

		ret += ll.StartTime.Format("2006/01/02 15:04:05.000000") + " - "
		ret += ll.typeString() + " "

		if ll.Type == 10 {
			if ll.ElapsTime == 0 {
				ret += fmt.Sprintf("START %s\n", ll.Msg)
			} else {
				ret += fmt.Sprintf("END %s (%f ms)\n", ll.Msg, float64(ll.ElapsTime)/1000000)

			}
		} else {
			ret += ll.Msg + "\n"
		}
	}

	if ll.Child != nil {
		ret += ll.Child.stringDepth(depth + 1)
	}

	if ll.Next != nil {
		ret += ll.Next.stringDepth(depth)
	}

	return ret
}
