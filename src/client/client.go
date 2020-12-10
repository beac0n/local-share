package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ConnConfig struct {
	serverHost     string
	pipeHost       string
	serverConnChan chan *net.Conn
	pipeConnChan   chan *net.Conn
	logErrors      bool
}

func (connConfig ConnConfig) initServerDial() {
	for {
		serverConn, err := net.Dial("tcp", connConfig.serverHost)
		if connConfig.sleepIfErr(err) {
			continue
		}

		connConfig.serverConnChan <- &serverConn
		pipeConn := <-connConfig.pipeConnChan

		_, _ = io.Copy(*pipeConn, serverConn)
		_ = (*pipeConn).Close()
	}
}

func (connConfig ConnConfig) initPipeDial() {
	for {
		pipeConn, err := net.Dial("tcp", connConfig.pipeHost)
		if connConfig.sleepIfErr(err) {
			continue
		}

		serverConn := <-connConfig.serverConnChan
		connConfig.pipeConnChan <- &pipeConn

		_, _ = io.Copy(*serverConn, pipeConn)
		_ = (*serverConn).Close()
	}
}

func (connConfig ConnConfig) sleepIfErr(err error) bool {
	if err == nil {
		return false
	}

	if connConfig.logErrors {
		log.Println(err)
	}

	time.Sleep(time.Second)
	return true
}

func initPipedConnection(logErrors bool, serverHost string, pipeHost string) {
	config := ConnConfig{
		logErrors:      logErrors,
		serverHost:     serverHost,
		pipeHost:       pipeHost,
		serverConnChan: make(chan *net.Conn),
		pipeConnChan:   make(chan *net.Conn),
	}

	go config.initServerDial()
	go config.initPipeDial()
}

func Run(serverHost, pipeHost, pipeProtocol string) {
	config := getPipedConnectionConfig(serverHost)

	serverHostSplit := strings.Split(serverHost, ":")
	serverHostIp := strings.Join(serverHostSplit[0:len(serverHostSplit)-1], ":")

	fmt.Println("visit", pipeProtocol+"://"+serverHostIp+":"+strconv.Itoa(config.Public))

	for i := 0; i < 100; i++ {
		initPipedConnection(i == 0, serverHostIp+":"+strconv.Itoa(config.Client), pipeHost)
	}

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
