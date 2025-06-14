package tcp

import "net"

type Dialer func(network string, address string) (net.Conn, error)

type UDPDialer func(network string, laddr *net.UDPAddr, raddr *net.UDPAddr) (*net.UDPConn, error)
