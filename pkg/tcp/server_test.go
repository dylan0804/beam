package tcp

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockNetListener struct {
	serverConn net.Conn
	clientConn net.Conn
	conn       chan net.Conn
	errs       chan error
	addr       net.Addr
}

func (m *mockNetListener) Accept() (net.Conn, error) {
	select {
	case conn := <-m.conn:
		return conn, nil
	case err := <-m.errs:
		return nil, err
	}
}
func (m *mockNetListener) Addr() net.Addr { return m.addr }
func (m *mockNetListener) Close() error {
	if m.clientConn != nil {
		m.clientConn.Close()
	}
	if m.serverConn != nil {
		m.serverConn.Close()
	}
	return nil
}

type mockListener struct {
	ln *mockNetListener
}

func (m *mockListener) Listen(addr string) (NetListener, error) {
	return m.ln, nil
}

type mockBroadcaster struct{}

func (m *mockBroadcaster) StartBroadcast(port int, host string) {}

type mockHandler struct {
	actual chan uint32
	err    chan error
}

func (m *mockHandler) HandleConnection(conn net.Conn) {
	defer conn.Close()

	var actual uint32
	err := binary.Read(conn, binary.BigEndian, &actual)
	if err != nil {
		m.err <- err
		return
	}
	m.actual <- actual
}

func TestServer(t *testing.T) {
	mockListener := &mockListener{
		ln: &mockNetListener{
			addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345},
			conn: make(chan net.Conn),
			errs: make(chan error),
		},
	}
	mockBroadcaster := &mockBroadcaster{}
	mockHandler := &mockHandler{
		actual: make(chan uint32),
		err:    make(chan error),
	}

	server := NewServer(mockListener, mockBroadcaster, mockHandler)

	serverChanErr := make(chan error)
	go func() {
		serverChanErr <- server.Start(":8080")
	}()

	serverConn, clientConn := net.Pipe()
	mockListener.ln.conn <- serverConn

	expected := uint32(5)
	err := binary.Write(clientConn, binary.BigEndian, expected)
	assert.NoError(t, err)

	select {
	case actual := <-mockHandler.actual:
		assert.Equal(t, expected, actual)
	case err := <-mockHandler.err:
		assert.NoError(t, err)
	}

	mockListener.ln.Close()
	mockListener.ln.errs <- net.ErrClosed

	select {
	case err := <-serverChanErr:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		assert.Fail(t, "test timed out waiting for server to shut down")
	}
}
