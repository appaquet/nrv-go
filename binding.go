package nrv

import (
	"regexp"
	"fmt"
)

type Binding struct {
	Path        string
	PathParams  []string
	Operation   int
	Pattern     Pattern
	Endpoint    Endpoint
	Consensus   ConsensusManager
	Persistence PersistenceManager

	Controller interface{}
	Method     string
	Closure    func(msg *Message) *Message

	pathRe *regexp.Regexp
}

func (b *Binding) init() {
	b.pathRe = regexp.MustCompile("^" + b.Path)
	log.Trace(b.Path)
}

func (b *Binding) Matches(path string) Map {
	m := b.pathRe.FindSubmatch([]uint8(path))
	if len(m) > 0 {
		m = m[1:]
		ret := NewMap()
		for i, arParam := range m {
			var key string
			if i < len(b.PathParams) {
				key = b.PathParams[i]
			} else {
				key = fmt.Sprintf("%d", i)
			}

			ret[key] = string(arParam)
		}

		return ret
	}

	return nil
}

func (b *Binding) Call(msg *Message) *Message {
	if b.Closure != nil {
		return b.Closure(msg)
	}

	// TODO: support controller

	return nil
}
