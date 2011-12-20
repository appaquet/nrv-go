package nrv

import (
	"fmt"
	"sort"
	"hash/crc32"
)

type Service struct {
	Name             string
	DefaultProtocol  Protocol

	cluster  Cluster
	bindings []*Binding
	Members  *ServiceMembers
}

func newService(cluster Cluster) *Service {
	return &Service{
		cluster:  cluster,
		bindings: make([]*Binding, 0),
		Members:  NewServiceMembers(),
	}
}

func (s *Service) GetDefaultProtocol() Protocol {
	if s.DefaultProtocol != nil {
		return s.DefaultProtocol
	}

	return s.cluster.GetDefaultProtocol()
}

func (s *Service) Bind(binding *Binding) *Binding {
	s.bindings = append(s.bindings, binding)
	binding.init(s, s.cluster)
	return binding
}

func (s *Service) BindMethod(path string, controller interface{}, method string) *Binding {
	return s.Bind(&Binding{
		Path: path,
		Controller: controller,
		Method: method,
	})
}

func (s *Service) FindBinding(path string) (*Binding, Map) {
	for _, binding := range s.bindings {
		if m := binding.Matches(path); m != nil {
			return binding, m
		}
	}
	return nil, nil
}

func (s *Service) CallChan(path string, request *Request) chan *ReceivedRequest {
	c := request.ReplyChan()
	s.Call(path, request)
	return c
}

func (s *Service) Call(path string, request *Request) {
	b, _ := s.FindBinding(path)

	if b == nil {
		Log.Error("Service> Cannot find binding for path %s", path)

		if request.OnReply != nil {
			request.OnReply(&ReceivedRequest{
				Message: &Message{
					Error: Error{"Path not found", 404},
				},
			})
		}

	} else {
		request.Message.Path = path
		b.Call(request)
	}
}

type Token uint32

func ResolveToken(data string) Token {
	if data == "" {
		return Token(0)
	}

	return Token(crc32.ChecksumIEEE([]byte(data)))
}

type ServiceMember struct {
	Token Token
	Node  *Node
}

type ServiceMembers struct {
	Array []ServiceMember
}

func NewServiceMembers(members ...ServiceMember) *ServiceMembers {
	return &ServiceMembers{members}
}

func (sm *ServiceMembers) Iter() chan ServiceMember {
	c := make(chan ServiceMember)
	go func() {
		for _, m := range sm.Array {
			c <- m
		}
		close(c)
	}()
	return c
}

func (sm *ServiceMembers) String() string {
	return fmt.Sprintf("%s", sm.Array)
}

func (sm *ServiceMembers) Get(i int) ServiceMember {
	return sm.Array[i]
}

func (sm *ServiceMembers) Add(member ServiceMember) {
	sm.Array = append(sm.Array, member)
	sort.Sort(sm)
}

func (sm *ServiceMembers) Len() int {
	return len(sm.Array)
}

func (sm *ServiceMembers) Empty() bool {
	if sm == nil || len(sm.Array) == 0 {
		return true
	}
	return false
}

func (sm *ServiceMembers) Less(i, j int) bool {
	return sm.Array[i].Token < sm.Array[j].Token
}

func (sm *ServiceMembers) Swap(i, j int) {
	sm.Array[i], sm.Array[j] = sm.Array[i], sm.Array[i]
}

type Resolver interface {
	CallHandler

	Resolve(service *Service, path string) *ServiceMembers
}

type ResolverOne struct {
	nextHandler     CallHandler
	previousHandler CallHandler
}

func (r *ResolverOne) SetNextHandler(handler CallHandler) {
	r.nextHandler = handler
}

func (r *ResolverOne) SetPreviousHandler(handler CallHandler) {
	r.previousHandler = handler
}

func (r *ResolverOne) Resolve(service *Service, path string) *ServiceMembers {
	pathToken := ResolveToken(path)
	ret := NewServiceMembers()

	var candidate *ServiceMember
	for member := range service.Members.Iter() {
		if member.Token <= pathToken && (candidate == nil || candidate.Token < member.Token) {
			candidate = &member
		}
	}

	if candidate == nil {
		if service.Members.Empty() {
			ret.Add(service.Members.Get(0))
		}
	} else {
		ret.Add(*candidate)
	}

	return ret
}

func (r *ResolverOne) HandleRequestSend(request *Request) *Request {
	if request.Message.IsDestinationEmpty() {
		request.Message.Destination = r.Resolve(request.Service, request.Message.Path)
	}

	request.respNeeded = request.Message.Destination.Len()
	return r.nextHandler.HandleRequestSend(request)
}

func (r *ResolverOne) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	return r.previousHandler.HandleRequestReceive(request)
}
