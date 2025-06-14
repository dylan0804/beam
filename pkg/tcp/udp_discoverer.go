package tcp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type ServerDiscoverer interface {
	Discover(ctx context.Context) (<-chan Host, <-chan error)
}

type UDPDiscoverer struct {
	ListenPort int

	receiverChan chan Host
	errChan      chan error
}

func NewUDPDiscoverer(listenPort int) *UDPDiscoverer {
	return &UDPDiscoverer{
		ListenPort: listenPort,

		receiverChan: make(chan Host, 5),
		errChan:      make(chan error, 5),
	}
}

func (u *UDPDiscoverer) Discover(ctx context.Context) (<-chan Host, <-chan error) {
	listenAddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: u.ListenPort,
	}

	sock, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		go func() {
			u.errChan <- err
			close(u.receiverChan)
			close(u.errChan)
		}()
		return u.receiverChan, u.errChan
	}

	go func() {
		defer close(u.receiverChan)
		defer close(u.errChan)
		defer sock.Close()

		buf := make([]byte, 1024)

		for {
			select {
			case <-ctx.Done():
				return

			default:
				// deadline here to prevent this goroutine from running perpetually if no server is running
				sock.SetReadDeadline(time.Now().Add(3 * time.Second))

				n, srcAddr, err := sock.ReadFromUDP(buf)
				if err != nil {
					if netError, ok := err.(net.Error); ok && netError.Timeout() {
						continue
					}
					u.errChan <- err
					continue
				}

				hostname, p, _ := strings.Cut(string(buf[:n]), "|")

				port, err := strconv.Atoi(p)
				if err != nil {
					u.errChan <- fmt.Errorf("error reading port %q: %w", port, err)
					continue
				}

				r := Host{
					Name: hostname,
					IP:   srcAddr.IP,
					Port: port,
				}

				u.receiverChan <- r
			}
		}
	}()

	return u.receiverChan, u.errChan
}
