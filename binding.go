package nrv

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var paramReplaceRegexp = regexp.MustCompile("\\(.*?\\)")

type CallHandler interface {
	InitHandler(binding *Binding)
	SetNextHandler(handler CallHandler)
	SetPreviousHandler(handler CallHandler)

	HandleRequestSend(request *Request) *Request
	HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest
}

type Pattern interface {
	CallHandler
}

type Binding struct {
	cluster Cluster
	service *Service

	Path     string
	nbParams int
	pathRe   *regexp.Regexp

	RequestLogger *RequestLogger
	Pattern       Pattern
	Resolver      Resolver
	Consensus     ConsensusManager
	Persistence   PersistenceManager
	Protocol      Protocol

	Timeout  int
	MaxRetry int

	Controller interface{}
	Method     string
	Closure    func(request *ReceivedRequest)
	rflMethod  *reflect.Method
	ctrlType   reflect.Type
}

func (b *Binding) String() string {
	return fmt.Sprintf("[Binding %s%s]", b.service.Name, b.Path)
}

func (b *Binding) InitHandler(binding *Binding) {
}

func (b *Binding) init(service *Service, cluster Cluster) {
	b.cluster = cluster
	b.service = service
	b.pathRe = regexp.MustCompile("^" + b.Path)
	b.nbParams = len(paramReplaceRegexp.FindAll([]byte(b.Path), -1))

	if b.RequestLogger == nil {
		b.RequestLogger = &RequestLogger{}
	}
	if b.Resolver == nil {
		b.Resolver = &ResolverPath{Count: 1}
	}
	if b.Pattern == nil {
		b.Pattern = &PatternRequestReply{}
	}
	if b.Protocol == nil {
		b.Protocol = service.GetDefaultProtocol()
	}

	// controller
	if b.Controller != nil && b.Method != "" {
		b.ctrlType = reflect.TypeOf(b.Controller)
		rMethod, found := b.ctrlType.MethodByName(b.Method)

		if found {
			b.rflMethod = &rMethod
		} else {
			Log.Fatal("nrv> Couldn't find method in controller: %s.%s", b.ctrlType, b.Method)
		}
	}

	b.SetNextHandler(b.RequestLogger)
	b.RequestLogger.SetPreviousHandler(b)
	b.RequestLogger.SetNextHandler(b.Resolver)
	b.Resolver.SetPreviousHandler(b.RequestLogger)
	b.Resolver.SetNextHandler(b.Pattern)
	b.Pattern.SetPreviousHandler(b.Resolver)
	b.Pattern.SetNextHandler(b.Protocol)

	b.InitHandler(b)
	b.RequestLogger.InitHandler(b)
	b.Resolver.InitHandler(b)
	b.Pattern.InitHandler(b)
}

func (b *Binding) getFirstBackwardHandler() CallHandler {
	return b.Pattern
}

func (b *Binding) getFirstForwardHandler() CallHandler {
	return b.RequestLogger
}

func (b *Binding) SetNextHandler(handler CallHandler)     {}
func (b *Binding) SetPreviousHandler(handler CallHandler) {}

func (b *Binding) MatchesMethod(controller interface{}, method string) bool {
	thisCtrlType := reflect.TypeOf(controller)

	if b.ctrlType != nil && b.ctrlType.AssignableTo(thisCtrlType) && b.Method == method {
		return true
	}

	return false
}

func (b *Binding) GetPath(params ...string) string {
	i := 0

	path := paramReplaceRegexp.ReplaceAllStringFunc(b.Path, func(match string) string {
		ret := params[i]
		i++
		return ret
	})

	return strings.Trim(path, "^$|()")
}

func (b *Binding) Matches(path string) Map {
	m := b.pathRe.FindSubmatch([]uint8(path))
	if len(m) > 0 {
		m = m[1:]
		ret := NewMap()
		for i, arParam := range m {
			ret[strconv.Itoa(i)] = string(arParam)
		}

		return ret
	}

	return nil
}

func (b *Binding) Call(reqBuild RequestBuilder) *Request {
	request := reqBuild.ToRequest()
	return b.HandleRequestSend(request)
}

func (b *Binding) HandleRequestSend(request *Request) *Request {
	Log.Debug("%s> New request to send %s", b, request)

	if request.Path == "" {
		request.Path = b.Path
	}

	request.Binding = b
	request.Message.Source = NewServiceMembers(ServiceMember{Token(0), b.cluster.GetLocalNode()})
	request.Message.ServiceName = b.service.Name

	return b.getFirstForwardHandler().HandleRequestSend(request)
}

func (b *Binding) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	Log.Debug("%s> Received new request %s", b, request)

	// if this is a response to a reply, call the handle reply method
	if request.InitRequest != nil && request.InitRequest.NeedReply() {
		request.InitRequest.handleReply(request)

		// else, call the closure
	} else if b.Closure != nil {
		b.Closure(request)

		// else, call a method by reflection
	} else if b.rflMethod != nil {
		ctrlVal := reflect.ValueOf(b.Controller)
		reqVal := reflect.ValueOf(request)

		// TODO: pass params
		nbParams := b.rflMethod.Func.Type().NumIn()
		values := make([]reflect.Value, nbParams)
		values[0] = ctrlVal
		values[1] = reqVal

		for i := 2; i < nbParams; i++ {
			if val, found := request.Data[strconv.Itoa(i-2)]; found {
				values[i] = reflect.ValueOf(val)
			} else {
				Log.Warning("Didn't get all paramters to call %s", b)
				values[i] = reflect.ValueOf("")
			}
		}

		b.rflMethod.Func.Call(values)

	} else {
		Log.Fatal("%s> No closure nor method set", b)
	}

	return request
}
