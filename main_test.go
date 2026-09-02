package main

import (
	"bytes"
	"io"
	"net"
	"testing"
)

func setupTestConnection(t *testing.T) net.Conn {
	t.Helper()

	// create a server using net.Listen
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	t.Cleanup(func() {
		ln.Close()
	})

	// run the server in a goroutine
	go serve(ln)

	// connect to that server using net.Dial
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

func TestServerAcceptsClientAndEchoesBytes(t *testing.T) {
	// arrange
	conn := setupTestConnection(t)
	want := []byte("One Piece")

	// act
	n, err := conn.Write(want)

	// check writing errors
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len(want) {
		t.Fatalf("wrote %d bytes, want %d bytes", n, len(want))
	}

	got := make([]byte, len(want))

	// check reading errors
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo failed: %v", err)
	}

	// assert
	if !bytes.Equal(got, want) {
		t.Errorf("received %q bytes, want %q bytes", got, want)
	}
}

func TestServerEchoesDataLargerThanReadBuffer(t *testing.T) {
	conn := setupTestConnection(t)
	want := bytes.Repeat([]byte("A"), 4097)

	n, err := io.Copy(conn, bytes.NewReader(want))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != int64(len(want)) {
		t.Fatalf("wrote %d bytes, want %d bytes", n, len(want))
	}

	got := make([]byte, len(want))

	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo failed: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("echoed data does not match sent data")
	}
}
