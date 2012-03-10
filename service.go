package nrv

import (
	"fmt"
	"sort"
)

type Service struct {
	Name            string
	DefaultProtocol Protocol

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

func (s *Service) BindClosure(path string, closure func(request *ReceivedRequest)) *Binding {
	return s.Bind(&Binding{
		Path:    path,
		Closure: closure,
	})
}

func (s *Service) BindMethod(path string, controller interface{}, method string) *Binding {
	return s.Bind(&Binding{
		Path:       path,
		Controller: controller,
		Method:     method,
	})
}

func (s *Service) Reverse(controller interface{}, method string, params ...string) string {
	for _, binding := range s.bindings {
		if binding.MatchesMethod(controller, method) {
			return binding.GetPath(params...)
		}
	}

	return ""
}

func (s *Service) FindBinding(path string) (*Binding, Map) {
	for _, binding := range s.bindings {
		if m := binding.Matches(path); m != nil {
			Log.Debug("Found matching binding for %s: %s", path, binding)
			return binding, m
		}
	}
	return nil, nil
}

func (s *Service) CallWait(path string, reqBuild RequestBuilder) *ReceivedRequest {
	return <-s.CallChan(path, reqBuild)
}

func (s *Service) CallChan(path string, reqBuild RequestBuilder) chan *ReceivedRequest {
	request := reqBuild.ToRequest()
	c := request.ReplyChan()
	s.Call(path, request)
	return c
}

func (s *Service) Call(path string, reqBuild RequestBuilder) {
	request := reqBuild.ToRequest()
	b, _ := s.FindBinding(path)

	if b == nil {
		Log.Error("Service> Cannot find binding for path %s", path)

		// TODO: better handling
		if request.OnReply != nil {
			request.handleReply(&ReceivedRequest{
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

func (s *Service) Resolve(token Token, count int) *ServiceMembers {
	ret := NewServiceMembers()

	// TODO: support for count

	var candidate *ServiceMember
	for _, member := range s.Members.Slice {
		if member.Token <= token && (candidate == nil || candidate.Token < member.Token) {
			candidate = &member
		}
	}

	if candidate == nil {
		if s.Members.Empty() {
			ret.Add(s.Members.Get(0))
		}
	} else {
		ret.Add(*candidate)
	}

	return ret
}

// Member of a service ring
type ServiceMember struct {
	Token Token
	Node  *Node
}

type ServiceMembers struct {
	Slice []ServiceMember
}

func NewServiceMembers(members ...ServiceMember) *ServiceMembers {
	return &ServiceMembers{members}
}

func (sm *ServiceMembers) String() string {
	return fmt.Sprintf("%s", sm.Slice)
}

func (sm *ServiceMembers) Get(i int) ServiceMember {
	// FIXME: what if no node???
	return sm.Slice[i]
}

func (sm *ServiceMembers) Add(member ServiceMember) {
	sm.Slice = append(sm.Slice, member)
	sort.Sort(sm)
}

func (sm *ServiceMembers) Len() int {
	return len(sm.Slice)
}

func (sm *ServiceMembers) Empty() bool {
	if sm == nil || len(sm.Slice) == 0 {
		return true
	}
	return false
}

func (sm *ServiceMembers) Less(i, j int) bool {
	return sm.Slice[i].Token < sm.Slice[j].Token
}

func (sm *ServiceMembers) Swap(i, j int) {
	sm.Slice[i], sm.Slice[j] = sm.Slice[i], sm.Slice[i]
}
