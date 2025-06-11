package tcp

import (
	"net"
)

type Listener interface {
	Listen(addr string) (net.Listener, error)
}

type Listen struct {
}

func NewListener() Listener {
	return &Listen{}
}

func (l *Listen) Listen(addr string) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
