package nrv

import (
	"fmt"
)

var (
	Log Logger = NewLogger(0)
)

// Error with an error code
type Error struct {
	Message string
	Code    uint16
}

func (e Error) Error() string {
	return fmt.Sprintf("(%d) %s", e.Code, e.Message)
}

func (e Error) Empty() bool {
	return e.Message == "" && e.Code == 0
}
