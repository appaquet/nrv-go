package nrv

import (
	"fmt"
	"testing"
)

type tStruct struct {
	A int
	B string
}

func TestMapInto(t *testing.T) {
	m := NewMap()
	m["a"] = 23
	m["B"] = "salut"

	ts := &tStruct{}
	m.Into(ts)

	fmt.Printf("%s", ts)
	if ts.A != 23 {
		t.Fail()
	}
	if ts.B != "salut" {
		t.Fail()
	}

	// shouldn't panic
	m["a"] = "toto"
	m.Into(ts)
}
