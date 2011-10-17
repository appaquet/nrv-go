package nrv

import (
	"fmt"
	"http"
	"strings"
)

type ProtocolHttp struct {
	Address string
	Port    int

	server *http.Server
}

func (ph *ProtocolHttp) Start(conf Config) {
	adr := fmt.Sprintf("%s:%d", ph.Address, ph.Port)
	ph.server = &http.Server{
		Addr:    adr,
		Handler: ph,
	}

	go func() {
		err := ph.server.ListenAndServe()
		if err != nil {
			log.Fatal("Couldn't start HTTP protocol: %s", err)
		}
	}()

	log.Info("ProtocolHttp> Started")
}

func (ph *ProtocolHttp) ServeHTTP(rq http.ResponseWriter, req *http.Request) {
	log.Debug("ProtocolHttp> Request received for %s %s", req.Host, req.RawURL)

	sp := strings.Split(req.Host, ":")
	d := GetDomain(sp[0])

	binding, params := d.FindBinding(req.RawURL)
	if binding != nil {
		// TODO: do something with return
		binding.Call(&Message{
			Params: params,
		})
	} else {
		log.Debug("ProtocolHttp> No binding found for %s %s", req.Host, req.RawURL)
		http.NotFound(rq, req)
	}
}
