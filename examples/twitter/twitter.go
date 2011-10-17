package main

func main() {
}

/*
import (
	"os"
	"fmt"
	"github.com/appaquet/nrv"
)

func main() {
	config := nrv.Config{
		DataPath: "../data",
		Seeds: []nrv.Seed{
			nrv.Seed{"127.0.0.1", 1234, 1235},
		},
	}
	nrv.Initialize(config)

	d := nrv.GetDomain("frontend.pwitter.com")
	d.Bind("/home", &nrv.Binding{
		Pattern: nrv.PatternRequestReply{
			Method: nrv.M_GET,
		},
		Endpoint:    nrv.EndpointOne{},
		Consensus:   nrv.ConsensusPaxos{},
		Persistence: nrv.PersistenceLog{},
	})
	d.Bind("/user/:id:/timeline", &nrv.Binding{
		Pattern:  nrv.PatternPublishSubscribe{},
		Endpoint: nrv.EndpointAll{},
	})

	nrv.Start()
}

func Test1() {
	g := nrv.GetDomain("frontend.pwitter.com")

	req, err := g.Get("/home", "test")
	if err != nil {
		printErr(err)
		_ = req.WaitReply()
	}

	g.Get("/home", "test", nrv.Request{
		OnReply: func() {
		},
	})

	g.Get("/user/:id:/timeline", nrv.Request{
		Params: nrv.Map{
			"id": 10,
		},
		OnReply: func(msg *nrv.Message) {
			fmt.Printf("Received new tweet for user 10 %s", msg)
		},
	})

	g.Post("/user/10/timeline", "this is a tweet")
}

func printErr(err os.Error) {
}*/
