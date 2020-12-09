package server

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

func handlePublicConn(publicListener *net.Listener, publicConnChan, clientConnChan chan *net.Conn) {
	for {
		publicConn, err := (*publicListener).Accept()

		clientConn := <-clientConnChan
		publicConnChan <- &publicConn

		if err != nil {
			log.Println(err)
			_ = (*clientConn).Close()
			_ = publicConn.Close()
			continue
		}

		_, _ = io.Copy(*clientConn, publicConn)
		_ = (*clientConn).Close()
	}

}

func handleClientConn(clientListener *net.Listener, publicConnChan, clientConnChan chan *net.Conn) {
	for {
		clientConn, err := (*clientListener).Accept()

		clientConnChan <- &clientConn
		publicConn := <-publicConnChan

		if err != nil {
			log.Println(err)
			_ = clientConn.Close()
			_ = (*publicConn).Close()
			continue
		}

		_, _ = io.Copy(*publicConn, clientConn)
		_ = (*publicConn).Close()
	}

}

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

	for i := 0; i < 10; i++ {
		createSinglePipedConnection(publicListener, clientListener)
	}

	return publicListenerPort, clientListenerPort
}

func getTcpListener() (net.Listener, error) {
	return net.Listen("tcp", "0.0.0.0:0")
}

func getPortFromListener(listener net.Listener) string {
	listenerAddrSplit := strings.Split(listener.Addr().String(), ":")
	return listenerAddrSplit[len(listenerAddrSplit)-1]
}

func createSinglePipedConnection(publicListener net.Listener, clientListener net.Listener) {
	publicConnChan := make(chan *net.Conn)
	clientConnChan := make(chan *net.Conn)

	go handlePublicConn(&publicListener, publicConnChan, clientConnChan)
	go handleClientConn(&clientListener, publicConnChan, clientConnChan)
}
