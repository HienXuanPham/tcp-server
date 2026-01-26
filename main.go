package main

import (
	"net"
	"log"
	"time"
	"io"
)

func main() {
	listener, lerr := net.Listen("tcp", ":8080")
	if lerr != nil {
		log.Fatal("Error listening:", lerr)
	}
	defer listener.Close()

	conn, aerr := listener.Accept()
	if aerr != nil {
		log.Fatal("Error accepting conn:", aerr)
	}

	log.Println("Accepted connection")
	log.Println(conn.RemoteAddr().String())
	log.Println(conn.LocalAddr().String())
		
	defer conn.Close()

	const readTimeout = 30 * time.Second
	const writeTimeout = 30 * time.Second
	buf := make([]byte, 1024)

	for {
		rterr := conn.SetReadDeadline(time.Now().Add(readTimeout))
		if rterr != nil {
			log.Println("SetReadDeadline failed", rterr)
			break
		}

		n, rerr := conn.Read(buf)

		if rerr != nil {
			if rerr == io.EOF {
				log.Println("EOF")
				break
			} else {
				log.Println("Read error:", rerr)
			}
			break
		}

		data := buf[:n]

		wterr := conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if wterr != nil {
			log.Println("SetWriteDeadline failed", wterr)
			break
		}

		_, werr := conn.Write(data)

		if werr != nil {
			log.Println("Write failed", werr)
			break
		}

	}
}