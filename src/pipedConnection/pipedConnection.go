package pipedConnection

import (
	"net"
	"strings"
	"time"
	"util"
)

type PipedConnection struct {
	done           bool
	publicListener *net.Listener
	publicConnChan *chan *net.Conn
	clientListener *net.Listener
	clientConnChan *chan *net.Conn
}

func NewPipedConnection() (PipedConnection, error) {
	publicListener, err := getTcpListener()
	if err != nil {
		return PipedConnection{}, err
	}

	clientListener, err := getTcpListener()
	if err != nil {
		return PipedConnection{}, err
	}

	maxConnections := 100
	publicConnChan := make(chan *net.Conn, maxConnections)
	clientConnChan := make(chan *net.Conn, maxConnections)

	connection := PipedConnection{
		done:           false,
		publicListener: &publicListener,
		publicConnChan: &publicConnChan,
		clientListener: &clientListener,
		clientConnChan: &clientConnChan,
	}

	go connection.createPublicConnections()
	go connection.createClientConnections()

	for i := 0; i < maxConnections; i++ {
		go connection.handleCopyConns()
	}

	return connection, nil
}

func (connection *PipedConnection) createClientConnections() {
	for {
		if connection.done {
			util.LogIfErr("", (*connection.clientListener).Close())
			return
		}

		conn, err := (*connection.clientListener).Accept()
		if util.LogIfErr("createClientConnections Accept", err) {
			return
		}
		err = conn.(*net.TCPConn).SetKeepAlive(true)
		util.LogIfErr("createClientConnections SetKeepAlive", err)

		err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Millisecond * 100)
		util.LogIfErr("createClientConnections SetKeepAlivePeriod", err)

		*connection.clientConnChan <- &conn
	}
}

func (connection *PipedConnection) createPublicConnections() {
	for {
		if connection.done {
			util.LogIfErr("", (*connection.publicListener).Close())
			return
		}

		conn, err := (*connection.publicListener).Accept()
		if util.LogIfErr("createPublicConnections Accept", err) {
			return
		}
		err = conn.(*net.TCPConn).SetKeepAlive(true)
		util.LogIfErr("createPublicConnections SetKeepAlive", err)

		err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Millisecond * 100)
		util.LogIfErr("createClientConnections SetKeepAlivePeriod", err)

		*connection.publicConnChan <- &conn
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

		clientConn := <-(*connection.clientConnChan)
		publicConn := <-(*connection.publicConnChan)

		done := make(chan struct{})
		go util.CopyConn(publicConn, clientConn, &done, "public<-client")
		go util.CopyConn(clientConn, publicConn, &done, "client<-public")

		<-done
		<-done

		_ = (*publicConn).Close()
		_ = (*clientConn).Close()
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
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	return listener, err
}

func getPortFromListener(listener *net.Listener) string {
	listenerAddrSplit := strings.Split((*listener).Addr().String(), ":")
	return listenerAddrSplit[len(listenerAddrSplit)-1]
}
