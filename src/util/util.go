package util

import (
	"io"
	"log"
	"net"
)

func CopyConns(src, dst *net.Conn) {
	done := make(chan struct{})

	go copyConn(src, dst, done)
	go copyConn(dst, src, done)

	<-done
	<-done
}

func copyConn(dst, src *net.Conn, done chan struct{}) {
	if _, err := io.Copy(*dst, *src); err != nil {
		log.Println("copyConn Copy ERR", err)
	}
	if err := (*dst).Close(); err != nil {
		log.Println("copyConn Close ERR", err)
	}
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
