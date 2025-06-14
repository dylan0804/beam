package tcp

import (
	"fmt"
	"log"
	"net"
	"time"
)

type Broadcaster interface {
	StartBroadcast(port int, hostname string)
}

type Broadcast struct{}

func NewBroadcaster() *Broadcast {
	return &Broadcast{}
}

func (b *Broadcast) StartBroadcast(port int, hostname string) {
	fmt.Printf("Broadcasting %s on port %d\n", hostname, port)
	bcAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", DefaultBroadcastPort))
	if err != nil {
		log.Fatal("error resolving UDP address:", err)
	}

	conn, err := net.DialUDP("udp4", nil, bcAddr)
	if err != nil {
		log.Fatal("error dialing UDP:", err)
	}
	defer conn.Close()

	msg := []byte(fmt.Sprintf("%s|%d", hostname, port))

	for {
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Printf("err sending msg: %v\n", err)
		}
		time.Sleep(DefaultBroadcastInterval)
	}
}
