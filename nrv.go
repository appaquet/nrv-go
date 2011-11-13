package nrv

import (
	"fmt"
)

var (
	log Logger = NewLogger()
)

type Error struct {
	Message string
	Code    uint16
}

func (e Error) String() string {
	return fmt.Sprintf("(%d) %s", e.Code, e.Message)
}

func (e Error) Empty() bool {
	return e.Message == "" && e.Code == 0
}
