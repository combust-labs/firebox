package utils

type Defers interface {
	Add(f func())
	Exec()
}

func NewDefers() Defers {
	return &defers{fs: []func(){}}
}

type defers struct {
	fs []func()
}

func (d *defers) Add(f func()) {
	// add in reverse order:
	d.fs = append([]func(){f}, d.fs...)
}

func (d *defers) Exec() {
	for _, f := range d.fs {
		f()
	}
}
