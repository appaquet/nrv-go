package examples

import (
	"fmt"
	"github.com/appaquet/nrv"
)

func main() {
	config := nrv.Config{
		DataPath: "../data",
		Seeds: []nrv.Seed{
			nrv.Seed{"127.0.0.1", 1234, 1235},
		},
		DefaultBinding: nrv.Binding{},
	}
	nrv.Initialize(config)

	g := nrv.GetGroup("frontend")
	g.Register("/home", nrv.Binding{
		Pattern: nrv.PatternRequestReply{
			Method: nrv.M_GET,
		},
		Endpoint:    nrv.EndpointOne{},
		Consensus:   nrv.ConsensusPaxos{},
		Persistence: nrv.PersistenceLog{},
	})

	g.Register("/user/(.*)/timeline", nrv.Binding{
		Pattern:  nrv.PatternPublishSubscribe{},
		Endpoint: nrv.EndpointAll{},
	})

	nrv.Boot()
}

func Test1() {
	g := nrv.GetGroup("frontend")

	req, err := g.Get("/home", "test")
	if err != nil {
		_ = req.WaitReply()
	}

	g.Get("/home", "test", nrv.Request{
		OnReply: func() {
		},
	})

	g.Get("/user/10/timeline", nrv.Request{
		OnReply: func(msg *nrv.Message) {
			fmt.Printf("Received new tweet for user 10 %s", msg)
		},
	})

	g.Post("/user/10/timeline", "this is a tweet")
}
