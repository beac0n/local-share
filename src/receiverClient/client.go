package receiverClient

import (
	"fmt"
	"local-share/src/util"
	"net"
	"os"
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

	util.WaitForSigInt(true)
}

func (config *ConnConfig) initServerDial(i int) {
	localPort := config.localPorts[i]
	remotePort := config.remotePorts[i]

	serverHostSplit := strings.Split(config.serverHost, ":")
	serverHostIp := strings.Join(serverHostSplit[0:len(serverHostSplit)-1], ":")

	localListener, err := net.Listen("tcp", "127.0.0.1:"+localPort)
	if util.LogIfErr("initServerDial", err) {
		return
	}

	clientConnChan := make(chan *net.Conn)
	go handleCreateCons(&localListener, clientConnChan)

	for {
		clientConn := <-clientConnChan
		go config.handleCopyCons(serverHostIp, remotePort, clientConn)
	}
}

func (config *ConnConfig) handleCopyCons(serverHostIp string, remotePort string, clientConn *net.Conn) {
	serverConn, err := net.Dial("tcp", serverHostIp+":"+remotePort)
	if util.LogIfErr("handleCopyCons", err) {
		return
	}

	done := make(chan struct{})

	go util.CopyConn(&serverConn, clientConn, done, "server<-client")
	go util.CopyConn(clientConn, &serverConn, done, "client<-sever")

	<-done
	<-done

	util.LogIfErr("handleCopyCons serverConn close", serverConn.Close())
	util.LogIfErr("handleCopyCons clientConn close", (*clientConn).Close())
}

func handleCreateCons(listener *net.Listener, connChan chan *net.Conn) {
	for {
		conn, err := (*listener).Accept()
		if !util.LogIfErr("handleCreateCons", err) {
			connChan <- &conn
		}
	}
}
