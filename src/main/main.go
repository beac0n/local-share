package main

import (
	"errors"
	"flag"
	"fmt"
	"receiverClient"
	"senderClient"
	"server"
	"strconv"
	"strings"
)

type PortFlags []string

func (list *PortFlags) String() string {
	return strings.Join(*list, " ")
}

func (list *PortFlags) Set(value string) error {
	if strings.Contains(value, "-") {
		split := strings.Split(value, "-")
		if len(split) != 2 {
			return errors.New("invalid port range")
		}

		rangeStart, err := strconv.Atoi(split[0])
		if err != nil {
			return err
		}
		rangeEnd, err := strconv.Atoi(split[1])
		if err != nil {
			return err
		}

		for i := rangeStart; i < rangeEnd; i++ {
			*list = append(*list, strconv.Itoa(i))
		}
	} else {
		if _, err := strconv.Atoi(value); err != nil {
			return err
		}

		*list = append(*list, value)
	}

	return nil
}

func main() {
	host := flag.String("host", "0.0.0.0:8080", "public available server http host")

	// senderClient flags
	isReceiver := flag.Bool("receiver", false, "run client in receiver mode")

	var localPorts PortFlags
	flag.Var(&localPorts, "local-ports", "list of localhost ports to send traffic to")

	var remotePorts PortFlags
	flag.Var(&remotePorts, "remote-ports", "list of remote ports to send traffic to")

	// isServer flags
	isServer := flag.Bool("server", false, "run in server mode")

	flag.Parse()

	if *isServer {
		fmt.Println("starting server...")
		server.Run(*host)
	} else if *isReceiver {
		fmt.Println("starting client in receiver mode...")
		receiverClient.Run(*host, localPorts, remotePorts)
	} else {
		fmt.Println("starting client in sender mode...")
		senderClient.Run(*host, localPorts)
	}
}
