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
	"gob"
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
		log.Fatal("ProtocolNrv> Can't start nrv TCP listener: %s", err)
	}
	go np.acceptTCP()

	udpAddr := net.UDPAddr{net.ParseIP(np.LocalAddress), int(np.UDPPort)}
	np.udpSock, err = net.ListenUDP("udp", &udpAddr)
	if err != nil {
		log.Fatal("ProtocolNrv> Can't start nrv UDP listener: %s", err)
	}
	go np.acceptUDP()

	log.Info("ProtocolNrv> Started")
}

func (np *ProtocolNrv) acceptTCP() {
	for {
		conn, err := np.tcpSock.Accept()
		if err != nil {
			log.Error("ProtocolNrv> Couldn't accept TCP connexion: %s\n", err)
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
			message, err := np.readMessage(reader)
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
	log.Trace("ProtocolNrv> Opening new UDP connection to %s", node)
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
		err := np.writeMessage(buf, request.Message)

		// TODO: handle errors by sending them back to the OnError callback
		if err != nil {
			log.Fatal("ProtocolNrv> Couldn't write message to connection %s", err)
		}
		err = buf.Flush()
		if err != nil {
			log.Fatal("ProtocolNrv> Got an error writing to connection: %s", err)
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

func (np *ProtocolNrv) writeMessage(writer io.Writer, message *Message) os.Error {
	encoder := gob.NewEncoder(writer)
	return encoder.Encode(message)
}

func (np *ProtocolNrv) readMessage(reader io.Reader) (message *Message, err os.Error) {
	decoder := gob.NewDecoder(reader)
	message = &Message{}
	err = decoder.Decode(message)
	return message, err
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
			log.Fatal("ProtocolHTTP> Couldn't start HTTP protocol: %s", err)
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
			log.Debug("ProtocolHTTP> Response timeout!")
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
