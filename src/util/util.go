package util

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

func CopyConn(dst, src *net.Conn, done chan struct{}) {
	_, err := io.Copy(*dst, *src)
	LogIfErr(err)

	done <- struct{}{}
}

func LogIfErr(err error) bool {
	isError := err != nil
	if isError {
		log.Println(err)
	}
	return isError
}

func WaitForSigInt(osExit bool) {
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt)
	<-signalChannel

	if osExit {
		os.Exit(0)
	}
}
