package nrv

const (
	M_ANY = iota
	M_GET
	M_POST
	M_PUT
	M_DELETE
)

type Pattern interface {
	CallHandler
}

type PatternRequestReply struct {
	Method int

	nextHandler     CallHandler
	previousHandler CallHandler
}

func (p *PatternRequestReply) SetNextHandler(handler CallHandler) {
	p.nextHandler = handler
}

func (p *PatternRequestReply) SetPreviousHandler(handler CallHandler) {
	p.previousHandler = handler
}

func (p *PatternRequestReply) HandleRequestSend(request *Request) *Request {
	return p.nextHandler.HandleRequestSend(request)
}

func (p *PatternRequestReply) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	return p.previousHandler.HandleRequestReceive(request)
}

/*
type PatternPublishSubscribe struct {

}

type PatternPushPull struct {

}*/
