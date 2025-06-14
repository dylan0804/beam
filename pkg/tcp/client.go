package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/k0kubun/go-ansi"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/schollz/progressbar/v3"
)

type pathType int

const (
	file pathType = iota
	folder
	endOfStream
)

type Host struct {
	Name string
	IP   net.IP
	Port int
}

type Client struct {
	hostId int
	mutex  sync.Mutex
	hosts  map[int]Host
	seen   map[string]bool

	dialer           Dialer
	serverDiscoverer ServerDiscoverer
	in               io.Reader
	out              io.Writer
}

func NewClient(d Dialer, s ServerDiscoverer, in io.Reader, out io.Writer) *Client {
	return &Client{
		hostId: 1,
		hosts:  make(map[int]Host, 5),
		seen:   make(map[string]bool, 5),

		dialer:           d,
		serverDiscoverer: s,
		in:               in,
		out:              out,
	}
}

func (c *Client) sendDirectory(t pathType, writer *bufio.Writer, path string, files *int) error {
	err := writer.WriteByte(byte(t))
	if err != nil {
		return fmt.Errorf("error writing path type: %w", err)
	}

	err = binary.Write(writer, binary.BigEndian, uint32(len(path)))
	if err != nil {
		return fmt.Errorf("error writing folder length: %w", err)
	}

	_, err = writer.Write([]byte(path))
	if err != nil {
		return fmt.Errorf("error writing folder name: %w", err)
	}

	*files++

	return writer.Flush()
}

func (c *Client) sendFile(t pathType, writer *bufio.Writer, path string, f *os.File, files *int, bar *progressbar.ProgressBar) error {
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return fmt.Errorf("error reading file info: %w", err)
	}
	fileSize := fileInfo.Size()

	err = writer.WriteByte(byte(t))
	if err != nil {
		return fmt.Errorf("error writing path type: %w", err)
	}

	if err := binary.Write(writer, binary.BigEndian, uint32(len(path))); err != nil {
		return fmt.Errorf("error writing file length: %w", err)
	}
	if _, err := writer.Write([]byte(path)); err != nil {
		return fmt.Errorf("error writing filename: %w", err)
	}
	if err := binary.Write(writer, binary.BigEndian, uint64(fileSize)); err != nil {
		return fmt.Errorf("error writing file size: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("error flushing writer: %w", err)
	}

	*files++
	bar.Reset()
	bar.ChangeMax64(fileSize)
	bar.Describe(fmt.Sprintf("[%d] Sending: %s", *files, path))

	_, err = io.CopyN(writer, io.MultiReader(f, bar), fileSize)
	if err != nil {
		return fmt.Errorf("error writing file contents to writer: %w", err)
	}

	return writer.Flush()
}

func (c *Client) sendPayload(conn net.Conn, p string, pathType pathType) error {
	defer conn.Close()

	writer := bufio.NewWriter(conn)

	ignoreFilePath := filepath.Join(p, ".gitignore")
	ignorer, err := gitignore.CompileIgnoreFile(ignoreFilePath)
	if err != nil {
		ignorer = gitignore.CompileIgnoreLines("")
	}

	bar := progressbar.NewOptions64(1,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionThrottle(500*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionUseANSICodes(true),
	)
	defer bar.Close()

	var files int

	switch pathType {
	case file:
		f, err := os.Open(p)
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}

		if err := c.sendFile(file, writer, filepath.Base(p), f, &files, bar); err != nil {
			return err
		}

	case folder:
		base := filepath.Base(p)
		err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if ignorer.MatchesPath(path) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			relativePath, err := filepath.Rel(p, path)
			if err != nil {
				return err
			}

			relativePath = filepath.Join(base, relativePath)

			if d.IsDir() {
				return c.sendDirectory(folder, writer, relativePath, &files)
			} else {
				f, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("error opening file: %w", err)
				}
				return c.sendFile(file, writer, relativePath, f, &files, bar)
			}
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("✅ Transfer complete: %d files", files)
	return nil
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
	for id, host := range c.hosts {
		fmt.Fprintf(c.out, "\r%d. %s -- %s:%d\n", id, host.Name, host.IP, host.Port)
	}
	fmt.Fprint(c.out, "Choose a device to send to > ")
}

func (c *Client) DialAndSend(path string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	receiverChan, errChan := c.serverDiscoverer.Discover(ctx)

	go func() {
		for {
			select {
			case host := <-receiverChan:
				if c.addHost(host) {
					c.redrawUI()
				}
			case err := <-errChan:
				log.Printf("error occured: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	selectedId, err := c.getDeviceID()
	if err != nil {
		return fmt.Errorf("error getting device id: %w", err)
	}
	selectedHost := c.hosts[selectedId]
	conn, err := c.dialer("tcp", fmt.Sprintf("%s:%d", selectedHost.IP, selectedHost.Port))
	if err != nil {
		return fmt.Errorf("error dialing receiver's end: %w", err)
	}

	pathType, err := c.getPathType(path)
	if err != nil {
		return fmt.Errorf("error getting path type: %w", err)
	}

	err = c.sendPayload(conn, path, pathType)
	if err != nil {
		return fmt.Errorf("error sending payload: %w", err)
	}

	return nil
}

func (c *Client) getDeviceID() (int, error) {
	var selectedId int
	for {
		_, err := fmt.Fscanln(c.in, &selectedId)
		if err != nil {
			return 0, fmt.Errorf("error reading user input: %w", err)
		}

		c.mutex.Lock()
		if _, exists := c.hosts[selectedId]; !exists {
			fmt.Printf("Device with ID %d doesn't exist\n", selectedId)
			c.mutex.Unlock()
			continue
		}
		break
	}

	return selectedId, nil
}

func (c *Client) getPathType(path string) (pathType, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if fileInfo.IsDir() {
		return folder, nil
	} else {
		return file, nil
	}
}
