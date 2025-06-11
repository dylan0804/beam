package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Host struct {
	Name string
	IP   net.IP
	Port int
}

type Client struct {
	hostId       int
	receiverChan chan Host
	errChan      chan error
	mutex        sync.Mutex
	hosts        map[int]Host
	seen         map[string]bool
}

func NewClient() *Client {
	return &Client{
		hostId:       1,
		receiverChan: make(chan Host, 5),
		errChan:      make(chan error, 5),
		hosts:        make(map[int]Host, 5),
		seen:         make(map[string]bool, 5),
	}
}

func (c *Client) sendPayload(conn net.Conn, path string) error {
	defer conn.Close()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(conn)

	fileName := fileInfo.Name()
	if err := binary.Write(writer, binary.BigEndian, uint32(len(fileName))); err != nil {
		return err
	}
	if _, err := writer.Write([]byte(fileName)); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, uint64(fileInfo.Size())); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	// manual streaming bcs for some reason io.Copy isnt working
	progressReader := &ProgressReader{
		total:     fileInfo.Size(),
		read:      0,
		startTime: time.Now(),
		writer:    writer,
		buf:       make([]byte, DefaultBufferSize),
		file:      file,
	}

	fmt.Printf("Sending '%s' (%d bytes)\n", fileName, progressReader.total)

	err = progressReader.Write()
	if err != nil {
		return fmt.Errorf("error writing file contents to writer: %v", err)
	}

	fmt.Printf("\nTransfer complete in %v!\n", time.Since(progressReader.startTime).Round(time.Second))

	return writer.Flush()
}

func (c *Client) addHost(host Host) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := fmt.Sprintf("%s:%d", host.IP, host.Port)
	if !c.seen[key] {
		c.seen[key] = true
		c.hosts[c.hostId] = host
		c.hostId++
		return true
	}

	return false
}

func (c *Client) redrawUI() {
	fmt.Print("\033[H\033[2J")

	for id, host := range c.hosts {
		fmt.Printf("%d. %s -- %s:%d\n", id, host.Name, host.IP, host.Port)
	}

	fmt.Printf("Choose a device to send to > ")
}

func (c *Client) DialAndSend(path string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.discoverServer(ctx)

	go func() {
		for {
			select {
			case host := <-c.receiverChan:
				if c.addHost(host) {
					c.redrawUI()
				}
			case err := <-c.errChan:
				log.Printf("error occured: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	var selectedId int
	for {
		_, err := fmt.Scanln(&selectedId)
		if err != nil {
			return fmt.Errorf("error reading user input: %v", err)
		}

		if _, exists := c.hosts[selectedId]; !exists {
			fmt.Printf("Device with ID %d doesn't exist\n", selectedId)
			continue
		}
		break
	}

	selectedHost := c.hosts[selectedId]

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", selectedHost.IP, selectedHost.Port))
	if err != nil {
		return fmt.Errorf("error dialing receiver's end: %v", err)
	}

	err = c.sendPayload(conn, path)
	if err != nil {
		return fmt.Errorf("error sending payload: %v", err)
	}

	return nil
}

func (c *Client) discoverServer(ctx context.Context) {
	listenAddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 9999,
	}
	sock, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		c.errChan <- err
		return
	}
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
				c.errChan <- err
				continue
			}

			hostname, p, _ := strings.Cut(string(buf[:n]), "|")

			port, err := strconv.Atoi(p)
			if err != nil {
				c.errChan <- fmt.Errorf("error reading port %q: %v", port, err)
				continue
			}

			r := Host{
				Name: hostname,
				IP:   srcAddr.IP,
				Port: port,
			}

			c.receiverChan <- r
		}
	}
}
