package senderClient

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

type ConnConfig struct {
	serverHost string
	pipeHost   string
	logErrors  bool
}

func (connConfig *ConnConfig) initServerDial(serverConnChan, pipeConnChan chan *net.Conn) {
	for {
		serverConn, err := net.Dial("tcp", connConfig.serverHost)
		if connConfig.sleepIfErr(err) {
			continue
		}

		serverConnChan <- &serverConn
		pipeConn := <-pipeConnChan

		_, _ = io.Copy(*pipeConn, serverConn)
		_ = (*pipeConn).Close()
	}
}

func (connConfig *ConnConfig) initPipeDial(serverConnChan, pipeConnChan chan *net.Conn) {
	for {
		pipeConn, err := net.Dial("tcp", connConfig.pipeHost)
		if connConfig.sleepIfErr(err) {
			continue
		}

		serverConn := <-serverConnChan
		pipeConnChan <- &pipeConn

		_, _ = io.Copy(*serverConn, pipeConn)
		_ = (*serverConn).Close()
	}
}

func (connConfig *ConnConfig) sleepIfErr(err error) bool {
	if err == nil {
		return false
	}

	if connConfig.logErrors {
		log.Println(err)
	}

	time.Sleep(time.Second)
	return true
}

func (connConfig *ConnConfig) initPipedConnection() {
	serverConnChan := make(chan *net.Conn)
	pipeConnChan := make(chan *net.Conn)

	go connConfig.initServerDial(serverConnChan, pipeConnChan)
	go connConfig.initPipeDial(serverConnChan, pipeConnChan)
}

func Run(serverHost string, ports []string) {
	deleteSuffixes := make(chan string, len(ports))

	for _, port := range ports {
		go initPipedConnectionForPort(serverHost, port, deleteSuffixes)
	}

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt)
	<-signalChannel

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

	serverHostSplit := strings.Split(serverHost, ".")
	serverHostIp := strings.Join(serverHostSplit[1:len(serverHostSplit)], ".")

	clientPortString := strconv.Itoa(config.Client)

	deleteSuffixes <- "?client=" + clientPortString + "&public=" + strconv.Itoa(config.Public)

	serverHostForClient := clientPortString + "." + serverHostIp
	pipeHost := "127.0.0.1:" + port

	fmt.Println(serverHostForClient)

	connConfigFirst := ConnConfig{logErrors: true, serverHost: serverHostForClient, pipeHost: pipeHost}
	connConfig := ConnConfig{logErrors: false, serverHost: serverHostForClient, pipeHost: pipeHost}

	connConfigFirst.initPipedConnection()
	for i := 1; i < 100; i++ {
		connConfig.initPipedConnection()
	}
}

func deletePipedConnection(serverHost string, deleteSuffix string) {
	req, _ := http.NewRequest("DELETE", "http://"+serverHost+deleteSuffix, nil)
	_, _ = (&http.Client{}).Do(req)
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
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	panicOnErr(err)

	var body Config
	err = json.Unmarshal(bodyBytes, &body)
	panicOnErr(err)

	return body
}

func panicOnErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
