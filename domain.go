package nrv

import (
	"fmt"
	"sort"
	"hash/crc32"
)

type Domain struct {
	Name	 string
	cluster  Cluster
	bindings []*Binding
	Members  *DomainMembers
}

func newDomain(cluster Cluster) *Domain {
	return &Domain{
		cluster:  cluster,
		bindings: make([]*Binding, 0),
		Members:  NewDomainMembers(),
	}
}

func (d *Domain) Bind(binding *Binding) *Binding {
	d.bindings = append(d.bindings, binding)
	binding.init(d, d.cluster)
	return binding
}

func (d *Domain) FindBinding(path string) (*Binding, Map) {
	for _, binding := range d.bindings {
		if m := binding.Matches(path); m != nil {
			return binding, m
		}
	}
	return nil, nil
}

func (d *Domain) CallChan(path string, request *Request) chan *ReceivedRequest {
	c := request.ReplyChan()
	d.Call(path, request)
	return c
}

func (d *Domain) Call(path string, request *Request) {
	b, _ := d.FindBinding(path)

	if b == nil {
		Log.Error("Domain> Cannot find binding for path %s", path)

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

type DomainMember struct {
	Token Token
	Node  *Node
}


type DomainMembers struct {
	Array []DomainMember
}

func NewDomainMembers(members... DomainMember) *DomainMembers {
	return &DomainMembers{members}
}

func (dm *DomainMembers) Iter() chan DomainMember {
	c := make(chan DomainMember)
	go func() {
		for _, m := range dm.Array {
			c <- m
		}
		close(c)
	}()
	return c
}

func (dm *DomainMembers) String() string {
	return fmt.Sprintf("%s", dm.Array)
}

func (dm *DomainMembers) Get(i int) DomainMember {
	return dm.Array[i]
}

func (dm *DomainMembers) Add(member DomainMember) {
	dm.Array = append(dm.Array, member)
	sort.Sort(dm)
}

func (dm *DomainMembers) Len() int {
	return len(dm.Array)
}

func (dm *DomainMembers) Empty() bool {
	if dm == nil || len(dm.Array) == 0 {
		return true
	}
	return false
}

func (dm *DomainMembers) Less(i, j int) bool {
	return dm.Array[i].Token < dm.Array[j].Token
}

func (dm *DomainMembers) Swap(i, j int) {
	dm.Array[i], dm.Array[j] = dm.Array[i], dm.Array[i]
}


type Resolver interface {
	CallHandler

	Resolve(domain *Domain, path string) *DomainMembers
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

func (r *ResolverOne) Resolve(domain *Domain, path string) *DomainMembers {
	pathToken := ResolveToken(path)
	ret := NewDomainMembers()

	var candidate *DomainMember
	for member := range domain.Members.Iter() {
		if member.Token <= pathToken && (candidate == nil || candidate.Token < member.Token) {
			candidate = &member
		}
	}

	if candidate == nil {
		if domain.Members.Empty() {
			ret.Add(domain.Members.Get(0))
		}
	} else {
		ret.Add(*candidate)
	}

	return ret
}

func (r *ResolverOne) HandleRequestSend(request *Request) *Request {
	if request.Message.IsDestinationEmpty() {
		request.Message.Destination = r.Resolve(request.Domain, request.Message.Path)
	}

	request.respNeeded = request.Message.Destination.Len()
	return r.nextHandler.HandleRequestSend(request)
}

func (r *ResolverOne) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	return r.previousHandler.HandleRequestReceive(request)
}
