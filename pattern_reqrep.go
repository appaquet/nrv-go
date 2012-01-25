package nrv

import (
	"time"
)

type PatternRequestReply struct {
	binding *Binding

	nextHandler     CallHandler
	previousHandler CallHandler

	rdvs   map[uint32]*Request
	newRdv chan newRdv
	getRdv chan *getRdv
	rdvId  chan uint32
}

func (p *PatternRequestReply) InitHandler(binding *Binding) {
	p.binding = binding

	p.newRdv = make(chan newRdv, 1)
	p.getRdv = make(chan *getRdv, 1)
	p.rdvId = make(chan uint32, 100)
	p.rdvs = make(map[uint32]*Request)

	p.handleRendezVous()
}

func (p *PatternRequestReply) SetNextHandler(handler CallHandler) {
	p.nextHandler = handler
}

func (p *PatternRequestReply) SetPreviousHandler(handler CallHandler) {
	p.previousHandler = handler
}

func (p *PatternRequestReply) HandleRequestSend(request *Request) *Request {
	if request.NeedReply() {
		// get a new rendez-vous id
		request.Message.SourceRdv = <-p.rdvId

		// setup new rendez-vous
		rdv := newRdv{request, make(chan bool)}
		p.newRdv <- rdv
		<-rdv.sync

		Log.Debug("PatternReqRep> Request %s will wait for a reply!", request)
	}

	return p.nextHandler.HandleRequestSend(request)
}

func (p *PatternRequestReply) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	Log.Debug("HandleReqReply> Received new request %s", request)

	// if there is a destination rdv, it's a response! we set the initial request in it
	if request.Message.DestinationRdv > 0 {
		response := request
		rdv := &getRdv{response, nil, make(chan bool)}
		p.getRdv <- rdv
		<-rdv.sync

		response.InitRequest = rdv.request

	} else {
		// set the OnReply callback so that a call to Reply() works
		if request.OnReply == nil {
			request.OnReply = func(message *Message) {
				if request.Message.SourceRdv > 0 {
					if message.Path == "" {
						message.Path = request.Message.Path
					}
					message.Destination = request.Message.Source
					message.DestinationRdv = request.Message.SourceRdv
					p.binding.Call(&Request{
						InitRequest: request,
						Message:     message,
					})
				} else {
					Log.Error("HandleReqRep> Cannot respond to a message with no rendez-vous id %s", message)
				}
			}
		}
	}

	return p.previousHandler.HandleRequestReceive(request)
}

type newRdv struct {
	request *Request
	sync    chan bool
}

type getRdv struct {
	response *ReceivedRequest
	request  *Request
	sync     chan bool
}

func (p *PatternRequestReply) handleRendezVous() {
	go func() {
		var rdvId uint32 = 0
		for {
			rdvId++
			p.rdvId <- rdvId
		}
	}()
	go func() {
		for {
			select {
			case rdv := <-p.newRdv:
				p.rdvs[rdv.request.Message.SourceRdv] = rdv.request
				rdv.sync <- true

			case rdv := <-p.getRdv:
				resp := rdv.response
				if req, found := p.rdvs[resp.Message.DestinationRdv]; found {
					rdv.request = req
					req.respReceived++
					if req.respNeeded >= req.respReceived {
						delete(p.rdvs, resp.Message.DestinationRdv)
					}
				} else {
					Log.Error("PatternReqRep> Received a response for an unknown request: %s", resp)
				}
				rdv.sync <- true

			case <-time.After(1000000000):
				// TODO: Handle timeouts!
			}

		}

	}()
}

/*
type PatternPublishSubscribe struct {

}

type PatternPushPull struct {

}*/
