package tcp

import (
	"errors"
	"net"
	"strings"
	"testing"
)

func TestDialAndSend(t *testing.T) {
	t.Run("error when discovering server", func(t *testing.T) {
		discoverServerFn = func(_ chan<- Host, errChan chan<- error) {
			errChan <- errors.New("boom")
		}

		var saw error
		discoverErrorHandler = func(err error) {
			saw = err
		}

		_ = DialAndSend("path/to/file")
			
		if saw == nil || saw.Error() != "boom" {
			t.Fatalf("expected boom; got %v", saw)
		}
	})

	t.Run("received hosts", func(t *testing.T) {
		discoverServerFn = func(receiverChan chan<- Host, _ chan<- error) {
			receiverChan <- Host{
				ID: 1,
				Name: "host-1",
				IP: net.ParseIP("127.0.0.1"),
				Port: 2000,
			}
		}

		scanLn = func(dest ...any) (n int, err error) {
			*dest[0].(*int) = 1
			return 1, nil
		}

		_ = DialAndSend("path/to/file")

		const expectedHostDetail = "1. host-1 -- 127.0.0.1:2000"

		if len(strings.TrimSpace(hostDetail)) == 0 || hostDetail != expectedHostDetail {
			t.Fatalf("expected %s; got %s", expectedHostDetail, hostDetail)
		}
	})
}