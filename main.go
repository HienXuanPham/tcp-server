package main

import (
	"errors"
	"io"
	"log"
	"net"
	"time"
)

const (
	addr         = "127.0.0.1:8080"
	readTimeout  = 30 * time.Second
	writeTimeout = 30 * time.Second
)

func main() {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	defer listener.Close()

	log.Printf("listening on %s", addr)

	conn, err := listener.Accept()
	if err != nil {
		log.Fatalf("accept failed: %v", err)
	}
	defer conn.Close()

	log.Printf("accepted remote=%s local=%s", conn.RemoteAddr(), conn.LocalAddr())

	if err := handleConn(conn); err != nil {
		log.Printf("connection ended: %v", err)
	}
}

func handleConn(conn net.Conn) error {
	buf := make([]byte, 4096)

	for {
		if err := conn.SetDeadline(time.Now().Add(readTimeout)); err != nil {
			return err
		}

		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return io.EOF
			}

			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return ne
			}
			return err
		}

		if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			return err
		}

		if err := writeAll(conn, buf[:n]); err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return ne
			}
			return err
		}
	}
}

func writeAll(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		p = p[n:]
	}
	return nil
}
