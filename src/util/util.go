package util

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

func CopyConn(dst, src *net.Conn, done chan struct{}, description string) {
	buffer := make([]byte, 1)

	LogIfErr("CopyConn SetReadDeadline First", (*src).SetReadDeadline(time.Now().Add(time.Hour)))
	n, err := (*src).Read(buffer)
	LogIfErr("CopyConn reset SetReadDeadline", (*src).SetReadDeadline(time.Time{}))
	if LogIfErr("CopyConn first Read "+description, err) {
		done <- struct{}{}
		return
	}

	_, err = (*dst).Write(buffer[0:n])
	if LogIfErr("CopyConn first Write "+description, err) {
		done <- struct{}{}
		return
	}

	LogIfErr("CopyConn SetReadDeadline", (*src).SetReadDeadline(time.Now().Add(time.Second*5)))
	_, _ = io.Copy(*dst, *src)
	LogIfErr("CopyConn reset SetReadDeadline", (*src).SetReadDeadline(time.Time{}))

	done <- struct{}{}
}

func LogIfErr(prefix string, err error) bool {
	isError := err != nil
	if isError {
		log.Println("ERROR:", prefix, err)
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
