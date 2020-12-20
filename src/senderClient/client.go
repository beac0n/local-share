package senderClient

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"util"
)

type ConnConfig struct {
	serverHost string
	pipeHost   string
	log        bool
}

func (connConfig *ConnConfig) sleepIfErr(err error) bool {
	if !connConfig.LogIfErr(err) {
		return false
	}

	time.Sleep(time.Second)
	return true
}

func (connConfig *ConnConfig) LogIfErr(err error) bool {
	if connConfig.log {
		util.LogIfErr(err)
	}

	return err != nil
}

func (connConfig *ConnConfig) initPipedConnection() {
	for {
		connConfig.handlePipedConnection()
	}
}

func (connConfig *ConnConfig) handlePipedConnection() {
	serverConn := connConfig.getServerConn()

	readBuf := makeMaxTcpPacketSizeBuf()

	// read some data from server to cache locally
	n, err := serverConn.Read(readBuf)
	util.LogIfErr(err)

	// create pipeConn late, in case pipe server has short keep-alive
	pipeConn := connConfig.getPipeConn()

	_, err = pipeConn.Write(readBuf[0:n])
	util.LogIfErr(err)

	done := make(chan struct{})
	go connConfig.handleReadServer(&serverConn, &pipeConn, &done)
	connConfig.handleWriteServer(&pipeConn, &serverConn)

	<-done

	err = pipeConn.Close()
	util.LogIfErr(err)

	err = serverConn.Close()
	util.LogIfErr(err)
}

func (connConfig *ConnConfig) handleWriteServer(pipeConn, serverConn *net.Conn) {
	for {
		readBuf := makeMaxTcpPacketSizeBuf()

		// pipe conn should send back immediately, because we've written to it
		err := (*pipeConn).SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		util.LogIfErr(err)

		// everything written to pipe, now read back everything
		n, err := (*pipeConn).Read(readBuf)
		if err != nil && (err == io.EOF || err.(net.Error).Timeout()) {
			return
		}
		util.LogIfErr(err)

		_, err = (*serverConn).Write(readBuf[0:n])
		util.LogIfErr(err)
	}
}

func makeMaxTcpPacketSizeBuf() []byte {
	return make([]byte, 65535)
}

func (connConfig *ConnConfig) handleReadServer(serverConn *net.Conn, pipeConn *net.Conn, done *chan struct{}) {
	for {
		readBuf := makeMaxTcpPacketSizeBuf()

		// after first read, data should come in until everything is here
		err := (*serverConn).SetReadDeadline(time.Now().Add(time.Second))
		util.LogIfErr(err)

		n, err := (*serverConn).Read(readBuf)
		if err != nil && (err == io.EOF || err.(net.Error).Timeout()) {
			*done <- struct{}{}
			return
		}
		util.LogIfErr(err)

		_, err = (*pipeConn).Write(readBuf[0:n])
		util.LogIfErr(err)
	}
}

func (connConfig *ConnConfig) getServerConn() net.Conn {
	serverConn, err := net.Dial("tcp", connConfig.serverHost)
	for err != nil {
		connConfig.LogIfErr(err)
		time.Sleep(time.Second)
		serverConn, err = net.Dial("tcp", connConfig.serverHost)
	}
	return serverConn
}

func (connConfig *ConnConfig) getPipeConn() net.Conn {
	pipeConn, err := net.Dial("tcp", connConfig.pipeHost)
	for err != nil {
		connConfig.LogIfErr(err)
		time.Sleep(time.Second)
		pipeConn, err = net.Dial("tcp", connConfig.pipeHost)
	}
	return pipeConn
}

func Run(serverHost string, ports []string) {
	deleteSuffixes := make(chan string, len(ports))

	for _, port := range ports {
		go initPipedConnectionForPort(serverHost, port, deleteSuffixes)
	}

	util.WaitForSigInt(false)

	deleteAllPipedConnections(serverHost, deleteSuffixes)

	os.Exit(0)
}

func deleteAllPipedConnections(serverHost string, deleteSuffixes chan string) {
	for {
		select {
		case deleteSuffix := <-deleteSuffixes:
			go deletePipedConnection(serverHost, deleteSuffix)
		default:
			return
		}
	}
}

func initPipedConnectionForPort(serverHost string, port string, deleteSuffixes chan string) {
	config := getPipedConnectionConfig(serverHost)

	serverHostSplit := strings.Split(serverHost, ":")
	serverHostIp := strings.Join(serverHostSplit[0:len(serverHostSplit)-1], ":")

	clientPortString := strconv.Itoa(config.Client)
	publicPortString := strconv.Itoa(config.Public)

	deleteSuffixes <- "?client=" + clientPortString + "&public=" + publicPortString

	pipeHost := "127.0.0.1:" + port
	serverHostForClient := serverHostIp + ":" + clientPortString
	serverHostForPublic := serverHostIp + ":" + publicPortString

	fmt.Println(pipeHost, "<-", serverHostForClient, "<-", serverHostForPublic, "<- sender")

	connConfigFirst := ConnConfig{log: true, serverHost: serverHostForClient, pipeHost: pipeHost}
	connConfig := ConnConfig{log: false, serverHost: serverHostForClient, pipeHost: pipeHost}

	go connConfigFirst.initPipedConnection()
	for i := 1; i < 100; i++ {
		go connConfig.initPipedConnection()
	}
}

func deletePipedConnection(serverHost string, deleteSuffix string) {
	req, err := http.NewRequest("DELETE", "http://"+serverHost+deleteSuffix, nil)
	util.LogIfErr(err)

	_, err = (&http.Client{}).Do(req)
	util.LogIfErr(err)
}

type Config struct {
	Client int
	Public int
}

func getPipedConnectionConfig(serverHost string) Config {
	req, err := http.NewRequest("GET", "http://"+serverHost, nil)
	panicOnErr(err)

	resp, err := (&http.Client{}).Do(req)
	panicOnErr(err)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	panicOnErr(err)

	var body Config
	err = json.Unmarshal(bodyBytes, &body)
	panicOnErr(err)

	util.LogIfErr(resp.Body.Close())

	return body
}

func panicOnErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
