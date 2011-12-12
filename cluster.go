package nrv

import (
	"fmt"
)

type Cluster interface {
	Start()
	GetDomain(name string) *Domain
	GetLocalNode() *Node

	RegisterProtocol(protocol Protocol)
	GetDefaultProtocol() Protocol
}

type Nodes []*Node

type Node struct {
	Address string
	TCPPort int
	UDPPort int
}

func (n *Node) String() string {
	return fmt.Sprintf("%s:%d:%d", n.Address, n.TCPPort, n.UDPPort)
}


type StaticCluster struct {
	localNode       *Node
	nodes           Nodes
	domains         map[string]*Domain
	protocols       []Protocol
	defaultProtocol Protocol
}

func NewStaticCluster(localNode *Node) *StaticCluster {
	c := &StaticCluster{
		localNode: localNode,
		domains:   make(map[string]*Domain),
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
}

func (c *StaticCluster) GetDefaultProtocol() Protocol {
	return c.defaultProtocol
}

func (c *StaticCluster) Start() {
	for _, protocol := range c.protocols {
		protocol.Start(c)
	}
}

func (c *StaticCluster) GetDomain(name string) *Domain {
	domain, found := c.domains[name]
	if !found {
		domain = newDomain(c)
		c.domains[name] = domain
		domain.Name = name
	}
	return domain
}
