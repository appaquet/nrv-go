package nrv

type Binding struct {
	Operation   int
	Pattern     Pattern
	Endpoint    Endpoint
	Consensus   ConsensusManager
	Persistence PersistenceManager

	Controller interface{}
	Method     string
	Function   func(msg *Message, params ...interface{}) *Message
}
