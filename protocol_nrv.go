package nrv

import (
	"bufio"
	"bytes"

	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	MAX_UDP_SIZE = 4096
)

type ProtocolMarshaller interface {
	MarshallerName() string
	CanMarshal(obj interface{}) bool
	Marshal(obj interface{}) ([]byte, error)
	Unmarshal(bytes []byte) (interface{}, error)
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
}

func (np *ProtocolNrv) init(cluster Cluster) {
	np.cluster = cluster
	np.marshallers = make(map[string]ProtocolMarshaller)
	gob.Register(&MarshalledObject{})
	gob.Register(&RequestLogger{})
}

func (np *ProtocolNrv) start() {
	var err error
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

		message, err := np.readMessage(conn)
		if err == nil {
			go np.handleReceivedMessage(message)
		} else {
			Log.Error("ProtocolNrv> Got an error reading TCP message %s", err)
		}

		// TODO: DON'T CLOSE IT, POOL IT
		conn.Close()
	}
}

func (np *ProtocolNrv) acceptUDP() {
	// Looping for new messages
	for {
		buf := make([]byte, MAX_UDP_SIZE)
		n, adr, err := np.udpSock.ReadFrom(buf)

		Log.Debug("ProtocolNrv> New UDP packet received of %d bytes from %s %s", n, adr, err)

		if err != nil {
			Log.Error("ProtocolNrv> Error while reading UDP (read %d) from %s: %s\n", n, adr, err)
		} else {
			reader := io.Reader(bytes.NewBuffer(buf))
			message, err := np.readMessage(reader)
			if err == nil {
				go np.handleReceivedMessage(message)
			} else {
				Log.Error("ProtocolNrv> Got an error reading UDP message %s", err)
			}
		}
	}
}

func (np *ProtocolNrv) handleReceivedMessage(message *Message) {
	service := np.cluster.GetService(message.ServiceName)
	binding, pathParams := service.FindBinding(message.Path)

	if binding != nil {
		message.Data.Merge(pathParams)
		binding.getFirstBackwardHandler().HandleRequestReceive(&ReceivedRequest{
			Message: message,
		})
	} else {
		Log.Error("ProtocolNrv> Got a message for a non existing. Service=%s Path=%s", service, message.Path)
	}
}

func (np *ProtocolNrv) getConnection(node *Node) *nrvConnection {
	// TODO: TCP pooling!
	// TODO: Find a way to find size of message

	/*
		Log.Debug("ProtocolNrv> Opening new UDP connection to %s", node)
		adr := net.UDPAddr{net.ParseIP(node.Address), int(node.UDPPort)}
		con, err := net.DialUDP("udp", nil, &adr) // TODO: should use local address instead of nil (implicitly local)
		if err != nil {
			Log.Error("Couldn't create UDP connection to node %s: %s", node, err)
			return nil
		}
		return &nrvConnection{con, false}
	*/

	Log.Debug("ProtocolNrv> Opening new TCP connection to %s", node)
	adr := net.TCPAddr{net.ParseIP(node.Address), int(node.TCPPort)}
	con, err := net.DialTCP("tcp", nil, &adr) // TODO: should use local address instead of nil (implicitly local)
	if err != nil {
		Log.Error("Couldn't create TCP connection to node %s: %s", node, err)
		return nil
	}
	return &nrvConnection{con, true}
}

func (np *ProtocolNrv) InitHandler(binding *Binding)           {}
func (np *ProtocolNrv) SetNextHandler(handler CallHandler)     {}
func (np *ProtocolNrv) SetPreviousHandler(handler CallHandler) {}

func (np *ProtocolNrv) HandleRequestSend(request *Request) *Request {
	Log.Debug("ProtocolNrv> Sending request %s", request)

	for _, dest := range request.Message.Destination.Slice {
		if dest.Node.Is(np.cluster.GetLocalNode()) {
			go np.handleReceivedMessage(request.Message)

		} else {
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
	}

	Log.Debug("ProtocolNrv> Sending request %s. Done!", request)
	return request
}

func (np *ProtocolNrv) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	Log.Fatal("ProtocolHTTP> Unsupported handling of received request")
	return request
}

func (np *ProtocolNrv) writeMessage(writer io.Writer, message *Message) error {
	mParams, err := np.preMarshal(message.Data)
	if err != nil {
		return err
	}

	message.Data = mParams.(Map)

	encoder := gob.NewEncoder(writer)
	return encoder.Encode(message)
}

func (np *ProtocolNrv) preMarshal(obj interface{}) (newObj interface{}, err error) {
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

func (np *ProtocolNrv) postUnmarshal(obj interface{}) (newObj interface{}, err error) {
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
			return nil, errors.New(fmt.Sprintf("Cannot find marshaller named %s", mObj.Name))
		}

	}

	return obj, err
}

func (np *ProtocolNrv) readMessage(reader io.Reader) (message *Message, err error) {
	decoder := gob.NewDecoder(reader)

	message = &Message{}
	err = decoder.Decode(message)
	if err != nil {
		return nil, err
	}

	var mParams interface{}
	mParams, err = np.postUnmarshal(message.Data)
	if err != nil {
		return nil, err
	}
	message.Data = mParams.(Map)

	return message, err
}

type nrvConnection struct {
	conn  net.Conn
	isTcp bool
}

func (c *nrvConnection) Release() {
	// TODO: if tcp, release to pool
	c.conn.Close()
}
