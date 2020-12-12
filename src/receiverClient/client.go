package receiverClient

import (
	"fmt"
	"local-share/src/util"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
)

type ConnConfig struct {
	localPorts  []string
	remotePorts []string
	serverHost  string
}

func Run(serverHost string, localPorts, remotePorts []string) {
	if len(localPorts) != len(remotePorts) {
		fmt.Println("ERROR: local and remote ports must be the same count")
		os.Exit(1)
	}

	config := ConnConfig{
		localPorts:  localPorts,
		remotePorts: remotePorts,
		serverHost:  serverHost,
	}

	for i := range localPorts {
		go config.initServerDial(i)
	}

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt)
	<-signalChannel

	os.Exit(0)

}

func (config *ConnConfig) initServerDial(i int) {
	localPort := config.localPorts[i]
	remotePort := config.remotePorts[i]

	serverHostSplit := strings.Split(config.serverHost, ".")
	serverHostIp := strings.Join(serverHostSplit[1:len(serverHostSplit)], ".")

	log.Println("LISTENING", "127.0.0.1:"+localPort)
	localListener, err := net.Listen("tcp", "127.0.0.1:"+localPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	clientConnChan := make(chan *net.Conn, 10)

	go handleCreateConns(&localListener, &clientConnChan)

	for {
		clientConn := <-clientConnChan
		go config.handleCopyConns(serverHostIp, remotePort, clientConn)
	}
}

func (config *ConnConfig) handleCopyConns(serverHostIp string, remotePort string, clientConn *net.Conn) {
	log.Println("DIALING", remotePort+"."+serverHostIp)
	serverConn, err := net.Dial("tcp", remotePort+"."+serverHostIp)
	if err != nil {
		log.Println("handleCopyConns ERR", err)
		return
	}

	util.CopyConns(&serverConn, clientConn)
}

func handleCreateConns(listener *net.Listener, connChan *chan *net.Conn) {
	for {
		util.HandleCreateConn(listener, connChan)
	}
}
