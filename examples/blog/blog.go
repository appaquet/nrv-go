package main

import (
	"time"
	"fmt"
	"flag"
	"github.com/appaquet/nrv"
)

var t = fmt.Errorf

func main() {
	var node *int = flag.Int("node", 0, "0 = lb, 1 = front")
	flag.Parse()

	lbNode := &nrv.Node{Address: "127.0.0.1", TCPPort: 12001, UDPPort: 12002}
	frontNode := &nrv.Node{Address: "127.0.0.1", TCPPort: 12011, UDPPort: 12012}


	var localNode *nrv.Node
	if *node == 0 {
		localNode = lbNode
	} else {
		localNode = frontNode
	}

	cls := nrv.NewClusterStatic(localNode)

	var httpProto nrv.Protocol
	if *node == 0 {
		httpProto = &nrv.ProtocolHTTP{
			LocalAddress: "127.0.0.1",
			Port:         8080,
		}
		cls.RegisterProtocol(httpProto)
	}

	// web worker
	frontend := cls.GetDomain("frontend")
	frontend.Members.Add(nrv.DomainMember{0, frontNode})
	frontend.Bind(&nrv.Binding{
		Path:       "/(home)/(.*)$",
		PathParams: []string{"test", "toto"},
		Closure: func(req *nrv.ReceivedRequest) {
			req.Respond(&nrv.Message{ Params: nrv.Map{
				"body": "<b>AllO!</b>",
			}});
		},
	})

	// front line
	if *node == 0 {
		lb := cls.GetDomain("localhost")
		lb.Members.Add(nrv.DomainMember{0, lbNode})
		lb.Bind(&nrv.Binding{
			Path:     "(/.*)$",
			Pattern:  &nrv.PatternRequestReply{},
			Resolver: &nrv.ResolverOne{},
			Protocol: httpProto,
			Closure: func(req *nrv.ReceivedRequest) {
				resp := <-frontend.CallChan(req.Message.Path, &nrv.Request{
					Message: &nrv.Message{
						Params: req.Message.Params,
					},
				})

				req.Respond(resp.Message)
			},
		})
	}

	cls.Start()

	for {
		time.Sleep(1000000)
	}
}
