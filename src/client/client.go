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

func handleServerDial(index int, serverHost string, serverConnChan, pipeConnChan chan *net.Conn) {
	for {
		serverConn, err := net.Dial("tcp", serverHost)
		if err != nil {
			if index == 0 {
				log.Println(err)
			}
			time.Sleep(time.Second)
			continue
		}

		serverConnChan <- &serverConn
		pipeConn := <-pipeConnChan

		_, _ = io.Copy(*pipeConn, serverConn)
		_ = (*pipeConn).Close()
	}
}

func handlePipeDial(index int, pipeHost string, serverConnChan, pipeConnChan chan *net.Conn) {
	for {
		pipeConn, err := net.Dial("tcp", pipeHost)
		if err != nil {
			if index == 0 {
				log.Println(err)
			}
			time.Sleep(time.Second)
			continue
		}

		serverConn := <-serverConnChan
		pipeConnChan <- &pipeConn

		_, _ = io.Copy(*serverConn, pipeConn)
		_ = (*serverConn).Close()
	}
}

func genPipedConnection(index int, serverHost string, pipeHost string) {
	serverConnChan := make(chan *net.Conn)
	pipeConnChan := make(chan *net.Conn)

	go handleServerDial(index, serverHost, serverConnChan, pipeConnChan)
	go handlePipeDial(index, pipeHost, serverConnChan, pipeConnChan)
}

func Run(serverHost, pipeHost, pipeProtocol string) {
	body := getBody(serverHost)

	serverHostSplit := strings.Split(serverHost, ":")
	serverHostIp := strings.Join(serverHostSplit[0:len(serverHostSplit)-1], ":")

	fmt.Println("visit", pipeProtocol + "://" + serverHostIp+":"+strconv.Itoa(body.Public))

	for i := 0; i < 10; i++ {
		genPipedConnection(i, serverHostIp+":"+strconv.Itoa(body.Client), pipeHost)
	}

	// make sure app does not quit
	<-make(chan struct{})
}

type Body struct {
	Client int
	Public int
}

func getBody(serverHost string) Body {
	req, err := http.NewRequest("GET", "http://"+serverHost, nil)
	if err != nil {
		panic(err)
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var body Body
	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		panic(err)
	}

	return body
}
