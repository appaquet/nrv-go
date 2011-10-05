package nrv

import ()


type Config struct {
	DataPath       string
	Seeds          []Seed
	DefaultBinding Binding
}

type Seed struct {
	Address string
	TCPPort int
	UDPPort int
}

func Initialize(config Config) {
}

func Boot() {
}
