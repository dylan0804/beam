package tcp

import (
	"log"
	"net"
	"os"
	"path/filepath"
)

func getListenAddr(listener net.Listener) (string, int) {
	port := listener.Addr().(*net.TCPAddr).Port
	var hostname string
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown host"
	}

	return hostname, port
}

func getDownloadPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot get user home directory: %v", err)
	}
	return filepath.Join(home, "Downloads")
}
