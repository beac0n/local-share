package pipedConnection

import (
	"local-share/src/util"
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

	go connection.handleCreateConns(connection.publicListener, connection.publicConnChan)
	go connection.handleCreateConns(connection.clientListener, connection.clientConnChan)
	go connection.handleCopyConns()

	return connection
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

		go util.CopyConns(clientConn, publicConn)
	}
}

func (connection *PipedConnection) handleCreateConns(listener *net.Listener, connChan *chan *net.Conn) {
	for {
		if connection.done {
			_ = (*listener).Close()
			return
		}

		util.HandleCreateConn(listener, connChan)
	}
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
