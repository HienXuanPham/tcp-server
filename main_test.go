package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"
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

func TestServerKeepsConnectionOpen(t *testing.T) {
	conn := setupTestConnection(t)
	messages := [][]byte{
		[]byte("One Piece"),
		[]byte("Luffy"),
		[]byte("Nami"),
	}

	for i, want := range messages {
		n, err := conn.Write(want)

		if err != nil {
			t.Fatalf("message %d: write failed: %v", i, err)
		}
		if n != len(want) {
			t.Fatalf("message %d: wrote %d bytes, want %d bytes", i, n, len(want))
		}

		got := make([]byte, len(want))

		if _, err := io.ReadFull(conn, got); err != nil {
			t.Fatalf("message %d: read echo failed: %v", i, err)
		}

		if !bytes.Equal(got, want) {
			t.Errorf("message %d: received %q, want %q", i, got, want)
		}
	}
}

func TestServerClosesInactiveConnection(t *testing.T) {
	conn := setupTestConnection(t)
	message := []byte("One Piece")
	got := make([]byte, len(message))
	time.Sleep(31 * time.Second)
	_, err := io.ReadFull(conn, got)

	if err == nil {
		t.Error("Expected connection to be closed by server timeout")
	} else if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Expected an EOF error, but got: %v", err)
	}
}
