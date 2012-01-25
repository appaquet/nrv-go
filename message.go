package nrv

import (
	"fmt"
	"reflect"
	"strings"
)

type RequestBuilder interface {
	ToRequest() *Request
}

type Request struct {
	*Message
	Binding *Binding

	// used by logging to trace sent request 	
	sendTrace loggerTrace

	// Response variables
	InitRequest *ReceivedRequest
	OnReply     func(request *ReceivedRequest)
	WaitReply   bool

	chanWait     chan *ReceivedRequest
	respReceived int
	respNeeded   int
}

func (r *Request) handleReply(request *ReceivedRequest) {
	r.OnReply(request)
}

func (r *Request) ToRequest() *Request {
	return r
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

	InitRequest *Request

	OnReply func(msg *Message)
}

func (rq *ReceivedRequest) Reply(data Map) {
	rq.ReplyMessage(&Message{Data: data})
}

func (rq *ReceivedRequest) ReplyMessage(msg *Message) {
	if rq.OnReply != nil {
		rq.OnReply(msg)
	} else {
		Log.Fatal("No 'OnReply' callback associated to received request")
	}
}

type Message struct {
	Logger

	ServiceName string
	Path        string

	Destination    *ServiceMembers
	DestinationRdv uint32
	Source         *ServiceMembers
	SourceRdv      uint32

	Data  Map
	Error Error
}

func (m *Message) IsDestinationEmpty() bool {
	return m.Destination.Empty()
}

func (m *Message) ToRequest() *Request {
	return &Request{
		Message: m,
	}
}

func (m *Message) String() string {
	return fmt.Sprintf("Dest=[%d %s], Src=[%d %s] %s %s", m.DestinationRdv, m.Destination, m.SourceRdv, m.Source, m.ServiceName, m.Path)
}

type Map map[string]interface{}

func NewMap() Map {
	return make(map[string]interface{})
}

func (m Map) Merge(other Map) {
	for k, v := range other {
		m[k] = v
	}
}

// TODO: return error if any error, but continue (best effort)
func (m Map) Into(dest interface{}) {
	rflDestPtr := reflect.ValueOf(dest)

	// get pointed
	rflDest := rflDestPtr.Elem()
	destTyp := rflDest.Type()

	for i := 0; i < destTyp.NumField(); i++ {
		structField := destTyp.Field(i)
		fieldVal := rflDest.Field(i)

		if fieldVal.CanSet() {
			// TODO: support for recursive structures

			if val, found := m[structField.Name]; found {
				rflVal := reflect.ValueOf(val)
				if structField.Type.AssignableTo(rflVal.Type()) {
					fieldVal.Set(rflVal)
				}

			} else if val, found := m[strings.ToLower(structField.Name)]; found {
				rflVal := reflect.ValueOf(val)
				if structField.Type.AssignableTo(rflVal.Type()) {
					fieldVal.Set(rflVal)
				}
			}
		}

	}
}

type Array []interface{}

func NewArray(vals ...interface{}) Array {
	return Array(vals)
}

func NewArraySize(size int) Array {
	return Array(make([]interface{}, size))
}

func (a Array) IntoStructSlice(slicePtr interface{}, structure interface{}) {
	rflStruct := reflect.ValueOf(structure)
	rflSlicePtr := reflect.ValueOf(slicePtr)
	rflSlice := rflSlicePtr.Elem()

	structType := rflStruct.Type()
	structIsPtr := false
	if rflStruct.Kind() == reflect.Ptr {
		structIsPtr = true
		structType = rflStruct.Elem().Type()
	}

	for _, val := range a {
		newElm := reflect.New(structType)

		if mapVal, ok := val.(Map); ok {
			mapVal.Into(newElm.Interface())

			if !structIsPtr {
				newElm = newElm.Elem()
			}
		} else {
			newElm.Set(reflect.ValueOf(val))
		}

		rflSlice.Set(reflect.Append(rflSlice, newElm))
	}
}
