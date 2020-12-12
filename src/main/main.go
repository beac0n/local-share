package main

import (
	"flag"
	"fmt"
	"local-share/src/client"
	"local-share/src/server"
	"strings"
)

type ArrayFlags []string

func (list *ArrayFlags) String() string {
	return strings.Join(*list, " ")
}

func (list *ArrayFlags) Set(value string) error {
	*list = append(*list, value)
	return nil
}

var ports ArrayFlags

func main() {
	host := flag.String("host", "0.0.0.0:8080", "public available server http host")

	// client flags
	flag.Var(&ports, "ports", "list of localhost ports to pipe traffic to")

	// isServer flags
	isServer := flag.Bool("server", false, "run in server mode")

	flag.Parse()

	if *isServer {
		fmt.Println("starting server...")
		server.Run(*host)
	} else {
		fmt.Println("starting client...")
		client.Run(*host, ports)
	}
}
