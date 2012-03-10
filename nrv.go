package nrv

import (
	"fmt"
	"hash/crc32"
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


// Token assigned to a node in a service 
type Token uint32

func HashToken(data string) Token {
	if data == "" {
		return Token(0)
	}

	return Token(crc32.ChecksumIEEE([]byte(data)))
}
