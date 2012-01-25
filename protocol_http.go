package nrv

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	HTTP_MAX_WAIT = 5000000000 // 5 secs
)

type ProtocolHTTP struct {
	LocalAddress   string
	Port           int
	DefaultService *Service

	server  *http.Server
	cluster Cluster
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
	Log.Debug("ProtocolHTTP> Request received for %s %s", req.Host, req.URL)

	sp := strings.Split(req.Host, ":")

	var service *Service = ph.DefaultService
	if ph.DefaultService == nil {
		service = ph.cluster.GetService(sp[0])
	}

	binding, params := service.FindBinding(req.URL.Path)
	if binding != nil {
		responseWait := make(chan *Message, 1)

		// parse url parameters + post parameters
		err := req.ParseForm()
		if err == nil {
			for k, v := range req.Form {
				params[k] = v
			}
		}
		params["method"] = req.Method

		// check if we need to trace this request // TODO: SECURITY!
		logLevel := Log.GetLevel()
		if _, found := params["nrv_trace"]; found {
			logLevel = 255
		}
		logger := &RequestLogger{
			Level: logLevel,
		}

		trc := logger.Trace("http_receive")
		binding.getFirstBackwardHandler().HandleRequestReceive(&ReceivedRequest{
			Message: &Message{
				Logger: logger,
				Path:   req.URL.Path,
				Data:   params,
			},
			OnReply: func(message *Message) {
				responseWait <- message
			},
		})

		select {
		case resp := <-responseWait:
			trc.End()

			if !resp.Error.Empty() {
				http.Error(respWriter, resp.Error.Message, int(resp.Error.Code))
			} else {
				if redirect_url, found := resp.Data["redirect_url"]; found {
					http.Redirect(respWriter, req, redirect_url.(string), 301)

				} else {
					// set content type
					contentType := "text/html"
					if newContentType, found := resp.Data["content-type"]; found {
						contentType = newContentType.(string)
					}
					respWriter.Header().Set("Content-Type", contentType)

					// body
					body := resp.Data["body"]
					strBody := fmt.Sprintf("%s", body)
					if _, found := params["nrv_trace"]; found {
						strBody += fmt.Sprintf("<pre style=\"font-size: 10px\">%s</pre>", logger)
					}
					respWriter.Write([]uint8(strBody))
				}
			}

		case <-time.After(HTTP_MAX_WAIT):
			Log.Debug("ProtocolHTTP> Response timeout!")
			http.Error(respWriter, "Response timeout", http.StatusBadGateway)
		}
	} else {
		Log.Debug("ProtocolHTTP> No binding found for %s %s", req.Host, req.URL)
		http.NotFound(respWriter, req)
	}
}

func (np *ProtocolHTTP) InitHandler(binding *Binding)           {}
func (np *ProtocolHTTP) SetNextHandler(handler CallHandler)     {}
func (np *ProtocolHTTP) SetPreviousHandler(handler CallHandler) {}

func (np *ProtocolHTTP) HandleRequestSend(request *Request) *Request {
	Log.Fatal("ProtocolHTTP> Sending request not yet supported in ProtocolHTTP")
	return request
}

func (np *ProtocolHTTP) HandleRequestReceive(request *ReceivedRequest) *ReceivedRequest {
	Log.Fatal("ProtocolHTTP> Unsupported handling of received request")
	return request
}
