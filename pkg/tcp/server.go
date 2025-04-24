package tcp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	netListen = net.Listen
	startBroadcastFn = startBroadcast
	handleConnectionFn = handleConnection
	osHostnameFn = os.Hostname

	logPrintln = log.Println
)

func formatBytes(b int64) string {
	const unit = 1000

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

	var lenBuf [4]byte

	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return err
	}

	filenameLength := binary.BigEndian.Uint32(lenBuf[:])

	nameBytes := make([]byte, filenameLength)
	if _, err := io.ReadFull(conn, nameBytes); err != nil {
		return err
	}

	filename := string(nameBytes)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	outPath := filepath.Join(homeDir, "Downloads", filename)
	
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	buf := make([]byte, 4096)
	var totalBytes int64
	for {
		n, err := conn.Read(buf)

		if err == io.EOF || err != nil {
			break
		}

		if _, err := outFile.Write(buf); err != nil {
			log.Println("error writing byte to file", err)
			break
		}

		totalBytes += int64(n)
	}

	fmt.Printf("\n✅ Transfer complete: %s received.\n", formatBytes(totalBytes))

	return nil
}

func startBroadcast(port int, hostname string) {
	fmt.Printf("Broadcasting %s on port %d", hostname, port)
	bcAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:9999")
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
		if _, err := conn.Write(msg); err != nil {
			log.Println("error when broadcasting port: ", err)
		}
		time.Sleep(1 * time.Second)
	}
}

func Listen(ctx context.Context, port int) error {
	listen, err := netListen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	
	go func(){
		<-ctx.Done()
		listen.Close()
	}()

	port = listen.Addr().(*net.TCPAddr).Port

	var hostname string
	hostname, err = osHostnameFn() 
	if err != nil {
		hostname = "unknown host"
	}

	go startBroadcastFn(port, hostname)

	errChan := make(chan error)
	var wg sync.WaitGroup

	go func(){
		for e := range errChan {
			logPrintln("error from handler:", e)
		}
	}()

	for {
		conn, err := listen.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			return err
		}
		
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			if err := handleConnectionFn(c); err != nil {
				errChan <- err
			}
		}(conn)
	}

	wg.Wait()
	close(errChan)

	return nil
}