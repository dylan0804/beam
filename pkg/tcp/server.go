package tcp

import (
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	DefaultBufferSize        = 48 * 1024 // 48KB
	DefaultBroadcastPort     = 9999
	DefaultBroadcastInterval = time.Second
)

type Server struct {
	listener Listener
	b        Broadcaster
	h        ConnectionHandler
}

func NewServer(listener Listener, b Broadcaster, h ConnectionHandler) *Server {
	return &Server{
		listener: listener,
		b:        b,
		h:        h,
	}
}

func (s *Server) Start(addr string) error {
	listener, err := s.listener.Listen(fmt.Sprintf(":%s", addr))
	if err != nil {
		return fmt.Errorf("error starting listener: %w", err)
	}
	defer listener.Close()

	hostname, port := getListenAddr(listener)

	// broadcast IP to clients
	go s.b.StartBroadcast(port, hostname)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			return err
		}

		go s.h.HandleConnection(conn)
	}

	return nil
}
