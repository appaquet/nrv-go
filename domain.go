package nrv

import ()

var (
	Domains map[string]*Domain = make(map[string]*Domain)
)

type Domain struct {
	bindings []*Binding
}

func GetDomain(name string) *Domain {
	domain, found := Domains[name]
	if !found {
		domain = &Domain{
			bindings: make([]*Binding, 0),
		}
		Domains[name] = domain
	}
	return domain
}

func (d *Domain) Bind(binding *Binding) {
	d.bindings = append(d.bindings, binding)
	binding.init()
}

func (d *Domain) FindBinding(path string) (*Binding, Map) {
	for _, binding := range d.bindings {
		if m := binding.Matches(path); m != nil {
			return binding, m
		}
	}

	return nil, nil
}

func (d *Domain) Get(path string, params ...interface{}) (*Request, Error) {
	return nil, nil
}

func (d *Domain) Post(path string, params ...interface{}) (*Request, Error) {
	return nil, nil
}

func (d *Domain) Put(path string, params ...interface{}) (*Request, Error) {
	return nil, nil
}

func (d *Domain) Delete(path string, params ...interface{}) (*Request, Error) {
	return nil, nil
}
