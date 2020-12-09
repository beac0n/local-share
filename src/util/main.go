package util

import (
	"io"
	"net"
)

func CopyConn(src net.Conn, dst net.Conn) {
	done := make(chan struct{})

	go func() {
		defer src.Close()
		//defer dst.Close()
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}()

	go func() {
		defer src.Close()
		defer dst.Close()
		_, _ = io.Copy(src, dst)
		done <- struct{}{}
	}()

	<-done
	<-done
}
