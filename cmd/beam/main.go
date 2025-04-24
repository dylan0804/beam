package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dylan0804/beam/pkg/tcp"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal(`need “receive” or “send” subcommand`)
	}


	switch os.Args[1] {
	case "receive":
		receiveCmd := flag.NewFlagSet("receive", flag.ExitOnError)
		port := receiveCmd.Int("port", 0, "port to listen on (0 for random)")
		receiveCmd.Parse(os.Args[2:])

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(context.Background())

		go func(){
			<-sigCh
			fmt.Println("shutting down…")
			cancel()
		}()

		if err := tcp.Listen(ctx, *port); err != nil {
			log.Fatalf("receive failed: %v", err)
		}
	case "send":
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		path := sendCmd.String("path", "", "absolute path of file")
		sendCmd.Parse(os.Args[2:])

		if *path == "" {
			log.Fatalln("input the absolute path of the file you wish to send")
		}

		if err := tcp.DialAndSend(*path); err != nil {
			log.Fatalf("dial and send failed: %v", err)
		}
	default:
		fmt.Printf("unknown command %s: want receive or send", os.Args[1])
	}
}