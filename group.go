package nrv

import (
	"os"
)

var (
	Groups map[string]*Group
)

type Group struct {

}

func GetGroup(name string) *Group {
	return nil
}

func (group *Group) Register(path string, binding Binding) {
}

func (group *Group) Get(path string, params ...interface{}) (*Request, os.Error) {
	return nil, nil
}

func (group *Group) Post(path string, params ...interface{}) (*Request, os.Error) {
	return nil, nil
}

func (group *Group) Put(path string, params ...interface{}) (*Request, os.Error) {
	return nil, nil
}

func (group *Group) Delete(path string, params ...interface{}) (*Request, os.Error) {
	return nil, nil
}

type Request struct {
	OnReply   interface{}
}

func (r *Request) WaitReply() *Message {
	return nil
}
