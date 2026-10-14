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

// source: https://stackoverflow.com/questions/30688685/how-does-one-test-net-conn-in-unit-tests-in-golang
func TestHandleConnectionTimesOutOnInactiveConnection(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	channelErr := make(chan error, 1)

	go func() {
		channelErr <- handleConn(serverConn, 50*time.Millisecond, time.Second)
	}()

	select {
	case err := <-channelErr:
		if err == nil {
			t.Fatal("expected inactivity timeout")
		}

		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatalf("expected timeout error, got: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("handleConn did not return after inactivity timeout")
	}
}

func TestHandleConnectionResetsInactivityDeadlineAfterReceivingData(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	if err := clientConn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client read deadline: %v", err)
	}

	go func() {
		handleConn(serverConn, 100*time.Millisecond, time.Second)
	}()

	exchange := func(want []byte) {
		t.Helper()

		n, err := clientConn.Write(want)
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if n != len(want) {
			t.Fatalf("wrote %d bytes, want %d bytes", n, len(want))
		}

		got := make([]byte, len(want))

		if _, err := io.ReadFull(clientConn, got); err != nil {
			t.Fatalf("read echo failed: %v", err)
		}

		if !bytes.Equal(got, want) {
			t.Errorf("echoed data does not match sent data")
		}
	}

	time.Sleep(60 * time.Millisecond)
	exchange([]byte("first"))

	time.Sleep(60 * time.Millisecond)
	exchange([]byte("second"))
}

func TestHandleConnHandlesClientDisconnect(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	done := make(chan error, 1)

	go func() {
		done <- handleConn(serverConn, 50*time.Millisecond, time.Second)
	}()

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("expected io.EOF after client disconnect, got: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("handleConn did not return after client disconnected")
	}
}
