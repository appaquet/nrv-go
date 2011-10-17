package main

import (
	"time"
	"fmt"
	"github.com/appaquet/nrv"
)

func main() {
	config := nrv.Config{
		DataPath: "data",
		Seeds: []nrv.Seed{
			nrv.Seed{"master.appaquet.com", 1234, 1235},
		},
		Protocols: []nrv.Protocol{
			&nrv.ProtocolNrv{Address: "127.0.0.1", TcpPort: 1234, UdpPort: 1235},
			&nrv.ProtocolHttp{Address: "127.0.0.1", Port: 8888},
		},
	}
	nrv.Initialize(config)

	d := nrv.GetDomain("localhost")
	d.Bind(&nrv.Binding{
		Path:       "/(home)/(.*)$",
		PathParams: []string{"test", "toto"},
		Pattern:    nrv.PatternRequestReply{},
		Endpoint:   nrv.EndpointOne{},
		Closure: func(msg *nrv.Message) *nrv.Message {
			fmt.Printf("%s", msg.Params)
			return nil
		},
	})
	nrv.Start()

	for {
		time.Sleep(1000000)
	}
}
