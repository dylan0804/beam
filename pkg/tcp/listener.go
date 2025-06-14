package tcp

import (
	"net"
)

type NetListener interface {
	Accept() (net.Conn, error)
	Close() error
	Addr() net.Addr
}

type Listener interface {
	Listen(addr string) (NetListener, error)
}

type Listen struct{}

func NewListener() *Listen {
	return &Listen{}
}

func (l *Listen) Listen(addr string) (NetListener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
