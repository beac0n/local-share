package util

import (
	"io"
	"log"
	"net"
)

func CopyConns(src, dst *net.Conn) {
	done := make(chan struct{})

	go CopyConn(src, dst, done)
	go CopyConn(dst, src, done)

	log.Println("CopyConns waiting for copy done")

	<-done
	<-done

	log.Println("CopyConns closing conns")
	if err := (*src).Close(); err != nil {
		log.Println("CopyConn Close src ERR", err)
	}

	if err := (*dst).Close(); err != nil {
		log.Println("CopyConn Close dst ERR", err)
	}
}

func CopyConn(dst, src *net.Conn, done chan struct{}) {
	_, _ = io.Copy(*dst, *src)
	done <- struct{}{}
}

func HandleCreateConn(listener *net.Listener, connChan *chan *net.Conn) {
	conn, err := (*listener).Accept()
	if err != nil {
		log.Println("HandleCreateConn ERR", err)
		return
	}

	*connChan <- &conn
}

func LogIfErr(err error) bool {
	isError := err != nil
	if isError {
		log.Println(err)
	}
	return isError
}
