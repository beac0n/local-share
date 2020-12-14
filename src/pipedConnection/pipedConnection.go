package pipedConnection

import (
	"local-share/src/util"
	"log"
	"net"
	"strings"
)

type PipedConnection struct {
	done           bool
	publicListener *net.Listener
	publicConnChan *chan *net.Conn
	clientListener *net.Listener
	clientConnChan *chan *net.Conn
}

func NewPipedConnection() PipedConnection {
	publicListener, _ := getTcpListener()
	clientListener, _ := getTcpListener()

	publicConnChan := make(chan *net.Conn, 10)
	clientConnChan := make(chan *net.Conn, 10)

	connection := PipedConnection{
		done:           false,
		publicListener: &publicListener,
		publicConnChan: &publicConnChan,
		clientListener: &clientListener,
		clientConnChan: &clientConnChan,
	}

	go connection.handleCreateConns()
	go connection.handleCopyConns()

	return connection
}

func (connection *PipedConnection) handleCreateConns() {
	for {
		if connection.done {
			_ = (*connection.publicListener).Close()
			_ = (*connection.clientListener).Close()
			return
		}

		publicConn, err := (*connection.publicListener).Accept()
		if err != nil {
			log.Println("handleCreateConns ERR", err)
			return
		}

		clientConn, err := (*connection.clientListener).Accept()
		if err != nil {
			log.Println("handleCreateConns ERR", err)
			_ = (*connection.publicListener).Close()
			return
		}

		*connection.publicConnChan <- &publicConn
		*connection.clientConnChan <- &clientConn
	}
}

func (connection *PipedConnection) Done() {
	connection.done = true
}

func (connection *PipedConnection) handleCopyConns() {
	for {
		if connection.done {
			return
		}

		publicConn := <-(*connection.publicConnChan)
		clientConn := <-(*connection.clientConnChan)

		go connection.handleCopyConn(publicConn, clientConn)
	}
}

func (connection *PipedConnection) handleCopyConn(publicConn *net.Conn, clientConn *net.Conn) {
	clientDone := make(chan struct{})
	publicDone := make(chan struct{})

	go util.CopyConn(clientConn, publicConn, publicDone)
	go util.CopyConn(publicConn, clientConn, clientDone)

	<-publicDone
	_ = (*publicConn).Close()

	<-clientDone
	_ = (*clientConn).Close()
}

func (connection *PipedConnection) GetPublicPort() string {
	return getPortFromListener(connection.publicListener)
}

func (connection *PipedConnection) GetClientPort() string {
	return getPortFromListener(connection.clientListener)
}

func (connection *PipedConnection) GetKey() string {
	return GetKeyFromPorts(connection.GetClientPort(), connection.GetPublicPort())
}

func GetKeyFromPorts(clientPort, publicPort string) string {
	return clientPort + "_" + publicPort
}

func getTcpListener() (net.Listener, error) {
	return net.Listen("tcp", "0.0.0.0:0")
}

func getPortFromListener(listener *net.Listener) string {
	listenerAddrSplit := strings.Split((*listener).Addr().String(), ":")
	return listenerAddrSplit[len(listenerAddrSplit)-1]
}
