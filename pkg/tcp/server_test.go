package tcp

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

type fakeListener struct {
	connOnce sync.Once
	conn net.Conn
}

func (f *fakeListener) Accept() (net.Conn, error) {
	f.connOnce.Do(func() {
		server, client := net.Pipe()
		client.Close()
		f.conn = server
	})
	if f.conn != nil {
		c := f.conn
		f.conn = nil
		return c, nil
	}
	return nil, net.ErrClosed
}
func (f *fakeListener) Close() error { return nil }
func (f *fakeListener) Addr() net.Addr { return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}}

var (
	bcalls struct {
		sync.Mutex
		count int
		lastPort int
		lastHost string
	}

	hcalls struct {
		sync.Mutex
		count int
	}
)

func fakeBroadcast(port int, hostname string) {
	bcalls.Lock()
	defer bcalls.Unlock()
	bcalls.count++
	bcalls.lastHost = hostname
	bcalls.lastPort = port
}

func fakeHandler(c net.Conn) error {
	hcalls.Lock()
	hcalls.count++
	hcalls.Unlock()
	return nil
}

func resetSpies() {
    bcalls.Lock()
    bcalls.count = 0
    bcalls.lastHost = ""
    bcalls.lastPort = 0
    bcalls.Unlock()

    hcalls.Lock()
    hcalls.count = 0
    hcalls.Unlock()
}

func setup() func() {
	origListen := netListen
	origBroadcast := startBroadcastFn
    origHandler := handleConnectionFn
    origHostname := osHostnameFn
    origLog := logPrintln

	resetSpies()

	return func() {
		netListen = origListen
        startBroadcastFn = origBroadcast
        handleConnectionFn = origHandler
        osHostnameFn = origHostname
        logPrintln = origLog
	}
}

func TestServer_SuccessfulShutdown(t *testing.T) {
	teardown := setup()
	defer teardown()

	netListen = func(network, addr string) (net.Listener, error) {
		return &fakeListener{}, nil
	}

	startBroadcastFn = fakeBroadcast
	handleConnectionFn = fakeHandler
	osHostnameFn = func() (name string, err error) { return "", errors.New("fail host")}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := Listen(ctx, 0); err != nil {
		t.Fatalf("Listen returned error %v", err)
	}

	bcalls.Lock()
	if bcalls.count != 1 {
		t.Errorf("broadcast called = %d; want 1", bcalls.count)
	}
	if bcalls.lastHost != "unknown host" {
		t.Errorf("hostname = %q; want %q", bcalls.lastHost, "unknown host")
	}
	bcalls.Unlock()

	hcalls.Lock()
    if hcalls.count != 1 {
        t.Errorf("handleConnection called = %d; want 1", hcalls.count)
    }
    hcalls.Unlock()
}

func TestServer_ListenError(t *testing.T) {
	teardown := setup()
	defer teardown()

	netListen = func(network, addr string) (net.Listener, error) {
		return nil, errors.New("listen error")
	}

	err := Listen(context.Background(), 0); 
	if err == nil || err.Error() != "listen error" {
		t.Fatalf("expected listen error; got %v", err)
	}
}

func TestServer_ConnectionError(t *testing.T) {
	teardown := setup()
	defer teardown()

	sentinel := errors.New("oops")

	netListen = func(network, addr string) (net.Listener, error) {
		return &fakeListener{}, nil
	}

	startBroadcastFn = fakeBroadcast

	osHostnameFn = func() (name string, err error) { return "host-ok", nil}

	handleConnectionFn = func(c net.Conn) error {
		return sentinel
	}

	var gotErrLog error

	logPrintln = func(args ...any) {
		if len(args) >= 2 {
			if e, ok := args[1].(error); ok {
				gotErrLog = e
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := Listen(ctx, 0); err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}

	if gotErrLog != sentinel {
		t.Errorf("logged error = %v; want %v", gotErrLog, sentinel)
	}
}