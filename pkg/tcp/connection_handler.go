package tcp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

var DownloadPath = getDownloadPath()

type ConnectionHandler interface {
	HandleConnection(conn net.Conn)
}

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) HandleConnection(conn net.Conn) {
	defer conn.Close()

	handleError := func(err error, msg string) {
		fmt.Printf("Error for client %s: %s - %v", conn.RemoteAddr(), msg, err)
	}

	reader := bufio.NewReader(conn)

	bar := progressbar.NewOptions64(1,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionThrottle(500*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionUseANSICodes(true),
	)
	defer bar.Close()

	var filesProcessed int

	for {
		t, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			handleError(err, "error reading type")
			return
		}

		var fileNameLength uint32
		err = binary.Read(reader, binary.BigEndian, &fileNameLength)
		if err != nil {
			handleError(err, "error reading filename length")
			return
		}

		fileName := make([]byte, fileNameLength)
		_, err = io.ReadFull(reader, fileName)
		if err != nil {
			handleError(err, "error reading filename")
			return
		}

		path := filepath.Join(DownloadPath, string(fileName))

		filesProcessed++

		switch pathType(t) {
		case file:
			var contentLength uint64
			err = binary.Read(reader, binary.BigEndian, &contentLength)
			if err != nil {
				handleError(err, "error reading content length")
				return
			}

			file, err := os.Create(path)
			if err != nil {
				handleError(err, "error creating download path")
				return
			}

			// anon func to handle folders where EOF isn't clear
			err = func() error {
				defer file.Close()

				bar.Reset()
				bar.ChangeMax64(int64(contentLength))
				bar.Describe(fmt.Sprintf("[%d] Receiving: %s", filesProcessed, string(fileName)))

				_, err = io.CopyN(io.MultiWriter(file, bar), reader, int64(contentLength))
				if err != nil {
					os.RemoveAll(path)
					handleError(err, "failed to copy contents into file")
					return err
				}

				return nil
			}()

			if err != nil {
				return
			}

		case folder:
			err := os.MkdirAll(path, 0755)
			if err != nil {
				handleError(err, fmt.Sprintf("error creating dir %s", path))
				return
			}
		}
	}

	fmt.Printf("\n✅ Transfer complete: %d files", filesProcessed)
}
