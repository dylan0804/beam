package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/dylan0804/beam/pkg/tcp"
)

type config struct {
	port string
	path string
}

func parseFlags() (*config, error) {
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("need \"receive\" or \"send\" subcommand")
	}

	cfg := &config{}

	switch os.Args[1] {
	case "receive":
		receiveCmd := flag.NewFlagSet("receive", flag.ExitOnError)
		receiveCmd.StringVar(&cfg.port, "port", "0", "port to listen on (0 for random)")
		if err := receiveCmd.Parse(os.Args[2:]); err != nil {
			return nil, fmt.Errorf("failed to parse receive flags: %v", err)
		}
	case "send":
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		sendCmd.StringVar(&cfg.path, "path", "", "absolute path of file")
		if err := sendCmd.Parse(os.Args[2:]); err != nil {
			return nil, fmt.Errorf("failed to parse send flags: %v", err)
		}
		if cfg.path == "" {
			return nil, fmt.Errorf("input the absolute path of the file you wish to send")
		}
	default:
		return nil, fmt.Errorf("unknown command %s: want receive or send", os.Args[1])
	}

	return cfg, nil
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	switch os.Args[1] {
	case "receive":
		// setup DI
		listener := tcp.NewListener()
		broadcaster := tcp.NewBroadcaster()
		connHandler := tcp.NewHandler()

		server := tcp.NewServer(listener, broadcaster, connHandler)
		if err := server.Start(cfg.port); err != nil {
			log.Fatalf("receive failed: %v", err)
		}
	case "send":
		discoverer := tcp.NewUDPDiscoverer(9999)
		dialer := net.Dial
		client := tcp.NewClient(dialer, discoverer, os.Stdin, os.Stdout)
		if err := client.DialAndSend(cfg.path); err != nil {
			log.Fatalf("dial and send failed: %v", err)
		}
	}
}
