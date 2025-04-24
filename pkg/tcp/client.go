package tcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	discoverServerFn = discoverServer
	sendFileFn = sendFile
	netDialFn = net.Dial
	scanLn = fmt.Scanln
	hostDetail string
	idDest int

	discoverErrorHandler = func(err error) {
		fmt.Printf("error when discovering server: %v", err)
	}
)

type Host struct {
	ID int
	Name string
	IP net.IP
	Port int
}

func sendFile(conn net.Conn, path string) error {
	defer conn.Close()
	
	file, err := os.Open(path)
	if err != nil {
		log.Println("error opening file: ", err)
		return err
	}
	defer file.Close()

	filename := filepath.Base(path)
	filenameBytes := []byte(filename)

	var lenBuf [4]byte

	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(filenameBytes)))

	if _, err := conn.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := conn.Write(filenameBytes); err != nil {
		return err
	}

	bytes := new(bytes.Buffer)
	io.Copy(bytes, file)

	if _, err := conn.Write(bytes.Bytes()); err != nil {
		log.Println("error transmitting files: ", err)
		return err
	}
	
	return err
}

func DialAndSend(path string) error {
	receiverChan := make(chan Host)
	errChan := make(chan error, 1)

	go discoverServerFn(receiverChan, errChan)

	go func() {
		for err := range errChan {
			discoverErrorHandler(err)
		}
	}()

	id := 1
	seen := map[string]bool{}
	var hosts []Host

	go func(){
		for r := range receiverChan {
			r.ID = id
			key := r.IP.String() + ":" + strconv.Itoa(r.Port)
			if seen[key] {
				continue
			}
			seen[key] = true
			hostDetail = fmt.Sprintf("%d. %s -- %s:%d\n", r.ID, r.Name, r.IP, r.Port)
			fmt.Println(hostDetail)	
			hosts = append(hosts, r)
			id++
		}
	}()

	// if len(hosts) == 0 { return nil }

	var selectedHost Host
	
	for {
		_, err := scanLn(&idDest)
		if err != nil {
			return err
		}	

		var found bool
		for _, h := range hosts {
			if h.ID == idDest {
				selectedHost = h
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("No server with ID %d. Try again.\n", idDest)
			continue
		}
		break
	}

	addr := fmt.Sprintf("%s:%d", selectedHost.IP, selectedHost.Port)
	conn, err := netDialFn("tcp", addr)
	if err != nil {
		return err
	}

	err = sendFileFn(conn, path)
	if err != nil {
		return err
	}

	return nil
}

func discoverServer(receiverChan chan<- Host, errChan chan<- error) {
	listenAddr := &net.UDPAddr{
		IP: net.IPv4zero,
		Port: 9999,
	}
	sock, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		errChan <- err
	}
	defer sock.Close()

	buf := make([]byte, 256)

	for {
		n, srcAddr, err := sock.ReadFromUDP(buf)
		if err != nil {
			errChan <- err
			continue
		}

		hostname, p, _ := strings.Cut(string(buf[:n]), "|")

		port, err := strconv.Atoi(p)
		if err != nil {
			errChan <- fmt.Errorf("error reading port %q: %v", port, err)
			continue
		}

		r := Host{
			Name: hostname,
			IP: srcAddr.IP,
			Port: port,
		}

		receiverChan <- r
	}
}
