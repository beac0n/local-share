package pipedConnection

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"util"
)

type PipedConnection struct {
	doneChan       chan struct{}
	publicListener *net.Listener
	publicConnChan chan *net.Conn
	clientListener *net.Listener
	clientConnChan chan *net.Conn
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

	maxConnections := 20
	connection := PipedConnection{
		doneChan:       make(chan struct{}),
		publicListener: &publicListener,
		publicConnChan: make(chan *net.Conn, maxConnections),
		clientListener: &clientListener,
		clientConnChan: make(chan *net.Conn, maxConnections),
	}

	go connection.createPublicConnections()
	go connection.createClientConnections()

	for i := 0; i < maxConnections; i++ {
		go connection.handleCopyCons()
	}

	return connection, nil
}

func (connection *PipedConnection) createClientConnections() {
	for {
		select {
		case <-connection.doneChan:
			connection.doneChan <- struct{}{}
			return
		default:
		}

		conn, err := (*connection.clientListener).Accept()
		if util.LogIfErr("createClientConnections Accept", err) {
			return
		}
		err = conn.(*net.TCPConn).SetKeepAlive(true)
		util.LogIfErr("createClientConnections SetKeepAlive", err)

		err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Millisecond * 100)
		util.LogIfErr("createClientConnections SetKeepAlivePeriod", err)

		connection.clientConnChan <- &conn
	}
}

func (connection *PipedConnection) createPublicConnections() {
	for {
		select {
		case <-connection.doneChan:
			connection.doneChan <- struct{}{}
			return
		default:
		}

		conn, err := (*connection.publicListener).Accept()
		if util.LogIfErr("createPublicConnections Accept", err) {
			return
		}
		err = conn.(*net.TCPConn).SetKeepAlive(true)
		util.LogIfErr("createPublicConnections SetKeepAlive", err)

		err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Millisecond * 100)
		util.LogIfErr("createClientConnections SetKeepAlivePeriod", err)

		connection.publicConnChan <- &conn
	}
}

func (connection *PipedConnection) Done() {
	_ = (*connection.clientListener).Close()
	_ = (*connection.publicListener).Close()
	connection.doneChan <- struct{}{}
}

func (connection *PipedConnection) handleCopyCons() {
	for {
		var clientConn *net.Conn
		var publicConn *net.Conn

		select {
		case <-connection.doneChan:
			connection.doneChan <- struct{}{}
			return
		case clientConn = <-connection.clientConnChan:
		}

		select {
		case <-connection.doneChan:
			connection.doneChan <- struct{}{}
			return
		case publicConn = <-connection.publicConnChan:
		}

		done := make(chan struct{})
		go util.CopyConn(publicConn, clientConn, done, "public<-client")
		go util.CopyConn(clientConn, publicConn, done, "client<-public")

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

// https://stackoverflow.com/questions/218839/assigning-tcp-ip-ports-for-in-house-application-use
const portRangeStart = 29151
const portRangeEnd = 49151

func initAvailableTcpPorts() *sync.Map {
	availableTcpPorts := &sync.Map{}
	for i := portRangeStart; i <= portRangeEnd; i++ {
		availableTcpPorts.Store(i, true)
	}
	return availableTcpPorts
}

var availableTcpPorts = initAvailableTcpPorts()

func getTcpListener() (net.Listener, error) {
	var listener net.Listener
	var err error
	for i := portRangeStart; i <= portRangeEnd; i++ {
		if portAvailable, ok := availableTcpPorts.Load(i); !ok || !portAvailable.(bool) {
			continue
		}

		availableTcpPorts.Store(i, false)
		listener, err = net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(i))
		if err == nil {
			break
		}
	}

	return listener, err
}

func getPortFromListener(listener *net.Listener) string {
	listenerAddrSplit := strings.Split((*listener).Addr().String(), ":")
	return listenerAddrSplit[len(listenerAddrSplit)-1]
}
