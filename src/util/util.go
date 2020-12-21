package util

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"
)

func CopyConn(dst, src *net.Conn, done *chan struct{}, description string) {
	buffer := MakeMaxTcpPacketSizeBuf()

	n, err := (*src).Read(buffer)
	// expected errors
	if netErr, ok := err.(net.Error); err == io.EOF || (ok && netErr.Timeout()) {
		*done <- struct{}{}
		return
	}

	// unexpected errors
	if LogIfErr("CopyConn first Read "+description, err) {
		*done <- struct{}{}
		return
	}

	_, err = (*dst).Write(buffer[0:n])
	if LogIfErr("CopyConn first Write "+description, err) {
		*done <- struct{}{}
		return
	}

	for {
		LogIfErr("CopyConn SetReadDeadline", (*src).SetReadDeadline(time.Now().Add(time.Second*5)))
		n, err := (*src).Read(buffer)
		LogIfErr("CopyConn SetReadDeadline", (*src).SetReadDeadline(time.Time{}))

		// expected errors
		if netErr, ok := err.(net.Error); err == io.EOF || (ok && netErr.Timeout()) || IsBrokenPipeError(err) {
			break
		}

		// unexpected errors
		if LogIfErr("CopyConn Read "+description, err) {
			break
		}

		_, err = (*dst).Write(buffer[0:n])
		// expected errors
		if IsBrokenPipeError(err) {
			break
		}

		// unexpected errors
		if LogIfErr("CopyConn Write "+description, err) {
			break
		}
	}

	*done <- struct{}{}
}

func IsBrokenPipeError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "broken pipe")
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

func MakeMaxTcpPacketSizeBuf() []byte {
	return make([]byte, 65535)
}
