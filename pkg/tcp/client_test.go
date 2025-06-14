package tcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockDiscoverer struct {
	receiverChan chan Host
	errChan      chan error
}

func newMockDiscoverer() *mockDiscoverer {
	return &mockDiscoverer{
		receiverChan: make(chan Host, 1),
		errChan:      make(chan error, 1),
	}
}

func (m *mockDiscoverer) Discover(ctx context.Context) (<-chan Host, <-chan error) {
	go func() {
		m.receiverChan <- Host{
			Name: "fake-address",
			IP:   net.ParseIP("127.0.0.1"),
			Port: 3000,
		}
		<-ctx.Done()
	}()

	return m.receiverChan, m.errChan
}

func setupTest(t *testing.T) func() {
	file, err := os.Create("test.txt")
	assert.NoError(t, err)

	return func() {
		file.Close()
		os.Remove("test.txt")
	}
}

func TestDialAndSend(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	go func() {
		// simulate reading - blocks until serverConn receives data
		io.Copy(io.Discard, serverConn)
		serverConn.Close()
	}()

	mockDialer := func(network, address string) (net.Conn, error) {
		if address != "127.0.0.1:8080" {
			assert.NotEqual(t, "127.0.0.1:8080", address)
		}
		return clientConn, nil
	}

	inputReader, inputWriter := io.Pipe()
	var output bytes.Buffer
	mockDiscoverer := newMockDiscoverer()
	client := NewClient(mockDialer, mockDiscoverer, inputReader, &output)

	clientErrChan := make(chan error)
	go func() {
		clientErrChan <- client.DialAndSend("test.txt")
	}()

	prompt := "Choose a device to send to > "
	deadline := time.Now().Add(2 * time.Second)
	for {
		if strings.Contains(output.String(), prompt) {
			break
		}
		if time.Now().After(deadline) {
			assert.Fail(t, fmt.Sprintf("timed out waiting for UI prompt.\nOutput so far:\n%s", output.String()))
		}
		time.Sleep(10 * time.Millisecond)
	}

	defer inputWriter.Close()
	_, err := fmt.Fprintln(inputWriter, "1")
	assert.NoError(t, err)

	select {
	case err := <-clientErrChan:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		assert.Fail(t, "test timed out after providing input")
	}
}
