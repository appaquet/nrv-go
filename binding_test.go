package nrv

import (
	"testing"
)

func TestBindingPathReverse(t *testing.T) {
	b := Binding{Path: "/aaa/(.*)/ccc"}
	if b.GetPath("bbb") != "/aaa/bbb/ccc" {
		t.Fatalf("Path is not equal: %s", b.GetPath("bbb"))
	}
}
