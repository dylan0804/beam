package tcp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/schollz/progressbar/v3"
)

const (
	DefaultBufferSize        = 64 * 1024 // 64KB
	DefaultBroadcastPort     = 9999
	DefaultBroadcastInterval = time.Second
)

type Server struct {
	listener Listener
}

func NewServer(listener Listener) *Server {
	return &Server{
		listener: listener,
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	handleError := func(err error, msg string) {
		log.Printf("Error for client %s: %s - %w", conn.RemoteAddr(), msg, err)
	}

	reader := bufio.NewReader(conn)

	var fileNameLength uint32
	err := binary.Read(reader, binary.BigEndian, &fileNameLength)
	if err != nil {
		handleError(err, "failed to read filename length")
		return
	}

	fileName := make([]byte, fileNameLength)
	_, err = io.ReadFull(reader, fileName)
	if err != nil {
		handleError(err, "failed to read filename")
		return
	}

	var contentLength uint64
	err = binary.Read(reader, binary.BigEndian, &contentLength)
	if err != nil {
		handleError(err, "failed to read content length")
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		handleError(err, "failed to get home dir")
		return
	}
	downloadPath := filepath.Join(homeDir, "Downloads", string(fileName))

	file, err := os.Create(downloadPath)
	if err != nil {
		handleError(err, "failed to create download path")
		return
	}
	defer file.Close()

	bar := progressbar.NewOptions(int(contentLength),
		progressbar.OptionSetDescription(fmt.Sprintf("Receiving '%s'", fileName)),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(25),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowBytes(true),
	)
	progressWriter := progressbar.NewReader(reader, bar)
	buf := make([]byte, DefaultBufferSize)
	_, err = io.CopyBuffer(file, &progressWriter, buf)
	if err != nil {
		os.Remove(downloadPath)
		handleError(err, "failed to copy contents into file")
		return
	}
}

func (s *Server) startBroadcast(port int, hostname string) {
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
			fmt.Printf("err sending msg: %w\n", err)
		}
		time.Sleep(DefaultBroadcastInterval)
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
	go s.startBroadcast(port, hostname)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			return err
		}

		go handleConnection(conn)
	}

	return nil
}
