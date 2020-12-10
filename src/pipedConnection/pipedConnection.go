package pipedConnection

import (
	"io"
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

		go copyConns(clientConn, publicConn)
	}
}

func copyConns(src, dst *net.Conn) {
	done := make(chan struct{})

	go copyConn(src, dst, done)
	go copyConn(dst, src, done)

	<-done
	<-done
}

func copyConn(dst, src *net.Conn, done chan struct{}) {
	_, _ = io.Copy(*dst, *src)
	_ = (*dst).Close()
	done <- struct{}{}
}

func (connection *PipedConnection) handleCreateConns(listener *net.Listener, connChan *chan *net.Conn) {
	for {
		if connection.done {
			_ = (*listener).Close()
			return
		}

		conn, err := (*listener).Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		*connChan <- &conn
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
