package nrv

import (
	"os"
)

var (
	log       Logger = GoLogger{}
	protocols []Protocol
	config    Config
)

type Config struct {
	DataPath  string
	Seeds     []Seed
	Protocols []Protocol
}

type Seed struct {
	Address string
	TCPPort int
	UDPPort int
}

func Initialize(config Config) {
	config = config
	protocols = config.Protocols
}

func Start() {
	for _, protocol := range protocols {
		protocol.Start(config)
	}
}

type Error interface {
	os.Error
}
