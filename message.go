package nrv

import (
	"fmt"
)

type Map map[string]interface{}

func NewMap() Map {
	return make(map[string]interface{})
}

func (m Map) Merge(other Map) {
	for k, v := range other {
		m[k] = v
	}
}

type Message struct {
	Params Map

	Destination    *ServiceMembers
	DestinationRdv uint32

	Source    *ServiceMembers
	SourceRdv uint32

	ServiceName string
	Path        string

	Error  Error
}

func (m *Message) IsDestinationEmpty() bool {
	return m.Destination.Empty()
}

func (m *Message) String() string {
	return fmt.Sprintf("Dest=[%d %s], Src=[%d %s] %s %s", m.DestinationRdv, m.Destination, m.SourceRdv, m.Source, m.ServiceName, m.Path)
}

type Request struct {
	*Message

	OnReply   func(request *ReceivedRequest)
	WaitReply bool

	respReceived int
	respNeeded   int
	rdvSync      chan bool

	Service *Service
	Binding *Binding

	chanWait chan *ReceivedRequest
}

func (r *Request) init() {
	r.rdvSync = make(chan bool, 1)
}

func (r *Request) String() string {
	return fmt.Sprintf("%s", r.Message)
}

func (r *Request) NeedReply() bool {
	return (r.OnReply != nil || r.WaitReply)
}

func (r *Request) ReplyChan() chan *ReceivedRequest {
	r.WaitReply = true
	r.chanWait = make(chan *ReceivedRequest, 1)
	r.OnReply = func(request *ReceivedRequest) {
		r.chanWait <- request
	}
	return r.chanWait
}

type ReceivedRequest struct {
	*Message
	OnRespond func(msg *Message)
}

func (rq *ReceivedRequest) Respond(data Map) {
	rq.RespondMessage(&Message{Params: data})
}

func (rq *ReceivedRequest) RespondMessage(msg *Message) {
	if rq.OnRespond != nil {
		rq.OnRespond(msg)
	} else {
		Log.Fatal("No respond callback associated to received request")
	}
}
