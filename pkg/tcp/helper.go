package tcp

import (
	"net"
	"os"
)

// func formatBytes(b int64) string {
// 	const unit = 1024

// 	if b < unit {
// 		return fmt.Sprintf("%d B", b)
// 	}

// 	div, exp := int64(unit), 0
// 	for n := b / unit; n >= unit; n /= unit {
// 		div *= unit
// 		exp++
// 	}

// 	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
// }

func getListenAddr(listener net.Listener) (string, int) {
	port := listener.Addr().(*net.TCPAddr).Port
	var hostname string
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown host"
	}

	return hostname, port
}
