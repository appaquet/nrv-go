package nrv

import (
	"fmt"
	"net/url"
)

type Cluster interface {
	Start()
	GetService(name string) *Service
	GetLocalNode() *Node
	GetBindingURL(bUrl *url.URL) (*Binding, Map)

	RegisterProtocol(protocol Protocol)
	GetDefaultProtocol() Protocol
}

type Nodes []*Node

type Node struct {
	Address string
	TCPPort int
	UDPPort int
}

func (n *Node) Is(other *Node) bool {
	if n.Address == other.Address && n.TCPPort == other.TCPPort && n.UDPPort == other.UDPPort {
		return true
	}

	return false
}

func (n *Node) String() string {
	return fmt.Sprintf("%s:%d:%d", n.Address, n.TCPPort, n.UDPPort)
}

type StaticCluster struct {
	localNode       *Node
	nodes           Nodes
	services        map[string]*Service
	protocols       []Protocol
	defaultProtocol Protocol
}

func NewStaticCluster(localNode *Node) *StaticCluster {
	c := &StaticCluster{
		localNode: localNode,
		services:  make(map[string]*Service),
	}

	nrvProto := &ProtocolNrv{
		LocalAddress: localNode.Address,
		TCPPort:      localNode.TCPPort,
		UDPPort:      localNode.UDPPort,
	}
	c.RegisterProtocol(nrvProto)
	c.defaultProtocol = nrvProto

	return c
}

func (c *StaticCluster) GetLocalNode() *Node {
	return c.localNode
}

func (c *StaticCluster) RegisterProtocol(protocol Protocol) {
	c.protocols = append(c.protocols, protocol)
	protocol.init(c)
}

func (c *StaticCluster) GetDefaultProtocol() Protocol {
	return c.defaultProtocol
}

func (c *StaticCluster) GetBindingURL(bUrl *url.URL) (*Binding, Map) {
	service := c.GetService(bUrl.Host)
	return service.FindBinding(bUrl.Path)
}

func (c *StaticCluster) Start() {
	for _, protocol := range c.protocols {
		protocol.start()
	}
}

func (c *StaticCluster) GetService(name string) *Service {
	service, found := c.services[name]
	if !found {
		service = newService(c)
		c.services[name] = service
		service.Name = name
	}
	return service
}
