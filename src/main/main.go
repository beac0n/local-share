package main

import (
	"flag"
	"fmt"
	"local-share/src/client"
	"local-share/src/server"
)

func main() {
	// client flags
	serverHost := flag.String("server-host", "127.0.0.1:8080", "remote server host")
	pipeHost := flag.String("pipe-host", "127.0.0.1:3000", "host to pipe requests to")
	pipeProtocol := flag.String("pipe-protocol", "http", "the protocol used on the piped host")

	// isServer flags
	isServer := flag.Bool("server", false, "run in server mode")
	host := flag.String("host", "0.0.0.0:8080", "public available http host")

	flag.Parse()

	if *isServer {
		fmt.Println("START SERVER")
		server.Run(*host)
	} else {
		fmt.Println("START CLIENT")
		client.Run(*serverHost, *pipeHost,*pipeProtocol)
	}
}
