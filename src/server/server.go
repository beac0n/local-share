package server

import (
	"net/http"
	"pipedConnection"
	"sync"
	"util"
)

type ReqHandler struct {
}

var pipedConnections *sync.Map

func Run(host string) {
	pipedConnections = &sync.Map{}

	if err := http.ListenAndServe(host, &ReqHandler{}); err != nil {
		panic(err)
	}
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

	conn, err := pipedConnection.NewPipedConnection()
	if util.LogIfErr("server create piped connection", err) {
		resWriter.WriteHeader(500)
		return
	}

	pipedConnections.Store(conn.GetKey(), &conn)

	message := "{\"client\":" + conn.GetClientPort() + ", \"public\": " + conn.GetPublicPort() + "}"
	if _, err := resWriter.Write([]byte(message)); util.LogIfErr("server create piped connection message", err) {
		resWriter.WriteHeader(500)
		return
	}
}

func (reqHandler *ReqHandler) delete(resWriter http.ResponseWriter, req *http.Request) {
	queryValues := req.URL.Query()
	clientPort := queryValues.Get("client")
	publicPort := queryValues.Get("public")

	key := pipedConnection.GetKeyFromPorts(clientPort, publicPort)
	conn, ok := pipedConnections.Load(key)

	if !ok {
		resWriter.WriteHeader(400)
		return
	}

	conn.(*pipedConnection.PipedConnection).Done()
	pipedConnections.Delete(key)
}
