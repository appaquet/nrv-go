package nrv

import (
	"reflect"
	"regexp"
	"fmt"
	"time"
)

type CallHandler interface {
	SetNextHandler(handler CallHandler)
	SetPreviousHandler(handler CallHandler)
	HandleRequestSend(request *Request) *Request
	HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest
}

type Binding struct {
	Path        string
	PathParams  []string
	Operation   int
	Pattern     Pattern
	Resolver    Resolver
	Consensus   ConsensusManager
	Persistence PersistenceManager
	Protocol    Protocol

	Timeout  int
	MaxRetry int

	Controller interface{}
	Method     string
	Closure    func(request *ReceivedRequest)

	rflMethod  *reflect.Method

	rdvs   map[uint32]*Request
	newRdv chan *Request
	getRdv chan *ReceivedRequest
	rdvId  chan uint32

	pathRe  *regexp.Regexp
	service *Service
	cluster Cluster
}

func (b *Binding) init(service *Service, cluster Cluster) {
	b.cluster = cluster
	b.service = service
	b.pathRe = regexp.MustCompile("^" + b.Path)
	b.newRdv = make(chan *Request, 1)
	b.getRdv = make(chan *ReceivedRequest, 1)
	b.rdvId = make(chan uint32, 100)
	b.rdvs = make(map[uint32]*Request)

	if b.Resolver == nil {
		b.Resolver = &ResolverOne{}
	}
	if b.Pattern == nil {
		b.Pattern = &PatternRequestReply{}
	}
	if b.Protocol == nil {
		b.Protocol = service.GetDefaultProtocol()
	}

	// controller
	if b.Controller != nil && b.Method != "" {
		typ := reflect.TypeOf(b.Controller)
		rMethod, found := typ.MethodByName(b.Method)

		// TODO: better error handling here
		if found {
			b.rflMethod = &rMethod
		} else {
			Log.Fatal("nrv> Couldn't find method in controller: %s.%s", typ, b.Method)
		}
	}

	b.Resolver.SetNextHandler(b.Pattern)
	b.Resolver.SetPreviousHandler(b)
	b.Pattern.SetNextHandler(b.Protocol)
	b.Pattern.SetPreviousHandler(b.Resolver)

	b.handleRendezVous()
}

func (b *Binding) handleRendezVous() {
	go func() {
		var rdvId uint32 = 0
		for {
			rdvId++
			b.rdvId <- rdvId
		}
	}()
	go func() {
		for {
			var req *Request
			var resp *ReceivedRequest

			select {
			case req = <-b.newRdv:
				b.rdvs[req.Message.SourceRdv] = req
				req.rdvSync <- true

			case resp = <-b.getRdv:
				if req, found := b.rdvs[resp.Message.DestinationRdv]; found {
					req.respReceived++
					if req.respNeeded >= req.respReceived {
						b.rdvs[resp.Message.DestinationRdv] = nil, false
					}

					go req.OnReply(resp)
				} else {
					Log.Error("Binding> Received a response for an unknown request: %s", resp)
				}

			case <-time.After(1000000000):
				// TODO: Handle timeouts!
			}

		}

	}()
}

func (b *Binding) SetNextHandler(handler CallHandler)     {}
func (b *Binding) SetPreviousHandler(handler CallHandler) {}

func (b *Binding) Matches(path string) Map {
	m := b.pathRe.FindSubmatch([]uint8(path))
	if len(m) > 0 {
		m = m[1:]
		ret := NewMap()
		for i, arParam := range m {
			var key string
			if i < len(b.PathParams) {
				key = b.PathParams[i]
			} else {
				key = fmt.Sprintf("%d", i)
			}

			ret[key] = string(arParam)
		}

		return ret
	}

	return nil
}

func (b *Binding) Call(request *Request) *Request {
	return b.HandleRequestSend(request)
}

func (b *Binding) HandleRequestSend(request *Request) *Request {
	request.init()

	Log.Trace("Binding> New request to send %s %s", request)
	request.Binding = b
	request.Service = b.service
	request.Message.Source = NewServiceMembers(ServiceMember{Token(0), b.cluster.GetLocalNode()})
	request.Message.ServiceName = b.service.Name

	if request.NeedReply() {
		request.Message.SourceRdv = <-b.rdvId
		b.newRdv <- request
		<-request.rdvSync
		Log.Trace("Binding> Request %s will wait for a reply!", request)
	}

	return b.Resolver.HandleRequestSend(request)
}

func (b *Binding) HandleProtocolRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	return b.Pattern.HandleRequestReceive(request)
}

func (b *Binding) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	Log.Trace("Binding> Received new request %s", request)

	// if there is a destination rdv, we call the rendez vous handler
	if request.Message.DestinationRdv > 0 {
		b.getRdv <- request
		return request
	}

	if request.OnRespond == nil {
		request.OnRespond = func(message *Message) {
			if request.Message.SourceRdv > 0 {
				if message.Path == "" {
					message.Path = request.Message.Path
				}
				message.Destination = request.Message.Source
				message.DestinationRdv = request.Message.SourceRdv
				b.Call(&Request{Message: message})
			} else {
				Log.Error("Binding> Cannot respond to a message with no rendez-vous id %s", message)
			}
		}
	}

	if b.Closure != nil {
		b.Closure(request)

	} else if b.rflMethod != nil {
		ctrlVal := reflect.ValueOf(b.Controller)
		reqVal := reflect.ValueOf(request)
		b.rflMethod.Func.Call([]reflect.Value{ctrlVal, reqVal})

	} else {
		Log.Error("Binding> No closure nor method set for binding %s", b)
	}
	return request
}
