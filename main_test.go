// Copyright NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

// TestMain sets a free listenPort so the package-level var (which returns ""
// when LISTEN_PORT is unset) does not cause log.Fatal inside main() if any
// test accidentally triggers it without a port.
func TestMain(m *testing.M) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("TestMain: could not find free port: " + err.Error())
	}
	listenPort = fmt.Sprintf(":%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	os.Exit(m.Run())
}

func TestNewPayload(t *testing.T) {
	p := NewPayload()
	if p == nil {
		t.Fatal("NewPayload returned nil")
	}
	if len(p.data) != 0 {
		t.Errorf("expected empty payload data, got %d bytes", len(p.data))
	}
}

func TestNewTCPReader(t *testing.T) {
	const addr = ":12345"
	r := NewTCPReader(addr)
	if r.Addr != addr {
		t.Errorf("expected Addr %q, got %q", addr, r.Addr)
	}
}

func TestResolveTCPAddr(t *testing.T) {
	addr := resolveTCPAddr("127.0.0.1:9999")
	if addr == nil {
		t.Fatal("expected non-nil TCPAddr")
	}
	if addr.Port != 9999 {
		t.Errorf("expected port 9999, got %d", addr.Port)
	}
	if addr.IP.String() != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", addr.IP)
	}
}

func TestNewTCPWriteHandler(t *testing.T) {
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	t.Setenv("TCP_ADDRESSES", target.Addr().String())

	// Should not panic or call log.Fatal.
	NewTCPWriteHandler()
}

func TestNewTCPWriteHandlerMultipleAddresses(t *testing.T) {
	t1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer t1.Close()

	t2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer t2.Close()

	t.Setenv("TCP_ADDRESSES", t1.Addr().String()+","+t2.Addr().String())

	NewTCPWriteHandler()
}

// TestMainIntegration starts the duplicator via main(), sends null-terminated
// data to its listen port, and verifies the data reaches the mock target.
func TestMainIntegration(t *testing.T) {
	// Mock target server that collects the first chunk of received data.
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("mock target listen:", err)
	}
	defer targetLn.Close()

	received := make(chan []byte, 1)
	go func() {
		conn, err := targetLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		received <- buf[:n]
	}()

	// Find a free port for the duplicator's listen side.
	dupLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("find free port:", err)
	}
	dupPort := dupLn.Addr().(*net.TCPAddr).Port
	dupLn.Close()

	// Override package-level vars for this test before starting main().
	listenPort = fmt.Sprintf(":%d", dupPort)
	flushInterval = 100 * time.Millisecond
	checkInterval = 20 * time.Millisecond
	t.Setenv("TCP_ADDRESSES", targetLn.Addr().String())

	go main()

	// Poll until the duplicator's listener is ready.
	var src net.Conn
	for i := 0; i < 100; i++ {
		src, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", dupPort))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal("connect to duplicator:", err)
	}
	defer src.Close()

	// tcpreader splits on null bytes, so terminate the message with \x00.
	msg := append([]byte("hello-integration-test"), 0)
	if _, err := src.Write(msg); err != nil {
		t.Fatal("write to duplicator:", err)
	}

	select {
	case data := <-received:
		if !bytes.Contains(data, []byte("hello-integration-test")) {
			t.Errorf("target received %q, want it to contain %q", data, "hello-integration-test")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for data at mock target")
	}
}
