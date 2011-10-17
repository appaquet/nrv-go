package nrv

type Message struct {
	Params Map
}

type Map map[string]interface{}

func NewMap() Map {
	return make(map[string]interface{})
}
func (m Map) Merge(other Map) {
	for k, v := range other {
		m[k] = v
	}

}

type Request struct {
	Params  Map
	OnReply interface{}
}

func (r *Request) WaitReply() *Message {
	return nil
}
