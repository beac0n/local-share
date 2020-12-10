package client

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

func Run(serverHost, pipeHost, pipeProtocol string) {
	config := getPipedConnectionConfig(serverHost)

	serverHostSplit := strings.Split(serverHost, ":")
	serverHostIp := strings.Join(serverHostSplit[0:len(serverHostSplit)-1], ":")

	publicPortString := strconv.Itoa(config.Public)
	fmt.Println("visit", pipeProtocol+"://"+serverHostIp+":"+publicPortString)

	clientPortString := strconv.Itoa(config.Client)
	serverHostForClient := serverHostIp + ":" + clientPortString

	connConfigFirst := ConnConfig{logErrors: true, serverHost: serverHostForClient, pipeHost: pipeHost}
	connConfig := ConnConfig{logErrors: false, serverHost: serverHostForClient, pipeHost: pipeHost}

	connConfigFirst.initPipedConnection()
	for i := 0; i < 99; i++ {
		connConfig.initPipedConnection()
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	go func() {
		<-signalChannel
		url := "http://" + serverHost + "?client=" + clientPortString + "&public=" + publicPortString
		req, _ := http.NewRequest("DELETE", url, nil)
		_, _ = (&http.Client{}).Do(req)

		os.Exit(0)
	}()

	// make sure app does not quit
	<-make(chan struct{})
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
		panic(err)
	}
}
