package tcpwriter

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestAttachAndDetachWriter(t *testing.T) {
	handler := TCPWriteHandler{}
	first := TCPWriter{Addr: mustResolveTCPAddr(t, "127.0.0.1:9101")}
	second := TCPWriter{Addr: mustResolveTCPAddr(t, "127.0.0.1:9102")}

	handler.AttachWriter(first)
	handler.AttachWriter(second)
	handler.DetachWriter(first)

	if len(handler._writers) != 1 {
		t.Fatalf("expected one writer after detach, got %d", len(handler._writers))
	}
	if got := handler._writers[0].Addr.String(); got != "127.0.0.1:9102" {
		t.Fatalf("unexpected remaining writer %q", got)
	}
}

func TestOpenConnectionAndCloseConnection(t *testing.T) {
	listener, accepted := startTCPServer(t)
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	}()

	writer := TCPWriter{Addr: listener.Addr().(*net.TCPAddr)}
	writer.openConnection()
	if writer.connection == nil {
		t.Fatal("expected connection to be opened")
	}

	writer.closeConnection()

	select {
	case conn := <-accepted:
		_ = conn.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("expected server to accept connection")
	}
}

func TestOpenConnectionFailureLeavesConnectionNil(t *testing.T) {
	listener, _ := startTCPServer(t)
	addr := listener.Addr().(*net.TCPAddr)
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	writer := TCPWriter{Addr: addr}
	writer.openConnection()

	if writer.connection != nil {
		t.Fatal("expected connection to remain nil after dial failure")
	}
}

func TestCloseConnectionWithNilConnection(t *testing.T) {
	writer := TCPWriter{}
	writer.closeConnection()
}

func TestDoWriteWritesBytes(t *testing.T) {
	listener, accepted := startTCPServer(t)
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	}()

	writer := TCPWriter{Addr: listener.Addr().(*net.TCPAddr)}
	writer.openConnection()
	if writer.connection == nil {
		t.Fatal("expected connection to be opened")
	}
	defer writer.closeConnection()

	serverConn := <-accepted
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Errorf("close server conn: %v", err)
		}
	}()

	payload := []byte("hello writer")
	if err := writer.doWrite(&payload); err != nil {
		t.Fatalf("doWrite failed: %v", err)
	}

	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(serverConn, buf); err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(buf) != string(payload) {
		t.Fatalf("unexpected payload %q", buf)
	}
}

func TestDoWriteReturnsErrorForClosedConnection(t *testing.T) {
	listener, accepted := startTCPServer(t)
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	}()

	writer := TCPWriter{Addr: listener.Addr().(*net.TCPAddr)}
	writer.openConnection()
	if writer.connection == nil {
		t.Fatal("expected connection to be opened")
	}

	serverConn := <-accepted
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Errorf("close server conn: %v", err)
		}
	}()

	writer.closeConnection()

	payload := []byte("closed")
	if err := writer.doWrite(&payload); err == nil {
		t.Fatal("expected write to fail after connection close")
	}
}

func TestWriteOpensConnectionOnDemand(t *testing.T) {
	listener, accepted := startTCPServer(t)
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	}()

	writer := TCPWriter{Addr: listener.Addr().(*net.TCPAddr)}
	payload := []byte("flush me")
	retryCount := 1
	var wg sync.WaitGroup

	wg.Add(1)
	go writer.write(&payload, &retryCount, &wg)

	serverConn := <-accepted
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Errorf("close server conn: %v", err)
		}
	}()

	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(serverConn, buf); err != nil {
		t.Fatalf("read failed: %v", err)
	}

	wg.Wait()
	if writer.connection == nil {
		t.Fatal("expected write to establish a connection")
	}
}

func TestWriteSkipsWhenConnectionCannotBeOpened(t *testing.T) {
	listener, _ := startTCPServer(t)
	addr := listener.Addr().(*net.TCPAddr)
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	writer := TCPWriter{Addr: addr}
	payload := []byte("flush me")
	retryCount := 1
	var wg sync.WaitGroup

	wg.Add(1)
	go writer.write(&payload, &retryCount, &wg)
	wg.Wait()

	if writer.connection != nil {
		t.Fatal("expected connection to stay nil when dial fails")
	}
}

func TestFlushDataWritesToAllWriters(t *testing.T) {
	listener1, accepted1 := startTCPServer(t)
	defer func() {
		if err := listener1.Close(); err != nil {
			t.Errorf("close listener1: %v", err)
		}
	}()
	listener2, accepted2 := startTCPServer(t)
	defer func() {
		if err := listener2.Close(); err != nil {
			t.Errorf("close listener2: %v", err)
		}
	}()

	handler := TCPWriteHandler{}
	handler.AttachWriter(TCPWriter{Addr: listener1.Addr().(*net.TCPAddr)})
	handler.AttachWriter(TCPWriter{Addr: listener2.Addr().(*net.TCPAddr)})

	payload := []byte("broadcast")
	retryCount := 1

	done1 := make(chan []byte, 1)
	done2 := make(chan []byte, 1)
	go capturePayload(t, accepted1, len(payload), done1)
	go capturePayload(t, accepted2, len(payload), done2)

	handler.FlushData(&payload, &retryCount)

	if got := <-done1; string(got) != string(payload) {
		t.Fatalf("unexpected payload for first writer %q", got)
	}
	if got := <-done2; string(got) != string(payload) {
		t.Fatalf("unexpected payload for second writer %q", got)
	}
}

func startTCPServer(t *testing.T) (*net.TCPListener, <-chan *net.TCPConn) {
	t.Helper()

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	accepted := make(chan *net.TCPConn, 1)
	go func() {
		conn, err := listener.AcceptTCP()
		if err == nil {
			accepted <- conn
		}
	}()

	return listener, accepted
}

func capturePayload(t *testing.T, accepted <-chan *net.TCPConn, size int, done chan<- []byte) {
	t.Helper()

	select {
	case conn := <-accepted:
		defer func() {
			if err := conn.Close(); err != nil {
				t.Errorf("close accepted conn: %v", err)
			}
		}()
		buf := make([]byte, size)
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Errorf("read failed: %v", err)
			return
		}
		done <- buf
	case <-time.After(2 * time.Second):
		t.Errorf("timed out waiting for connection")
	}
}

func mustResolveTCPAddr(t *testing.T, address string) *net.TCPAddr {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	return addr
}
