package nrv

import (
	"net"
	"os"
)

type Protocol interface {
	Start(conf Config)
}

type ProtocolNrv struct {
	Address string
	TcpPort int
	UdpPort int

	tcpSock *net.TCPListener
	udpSock *net.UDPConn
}

func (np *ProtocolNrv) Start(conf Config) {
	tcpAddr := net.TCPAddr{net.ParseIP(np.Address), int(np.TcpPort)}

	var err os.Error
	np.tcpSock, err = net.ListenTCP("tcp", &tcpAddr)
	if err != nil {
		log.Fatal("Can't start nrv tcp listener: %s", err)
	}
	go np.acceptTcp()

	log.Info("ProtocolNrv> Started")
}

func (np *ProtocolNrv) acceptTcp() {
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
