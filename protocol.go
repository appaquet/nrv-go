package nrv

import (
	"net"
	"os"
	"io"
	"bufio"
	"bytes"
	"fmt"
	"http"
	"strings"
	"time"

	"github.com/appaquet/typedio"
)

const (
	MAX_UDP_SIZE = 1024
	HTTP_MAX_WAIT = 5000000000 // 5 secs
)

type Protocol interface {
	CallHandler
	Start(cluster Cluster)
}

type ProtocolNrv struct {
	LocalAddress string
	TCPPort      int
	UDPPort      int

	tcpSock *net.TCPListener
	udpSock *net.UDPConn
	cluster Cluster

	nextHandler     CallHandler
	previousHandler CallHandler
}

func (np *ProtocolNrv) Start(cluster Cluster) {
	np.cluster = cluster

	var err os.Error
	tcpAddr := net.TCPAddr{net.ParseIP(np.LocalAddress), int(np.TCPPort)}
	np.tcpSock, err = net.ListenTCP("tcp", &tcpAddr)
	if err != nil {
		log.Fatal("Can't start nrv tcp listener: %s", err)
	}
	go np.acceptTCP()

	udpAddr := net.UDPAddr{net.ParseIP(np.LocalAddress), int(np.UDPPort)}
	np.udpSock, err = net.ListenUDP("udp", &udpAddr)
	if err != nil {
		log.Fatal("Can't start nrv tcp listener: %s", err)
	}
	go np.acceptUDP()

	log.Info("ProtocolNrv> Started")
}

func (np *ProtocolNrv) acceptTCP() {
	for {
		conn, err := np.tcpSock.Accept()
		if err != nil {
			log.Error("Couldn't accept TCP connexion: %s\n", err)
		}

		// TODO: do something with conn
		_ = conn
		// go s.handleTCPConnection(conn)
	}
}

func (np *ProtocolNrv) acceptUDP() {
	// Looping for new messages
	for {
		buf := make([]byte, MAX_UDP_SIZE)
		n, adr, err := np.udpSock.ReadFrom(buf)

		log.Trace("ProtocolNrv> New UDP packet received of %d bytes from %s %s", n, adr, err)

		if err != nil {
			log.Error("ProtocolNrv> Error while reading UDP (read %d) from %s: %s\n", n, adr, err)
		} else {
			reader := io.Reader(bytes.NewBuffer(buf))
			tio := typedio.NewReader(reader)

			message, err := np.readMessage(tio)
			if err != nil {
				log.Error("ProtocolNrv> Got an error reading message %s", err)
			}

			domain := np.cluster.GetDomain(message.DomainName)
			binding, pathParams := domain.FindBinding(message.Path)

			if binding != nil {
				message.Params.Merge(pathParams)
				binding.HandleRequestReceive(&ReceivedRequest{
					Message: message,
				})
			} else {
				log.Error("ProtocolNrv> Got a message for a non existing path/domain %s %s", domain, message.Path)
			}


		}
	}
}

func (np *ProtocolNrv) getConnection(node *Node) *NrvConnection {
	log.Trace("Opening new UDP connection to %s", node)
	adr := net.UDPAddr{net.ParseIP(node.Address), int(node.UDPPort)}
	con, err := net.DialUDP("udp", nil, &adr) // TODO: should use local address instead of nil (implicitly local)
	if err != nil {
		log.Error("Couldn't create connection to node %s: %s", node, err)
		return nil
	}

	return &NrvConnection{con}
}


func (np *ProtocolNrv) SetNextHandler(handler CallHandler) {
	np.nextHandler = handler
}

func (np *ProtocolNrv) SetPreviousHandler(handler CallHandler) {
	np.previousHandler = handler
}

func (np *ProtocolNrv) HandleRequestSend(request *Request) *Request {
	log.Trace("ProtocolNrv> Sending request %s", request)

	for dest := range request.Message.Destination.Iter() {
		// TODO: bypass if dest == local
		conn := np.getConnection(dest.Node)
		buf := bufio.NewWriter(conn.conn)
		writer := typedio.NewWriter(buf)
		err := np.writeMessage(writer, request.Message)

		// TODO: handle errors by sending them back to the OnError callback
		if err != nil {
			log.Fatal("Couldn't write message to connection %s", err)
		}
		err = buf.Flush()
		if err != nil {
			log.Fatal("Got an error writing to connection: %s", err)
		}
		conn.Release()
	}

	log.Trace("ProtocolNrv> Sending request %s. Done!", request)
	return request
}

func (np *ProtocolNrv) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	log.Trace("ProtocolNrv> Received request %s", request)

	return request
}

func (np *ProtocolNrv) writeMessage(tio typedio.Writer, message *Message) (err os.Error) {
	if err = tio.WriteString(message.DomainName); err != nil {
		return
	}
	if err = tio.WriteString(message.Path); err != nil {
		return
	}
	if err = np.writeDomainMember(tio, message.Destination); err != nil {
		return
	}
	if err = tio.WriteUint32(message.DestinationRdv); err != nil {
		return
	}
	if err = np.writeDomainMember(tio, message.Source); err != nil {
		return
	}
	if err = tio.WriteUint32(message.SourceRdv); err != nil {
		return
	}
	if err = tio.WriteString(message.Error.Message); err != nil {
		return
	}
	if err = tio.WriteUint16(message.Error.Code); err != nil {
		return
	}
	if err = tio.WriteUint16(uint16(len(message.Params))); err != nil {
		return
	}

	for k, v := range message.Params {
		if err = tio.WriteString(k); err != nil {
			return
		}

		switch (v.(type)) {
		case string:
			if err = tio.WriteString("string"); err != nil {
				return
			}
			if err = tio.WriteString(v.(string)); err != nil {
				return
			}
		case int:
			if err = tio.WriteString("int"); err != nil {
				return
			}
			if err = tio.WriteInt64(int64(v.(int))); err != nil {
				return
			}
		case bool: 
			if err = tio.WriteString("bool"); err != nil {
				return
			}
			if err = tio.WriteBool(v.(bool)); err != nil {
				return
			}
		default:
			log.Fatal("Unsupported type for paramter %s: %s", k, v)
		}
	}

	return err
}

func (np *ProtocolNrv) writeDomainMember(tio typedio.Writer, domainMembers *DomainMembers) (err os.Error) {
	if err = tio.WriteUint8(uint8(domainMembers.Len())); err != nil {
		return
	}

	for domainMember := range domainMembers.Iter() {
		if err = tio.WriteUint32(uint32(domainMember.Token)); err != nil {
			return
		}
		if err = tio.WriteString(domainMember.Node.Address); err != nil {
			return
		}
		if err = tio.WriteUint16(uint16(domainMember.Node.TCPPort)); err != nil {
			return
		}
		if err = tio.WriteUint16(uint16(domainMember.Node.UDPPort)); err != nil {
			return
		}
	}
	return err
}

func (np *ProtocolNrv) readMessage(tio typedio.Reader) (message *Message, err os.Error) {
	message = &Message{}

	if message.DomainName, err = tio.ReadString(); err != nil {
		return
	}
	if message.Path, err = tio.ReadString(); err != nil {
		return
	}
	if message.Destination, err = np.readDomainMember(tio); err != nil {
		return
	}
	if message.DestinationRdv, err = tio.ReadUint32(); err != nil {
		return
	}
	if message.Source, err = np.readDomainMember(tio); err != nil {
		return
	}
	if message.SourceRdv, err = tio.ReadUint32(); err != nil {
		return
	}
	if message.Error.Message, err = tio.ReadString(); err != nil {
		return
	}
	if message.Error.Code, err = tio.ReadUint16(); err != nil {
		return
	}
	var nbParam uint16
	if nbParam, err = tio.ReadUint16(); err != nil {
		return
	}

	message.Params = NewMap()
	for i:=0; i < int(nbParam); i++ {

		var name string
		if name, err = tio.ReadString(); err != nil {
			return
		}

		var typ string
		if typ, err = tio.ReadString(); err != nil {
			return
		}

		var val interface{}
		switch (typ) {
		case "string":
			if val, err = tio.ReadString(); err != nil {
				return
			}
		case "int":
			if val, err = tio.ReadInt64(); err != nil {
				return
			}
		case "bool": 
			if val, err = tio.ReadBool(); err != nil {
				return
			}
		default:
			log.Fatal("Unsupported type for paramter %s: %s", name, typ)
		}

		message.Params[name] = val
	}

	return message, err
}

func (np *ProtocolNrv) readDomainMember(tio typedio.Reader) (domainMembers *DomainMembers, err os.Error) {
	domainMembers = NewDomainMembers()

	var count uint8
	if count, err = tio.ReadUint8(); err != nil {
		return
	}

	for i:=uint8(0); i < count; i++ {
		domainMember := DomainMember{Token(0), &Node{}}
		
		var token uint32
		if token, err = tio.ReadUint32(); err != nil {
			return
		}
		domainMember.Token = Token(token)

		if domainMember.Node.Address, err = tio.ReadString(); err != nil {
			return
		}

		var port uint16
		if port, err = tio.ReadUint16(); err != nil {
			return
		}
		domainMember.Node.TCPPort = int(port)

		if port, err = tio.ReadUint16(); err != nil {
			return
		}
		domainMember.Node.UDPPort = int(port)

		domainMembers.Add(domainMember)
	}

	return domainMembers, err
}


type NrvConnection struct {
	conn  net.Conn
}

func (c *NrvConnection) Release() {
	c.conn.Close()
}


type ProtocolHTTP struct {
	LocalAddress string
	Port         int

	server *http.Server
	cls    Cluster

	nextHandler     CallHandler
	previousHandler CallHandler
}

func (ph *ProtocolHTTP) Start(cls Cluster) {
	adr := fmt.Sprintf("%s:%d", ph.LocalAddress, ph.Port)
	ph.cls = cls

	ph.server = &http.Server{
		Addr: adr,
		Handler: ph,
		ReadTimeout:  5000000000, // 5 seconds
		WriteTimeout: 5000000000, // 5 seconds
	}

	go func() {
		err := ph.server.ListenAndServe()
		if err != nil {
			log.Fatal("Couldn't start HTTP protocol: %s", err)
		}
	}()

	log.Info("ProtocolHTTP> Started")
}

func (ph *ProtocolHTTP) ServeHTTP(respWriter http.ResponseWriter, req *http.Request) {
	log.Debug("ProtocolHTTP> Request received for %s %s", req.Host, req.RawURL)

	sp := strings.Split(req.Host, ":")
	d := ph.cls.GetDomain(sp[0])

	binding, params := d.FindBinding(req.RawURL)
	if binding != nil {
		responseWait := make(chan *Message, 1)
		binding.HandleRequestReceive(&ReceivedRequest{
			Message: &Message{
				Path:   req.RawURL,
				Params: params,
			},
			OnRespond: func(message *Message) {
				responseWait <- message
			},
		})

		select {
		case resp := <-responseWait:
			if !resp.Error.Empty() {
				http.Error(respWriter, resp.Error.Message, int(resp.Error.Code))
			} else {
				body := resp.Params["body"]
				respWriter.Write([]uint8(fmt.Sprintf("%s", body)))
			}

		case <-time.After(HTTP_MAX_WAIT):
			http.Error(respWriter, "Response timeout", http.StatusBadGateway)
		}
	} else {
		log.Debug("ProtocolHTTP> No binding found for %s %s", req.Host, req.RawURL)
		http.NotFound(respWriter, req)
	}
}

func (np *ProtocolHTTP) SetNextHandler(handler CallHandler) {
	np.nextHandler = handler
}

func (np *ProtocolHTTP) SetPreviousHandler(handler CallHandler) {
	np.previousHandler = handler
}

func (np *ProtocolHTTP) HandleRequestSend(request *Request) *Request {
	log.Trace("ProtocolHTTP> Sending request %s", request)
	return request
}

func (np *ProtocolHTTP) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	log.Trace("ProtocolHTTP> Received request %s", request)
	return request
}
