package main

import (
	"errors"
	"fmt"
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

	if err := serve(listener); err != nil {
		log.Printf("server ended: %v", err)
	}
}

func serve(ln net.Listener) error {
	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("accept failed: %w", err)
	}
	defer conn.Close()

	log.Printf("accepted remote=%s local=%s", conn.RemoteAddr(), conn.LocalAddr())

	if err := handleConn(conn, readTimeout, writeTimeout); err != nil {
		return fmt.Errorf("handle connection: %w", err)
	}

	return nil
}

func handleConn(
	conn net.Conn,
	readTimeout time.Duration,
	writeTimeout time.Duration,
) error {
	buf := make([]byte, 4096)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			return err
		}

		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
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
		if n == 0 {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}
