package tcpreader

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestListenRejectsEmptyAddress(t *testing.T) {
	reader := TCPReader{}
	if err := reader.Listen(); err == nil {
		t.Fatal("expected listen to reject empty address")
	}
}

func TestListenAndAcceptConn(t *testing.T) {
	reader := TCPReader{Addr: "127.0.0.1:0"}
	if err := reader.Listen(); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	t.Cleanup(func() {
		if err := reader.listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := net.Dial("tcp", reader.listener.Addr().String())
		if err != nil {
			t.Errorf("dial failed: %v", err)
			return
		}
		defer func() {
			if err := conn.Close(); err != nil {
				t.Errorf("close dialed conn: %v", err)
			}
		}()
	}()

	conn, err := reader.AcceptConn()
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	_ = conn.Close()
	<-done
}

func TestReadAppendsAllZeroDelimitedMessages(t *testing.T) {
	reader := TCPReader{}
	serverConn, clientConn := net.Pipe()
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Errorf("close server conn: %v", err)
		}
	}()

	var (
		mu   sync.Mutex
		buff []byte
		done = make(chan struct{})
	)

	go func() {
		defer close(done)
		reader.Read(serverConn, &mu, &buff)
	}()

	if _, err := clientConn.Write([]byte("one\x00two\x00tail")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	_ = clientConn.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not finish")
	}

	if got := string(buff); got != "one\x00two\x00tail" {
		t.Fatalf("unexpected buffered content %q", got)
	}
}

func TestSplitByZeroByte(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		atEOF       bool
		wantAdvance int
		wantToken   []byte
	}{
		{
			name:        "delimiter found",
			data:        []byte("abc\x00rest"),
			wantAdvance: 4,
			wantToken:   []byte("abc\x00"),
		},
		{
			name:        "end of file without delimiter",
			data:        []byte("tail"),
			atEOF:       true,
			wantAdvance: 4,
			wantToken:   []byte("tail"),
		},
		{
			name:  "empty eof",
			atEOF: true,
		},
		{
			name:      "need more data",
			data:      []byte("partial"),
			wantToken: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			advance, token, err := splitByZeroByte(tc.data, tc.atEOF)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if advance != tc.wantAdvance {
				t.Fatalf("unexpected advance: got %d want %d", advance, tc.wantAdvance)
			}
			if string(token) != string(tc.wantToken) {
				t.Fatalf("unexpected token: got %q want %q", token, tc.wantToken)
			}
		})
	}
}
