package nrv

// Service resolver that resolve from a path to a member of the ring
type Resolver interface {
	CallHandler
}


// Use full path to resolve token
type ResolverPath struct {
	Count		int
	binding         *Binding
	nextHandler     CallHandler
	previousHandler CallHandler
}

func (r *ResolverPath) InitHandler(binding *Binding) {
	r.binding = binding
}

func (r *ResolverPath) SetNextHandler(handler CallHandler) {
	r.nextHandler = handler
}

func (r *ResolverPath) SetPreviousHandler(handler CallHandler) {
	r.previousHandler = handler
}

func (r *ResolverPath) HandleRequestSend(request *Request) *Request {
	if request.Message.IsDestinationEmpty() {
		request.Message.Destination = r.binding.service.Resolve(HashToken(request.Message.Path), r.Count)
	}

	request.respNeeded = request.Message.Destination.Len()
	return r.nextHandler.HandleRequestSend(request)
}

func (r *ResolverPath) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	return r.previousHandler.HandleRequestReceive(request)
}

// Use first parameter to resolve token (/something/(<PARAM>)/...)
type ResolverParam struct {
	Count		int
	binding         *Binding
	nextHandler     CallHandler
	previousHandler CallHandler
}

func (r *ResolverParam) InitHandler(binding *Binding) {
	r.binding = binding
}

func (r *ResolverParam) SetNextHandler(handler CallHandler) {
	r.nextHandler = handler
}

func (r *ResolverParam) SetPreviousHandler(handler CallHandler) {
	r.previousHandler = handler
}

func (r *ResolverParam) HandleRequestSend(request *Request) *Request {
	if request.Message.IsDestinationEmpty() {
		data := r.binding.Matches(request.Message.Path)

		var token Token = Token(0)
		if param, found := data["0"]; found {
			token = HashToken(param.(string))
		}

		request.Message.Destination = r.binding.service.Resolve(token, r.Count)
	}

	request.respNeeded = request.Message.Destination.Len()
	return r.nextHandler.HandleRequestSend(request)
}

func (r *ResolverParam) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	return r.previousHandler.HandleRequestReceive(request)
}
