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
	MAX_UDP_SIZE  = 1024
	HTTP_MAX_WAIT = 5000000000 // 5 secs
)

type ProtocolMarshaller interface {
	MarshallerName() string
	CanMarshal(obj interface{}) bool
	Marshal(obj interface{}) ([]byte, os.Error)
	Unmarshal(bytes []byte) (interface{}, os.Error)
}

type MarshalledObject struct {
	Name  string
	Bytes []byte
}

type Protocol interface {
	CallHandler
	AddMarshaller(marshaller ProtocolMarshaller)

	init(cluster Cluster)
	start()
}

type ProtocolNrv struct {
	LocalAddress string
	TCPPort      int
	UDPPort      int

	tcpSock     *net.TCPListener
	udpSock     *net.UDPConn
	cluster     Cluster
	marshallers map[string]ProtocolMarshaller

	nextHandler     CallHandler
	previousHandler CallHandler
}

func (np *ProtocolNrv) init(cluster Cluster) {
	np.cluster = cluster
	np.marshallers = make(map[string]ProtocolMarshaller)
	gob.Register(&MarshalledObject{})
}

func (np *ProtocolNrv) start() {
	var err os.Error
	tcpAddr := net.TCPAddr{net.ParseIP(np.LocalAddress), int(np.TCPPort)}
	np.tcpSock, err = net.ListenTCP("tcp", &tcpAddr)
	if err != nil {
		Log.Fatal("ProtocolNrv> Can't start nrv TCP listener: %s", err)
	}
	go np.acceptTCP()

	udpAddr := net.UDPAddr{net.ParseIP(np.LocalAddress), int(np.UDPPort)}
	np.udpSock, err = net.ListenUDP("udp", &udpAddr)
	if err != nil {
		Log.Fatal("ProtocolNrv> Can't start nrv UDP listener: %s", err)
	}
	go np.acceptUDP()

	Log.Info("ProtocolNrv> Started")
}

func (np *ProtocolNrv) AddMarshaller(marshaller ProtocolMarshaller) {
	np.marshallers[marshaller.MarshallerName()] = marshaller
}

func (np *ProtocolNrv) acceptTCP() {
	for {
		conn, err := np.tcpSock.Accept()
		if err != nil {
			Log.Error("ProtocolNrv> Couldn't accept TCP connexion: %s\n", err)
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

		Log.Trace("ProtocolNrv> New UDP packet received of %d bytes from %s %s", n, adr, err)

		if err != nil {
			Log.Error("ProtocolNrv> Error while reading UDP (read %d) from %s: %s\n", n, adr, err)
		} else {
			reader := io.Reader(bytes.NewBuffer(buf))
			message, err := np.readMessage(reader)
			if err != nil {
				Log.Error("ProtocolNrv> Got an error reading message %s", err)
			}

			service := np.cluster.GetService(message.ServiceName)
			binding, pathParams := service.FindBinding(message.Path)

			if binding != nil {
				message.Params.Merge(pathParams)
				binding.HandleRequestReceive(&ReceivedRequest{
					Message: message,
				})
			} else {
				Log.Error("ProtocolNrv> Got a message for a non existing. Service=%s Path=%s", service, message.Path)
			}

		}
	}
}

func (np *ProtocolNrv) getConnection(node *Node) *nrvConnection {
	Log.Trace("ProtocolNrv> Opening new UDP connection to %s", node)
	adr := net.UDPAddr{net.ParseIP(node.Address), int(node.UDPPort)}
	con, err := net.DialUDP("udp", nil, &adr) // TODO: should use local address instead of nil (implicitly local)
	if err != nil {
		Log.Error("Couldn't create connection to node %s: %s", node, err)
		return nil
	}

	return &nrvConnection{con}
}

func (np *ProtocolNrv) SetNextHandler(handler CallHandler) {
	np.nextHandler = handler
}

func (np *ProtocolNrv) SetPreviousHandler(handler CallHandler) {
	np.previousHandler = handler
}

func (np *ProtocolNrv) HandleRequestSend(request *Request) *Request {
	Log.Trace("ProtocolNrv> Sending request %s", request)

	for dest := range request.Message.Destination.Iter() {
		// TODO: bypass if dest == local
		conn := np.getConnection(dest.Node)
		buf := bufio.NewWriter(conn.conn)
		err := np.writeMessage(buf, request.Message)

		// TODO: handle errors by sending them back to the OnError callback
		if err != nil {
			Log.Fatal("ProtocolNrv> Couldn't write message to connection %s", err)
		}
		err = buf.Flush()
		if err != nil {
			Log.Fatal("ProtocolNrv> Got an error writing to connection: %s", err)
		}
		conn.Release()
	}

	Log.Trace("ProtocolNrv> Sending request %s. Done!", request)
	return request
}

func (np *ProtocolNrv) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	Log.Trace("ProtocolNrv> Received request %s", request)

	return request
}

func (np *ProtocolNrv) writeMessage(writer io.Writer, message *Message) os.Error {
	mParams, err := np.preMarshal(message.Params)
	if err != nil {
		return err
	}

	message.Params = mParams.(Map)

	encoder := gob.NewEncoder(writer)
	return encoder.Encode(message)
}

func (np *ProtocolNrv) preMarshal(obj interface{}) (newObj interface{}, err os.Error) {
	switch obj.(type) {
	case Map:
		mp := obj.(Map)
		for k, v := range mp {
			mp[k], err = np.preMarshal(v)
			if err != nil {
				return
			}
		}

	case []interface{}:
		ar := obj.([]interface{})
		for i, v := range ar {
			ar[i], err = np.preMarshal(v)
			if err != nil {
				return
			}
		}

	default:
		for marshName, marsh := range np.marshallers {
			if marsh.CanMarshal(obj) {
				bytes, err := marsh.Marshal(obj)
				if err != nil {
					return nil, err
				}
				return MarshalledObject{marshName, bytes}, nil
			}
		}
	}

	return obj, err
}

func (np *ProtocolNrv) postUnmarshal(obj interface{}) (newObj interface{}, err os.Error) {
	switch obj.(type) {
	case Map:
		mp := obj.(Map)
		for k, v := range mp {
			mp[k], err = np.postUnmarshal(v)
			if err != nil {
				return
			}
		}

	case []interface{}:
		ar := obj.([]interface{})
		for i, v := range ar {
			ar[i], err = np.postUnmarshal(v)
			if err != nil {
				return
			}
		}

	case *MarshalledObject:
		mObj := obj.(*MarshalledObject)
		marsh, found := np.marshallers[mObj.Name]
		if found {
			return marsh.Unmarshal(mObj.Bytes)
		} else {
			return nil, os.NewError(fmt.Sprintf("Cannot find marshaller named %s", mObj.Name))
		}

	}

	return obj, err
}

func (np *ProtocolNrv) readMessage(reader io.Reader) (message *Message, err os.Error) {
	decoder := gob.NewDecoder(reader)

	message = &Message{}
	err = decoder.Decode(message)
	if err != nil {
		return nil, err
	}

	var mParams interface{}
	mParams, err = np.postUnmarshal(message.Params)
	if err != nil {
		return nil, err
	}
	message.Params = mParams.(Map)

	return message, err
}

type nrvConnection struct {
	conn net.Conn
}

func (c *nrvConnection) Release() {
	c.conn.Close()
}

type ProtocolHTTP struct {
	LocalAddress       string
	Port               int
	DefaultService     *Service

	server     *http.Server
	cluster    Cluster

	nextHandler     CallHandler
	previousHandler CallHandler
}

func (ph *ProtocolHTTP) init(cluster Cluster) {
	ph.cluster = cluster
}

func (ph *ProtocolHTTP) start() {
	adr := fmt.Sprintf("%s:%d", ph.LocalAddress, ph.Port)

	ph.server = &http.Server{
		Addr:         adr,
		Handler:      ph,
		ReadTimeout:  5000000000, // 5 seconds
		WriteTimeout: 5000000000, // 5 seconds
	}

	go func() {
		err := ph.server.ListenAndServe()
		if err != nil {
			Log.Fatal("ProtocolHTTP> Couldn't start HTTP protocol: %s", err)
		}
	}()

	Log.Info("ProtocolHTTP> Started")
}

func (ph *ProtocolHTTP) AddMarshaller(marshaller ProtocolMarshaller) {
	panic("ProtocolHTTP doesn't support protocol marshaller yet")
}

func (ph *ProtocolHTTP) ServeHTTP(respWriter http.ResponseWriter, req *http.Request) {
	Log.Debug("ProtocolHTTP> Request received for %s %s", req.Host, req.RawURL)

	sp := strings.Split(req.Host, ":")

	var service *Service = ph.DefaultService
	if ph.DefaultService == nil {
		service = ph.cluster.GetService(sp[0])
	}

	binding, params := service.FindBinding(req.RawURL)
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
				// set content type
				contentType := "text/html"
				if newContentType, found := resp.Params["content-type"]; found {
					contentType = newContentType.(string)
				}
				respWriter.Header().Set("Content-Type", contentType)

				// body
				body := resp.Params["body"]
				respWriter.Write([]uint8(fmt.Sprintf("%s", body)))
			}

		case <-time.After(HTTP_MAX_WAIT):
			Log.Debug("ProtocolHTTP> Response timeout!")
			http.Error(respWriter, "Response timeout", http.StatusBadGateway)
		}
	} else {
		Log.Debug("ProtocolHTTP> No binding found for %s %s", req.Host, req.RawURL)
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
	Log.Trace("ProtocolHTTP> Sending request %s", request)
	return request
}

func (np *ProtocolHTTP) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	Log.Trace("ProtocolHTTP> Received request %s", request)
	return request
}
