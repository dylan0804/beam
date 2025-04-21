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
		log.Println("error opening file", err)
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
		log.Println("error transmitting files", err)
		return err
	}
	
	return err
}

func DialAndSend(path string) error {
	receiver := make(chan Host)
	receiverErr := make(chan error)

	go discoverServer(receiver, receiverErr)

	go func(){
		for err := range receiverErr {
			fmt.Printf("error when discovering server: %v\n", err)
		}
	}()

	id := 1
	seen := map[string]bool{}
	var hosts []Host
	const prompt = "Which device to send to? "
	go func(){
		for r := range receiver {
			r.ID = id
			key := r.IP.String() + ":" + strconv.Itoa(r.Port)
			if seen[key] {
				continue
			}
			seen[key] = true
			fmt.Printf("\n%d. %s -- %s:%d\n%s", r.ID, r.Name, r.IP, r.Port, prompt)			
			hosts = append(hosts, r)
			id++
		}
	}()

	var idDest int
	var selectedHost Host
	
	for {
		_, err := fmt.Scanln(&idDest)
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
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	err = sendFile(conn, path)
	if err != nil {
		return err
	}

	return nil
}

func discoverServer(receiver chan<- Host, receiverError chan<- error) {
	listenAddr := &net.UDPAddr{
		IP: net.IPv4zero,
		Port: 9999,
	}
	sock, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		receiverError <- err
	}
	defer sock.Close()

	buf := make([]byte, 256)

	for {
		n, srcAddr, err := sock.ReadFromUDP(buf)
		if err != nil {
			receiverError <- err
			continue
		}

		hostname, p, _ := strings.Cut(string(buf[:n]), "|")

		port, err := strconv.Atoi(p)
		if err != nil {
			receiverError <- fmt.Errorf("error reading port %q: %v", port, err)
			continue
		}

		r := Host{
			Name: hostname,
			IP: srcAddr.IP,
			Port: port,
		}

		receiver <- r
	}
}
