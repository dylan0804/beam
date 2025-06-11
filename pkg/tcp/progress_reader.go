package tcp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

type ProgressReader struct {
	writer    *bufio.Writer
	reader    *bufio.Reader
	total     int64
	read      int64
	file      *os.File
	startTime time.Time
	buf       []byte
}

func (pr *ProgressReader) Write() error {
	for {
		n, err := pr.file.Read(pr.buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}

		if _, err := pr.writer.Write(pr.buf[:n]); err != nil {
			return fmt.Errorf("error writing file to buffer: %v", err)
		}

		pr.read += int64(n)

		if pr.total%100000 == 0 || pr.read == pr.total {
			elapsed := time.Since(pr.startTime)
			percentage := float64(pr.read) / float64(pr.total) * 100
			speed := float64(pr.read) / elapsed.Seconds()

			var eta time.Duration
			if speed > 0 {
				remaining := pr.total - pr.read
				eta = time.Duration(float64(remaining)/speed) * time.Second
			}

			fmt.Print("\033[2K\033[G")
			fmt.Printf("%.1f%% (%s/%s) - %s/s - ETA: %s",
				percentage,
				formatBytes(pr.read),
				formatBytes(pr.total),
				formatBytes(int64(speed)),
				eta.Round(time.Second))
		}
	}

	return nil
}

func (pr *ProgressReader) Read() error {
	for {
		n, err := pr.reader.Read(pr.buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading buffer: %v", err)
		}

		if _, err := pr.file.Write(pr.buf[:n]); err != nil {
			return fmt.Errorf("error writing contents to writer: %v", err)
		}

		pr.read += int64(n)

		if pr.total%100000 == 0 || pr.read == pr.total {
			elapsed := time.Since(pr.startTime)
			percentage := float64(pr.read) / float64(pr.total) * 100
			speed := float64(pr.read) / elapsed.Seconds()

			var eta time.Duration
			if speed > 0 {
				remaining := pr.total - pr.read
				eta = time.Duration(float64(remaining)/speed) * time.Second
			}

			fmt.Print("\033[2K\033[G")
			fmt.Printf("%.1f%% (%s/%s) - %s/s - ETA: %s",
				percentage,
				formatBytes(pr.read),
				formatBytes(pr.total),
				formatBytes(int64(speed)),
				eta.Round(time.Second))
		}
	}

	return nil
}
