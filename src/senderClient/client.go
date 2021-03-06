package senderClient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"local-share/src/util"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type ConnConfig struct {
	serverHost string
	pipeHost   string
}

func (connConfig *ConnConfig) initPipedConnections() {
	for i := 0; i < 20; i++ {
		go connConfig.initPipedConnection()
	}
}

func (connConfig *ConnConfig) initPipedConnection() {
	for {
		serverConn := connConfig.getServerConn()

		done := make(chan struct{})

		buffer := make([]byte, 1)

		n, err := serverConn.Read(buffer)
		if util.LogIfErr("initPipedConnection Read", err) {
			_ = serverConn.Close()
		}

		// init pipeConn as late as possible => some tcp servers drop connection if data is not send immediately
		pipeConn := connConfig.getPipeConn()
		_, err = pipeConn.Write(buffer[0:n])
		if util.LogIfErr("initPipedConnection Write", err) {
			_ = pipeConn.Close()
			_ = serverConn.Close()
		}

		go util.CopyConn(&pipeConn, &serverConn, done, "pipe<-server")
		go util.CopyConn(&serverConn, &pipeConn, done, "server<-pipe")

		<-done
		<-done

		_ = pipeConn.Close()
		_ = serverConn.Close()
	}
}

func (connConfig *ConnConfig) getServerConn() net.Conn {
	serverConn, err := net.Dial("tcp", connConfig.serverHost)
	for err != nil {
		util.LogIfErr("getServerConn", err)
		time.Sleep(time.Second)
		serverConn, err = net.Dial("tcp", connConfig.serverHost)
	}
	return serverConn
}

func (connConfig *ConnConfig) getPipeConn() net.Conn {
	pipeConn, err := net.Dial("tcp", connConfig.pipeHost)
	for err != nil {
		util.LogIfErr("getPipeConn", err)
		time.Sleep(time.Second)
		pipeConn, err = net.Dial("tcp", connConfig.pipeHost)
	}
	return pipeConn
}

func Run(serverHost string, ports []string) {
	deleteSuffixes := make(chan string, len(ports))
	receiverCommandSuffixes := make(chan string, len(ports))

	receiverCommand := "./local-share -receiver -host " + serverHost

	for _, port := range ports {
		go initPipedConnectionForPort(serverHost, port, deleteSuffixes, receiverCommandSuffixes)
	}

	for range ports {
		receiverCommandSuffix := <-receiverCommandSuffixes
		receiverCommand += receiverCommandSuffix
	}

	fmt.Println("receiver command:", receiverCommand)

	util.WaitForSigInt(false)

	for range ports {
		deleteSuffix := <-deleteSuffixes
		deletePipedConnection(serverHost, deleteSuffix)
	}

	os.Exit(0)
}

func initPipedConnectionForPort(serverHost string, port string, deleteSuffixes, receiverCommandSuffixes chan string) {
	config := getPipedConnectionConfig(serverHost)

	serverHostSplit := strings.Split(serverHost, ":")
	serverHostIp := strings.Join(serverHostSplit[0:len(serverHostSplit)-1], ":")

	clientPortString := strconv.Itoa(config.Client)
	publicPortString := strconv.Itoa(config.Public)

	deleteSuffixes <- "?client=" + clientPortString + "&public=" + publicPortString
	receiverCommandSuffixes <- " -local-ports " + port + " -remote-ports " + publicPortString

	connConfig := ConnConfig{serverHost: serverHostIp + ":" + clientPortString, pipeHost: "127.0.0.1:" + port}
	connConfig.initPipedConnections()
}

func deletePipedConnection(serverHost string, deleteSuffix string) {
	req, err := http.NewRequest("DELETE", "http://"+serverHost+deleteSuffix, nil)
	util.LogIfErr("deletePipedConnection NewRequest", err)

	_, err = (&http.Client{}).Do(req)
	util.LogIfErr("deletePipedConnection Do", err)
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

	util.LogIfErr("getPipedConnectionConfig body close", resp.Body.Close())

	return body
}

func panicOnErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
