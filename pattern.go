package nrv

const (
	M_ANY = iota
	M_GET
	M_POST
	M_PUT
	M_DELETE
)

type Pattern interface {
	/*HandleGet()
	HandlePost()
	HandleDelete()
	HandlePut()*/
}

type PatternRequestReply struct {
	Method int
}

type PatternPublishSubscribe struct {

}

type PatternPushPull struct {

}
