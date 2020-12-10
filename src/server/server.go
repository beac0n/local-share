package server

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type ReqHandler struct {
}

func (reqHandler *ReqHandler) ServeHTTP(resWriter http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		reqHandler.create(resWriter)
	} else if req.Method == "DELETE" {
		reqHandler.delete(resWriter, req)
	} else {
		resWriter.WriteHeader(404)
	}
}

func (reqHandler *ReqHandler) create(resWriter http.ResponseWriter) {
	headers := resWriter.Header()
	headers.Add("Content-Type", "application/json")

	publicPort, clientPort := createPipedConnection()

	if _, err := resWriter.Write([]byte("{\"client\":" + clientPort + ", \"public\": " + publicPort + "}")); err != nil {
		resWriter.WriteHeader(500)
		return
	}
}

func (reqHandler *ReqHandler) delete(writer http.ResponseWriter, req *http.Request) {
	queryValues := req.URL.Query()
	clientPort := queryValues.Get("client")
	publicPort := queryValues.Get("public")

	// TODO: implement stopping
	log.Println(clientPort, publicPort)
}

func Run(host string) {
	if err := http.ListenAndServe(host, &ReqHandler{}); err != nil {
		panic(err)
	}
}

func createPipedConnection() (string, string) {
	publicListener, _ := getTcpListener()
	clientListener, _ := getTcpListener()

	publicListenerPort := getPortFromListener(publicListener)
	clientListenerPort := getPortFromListener(clientListener)

	createSinglePipedConnection(&publicListener, &clientListener)

	return publicListenerPort, clientListenerPort
}

func createSinglePipedConnection(publicListener *net.Listener, clientListener *net.Listener) {
	publicConnChan := make(chan *net.Conn, 100)
	clientConnChan := make(chan *net.Conn, 100)

	go handlePublicConn(publicListener, publicConnChan, clientConnChan)
	go handleClientConn(clientListener, publicConnChan, clientConnChan)
}

func handlePublicConn(publicListener *net.Listener, publicConnChan, clientConnChan chan *net.Conn) {
	for {
		publicConn, err := (*publicListener).Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		clientConn := <-clientConnChan
		publicConnChan <- &publicConn

		_, _ = io.Copy(*clientConn, publicConn)
		_ = (*clientConn).Close()
	}

}

func handleClientConn(clientListener *net.Listener, publicConnChan, clientConnChan chan *net.Conn) {
	for {
		clientConn, err := (*clientListener).Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		clientConnChan <- &clientConn
		publicConn := <-publicConnChan

		_, _ = io.Copy(*publicConn, clientConn)
		_ = (*publicConn).Close()
	}

}

func getTcpListener() (net.Listener, error) {
	return net.Listen("tcp", "0.0.0.0:0")
}

func getPortFromListener(listener net.Listener) string {
	listenerAddrSplit := strings.Split(listener.Addr().String(), ":")
	return listenerAddrSplit[len(listenerAddrSplit)-1]
}
