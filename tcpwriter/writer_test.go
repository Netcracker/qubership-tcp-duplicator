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

package tcpwriter

import (
	"net"
	"reflect"
	"testing"
)

func TestTCPWriteHandler_AttachWriter(t *testing.T) {
	handler := TCPWriteHandler{}
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	writer := TCPWriter{Addr: addr}

	handler.AttachWriter(writer)

	if len(handler._writers) != 1 {
		t.Errorf("Expected 1 writer, got %d", len(handler._writers))
	}
	if !reflect.DeepEqual(handler._writers[0].Addr, addr) {
		t.Errorf("Writer not attached correctly")
	}
}

func TestTCPWriteHandler_DetachWriter(t *testing.T) {
	handler := TCPWriteHandler{}
	addr1, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	addr2, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8081")
	writer1 := TCPWriter{Addr: addr1}
	writer2 := TCPWriter{Addr: addr2}

	handler.AttachWriter(writer1)
	handler.AttachWriter(writer2)

	if len(handler._writers) != 2 {
		t.Errorf("Expected 2 writers, got %d", len(handler._writers))
	}

	handler.DetachWriter(writer1)

	if len(handler._writers) != 1 {
		t.Errorf("Expected 1 writer after detach, got %d", len(handler._writers))
	}
	if !reflect.DeepEqual(handler._writers[0].Addr, addr2) {
		t.Errorf("Wrong writer remaining after detach")
	}
}

func TestTCPWriteHandler_DetachWriter_NotFound(t *testing.T) {
	handler := TCPWriteHandler{}
	addr1, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	addr2, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8081")
	writer1 := TCPWriter{Addr: addr1}
	writer3 := TCPWriter{Addr: addr2} // Not attached

	handler.AttachWriter(writer1)

	handler.DetachWriter(writer3) // Should not panic or change

	if len(handler._writers) != 1 {
		t.Errorf("Expected 1 writer, got %d", len(handler._writers))
	}
}

func TestTCPWriteHandler_FlushData(t *testing.T) {
	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	// Accept connections in a goroutine
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		if n > 0 {
			// Verify data was received
			if string(buf[:n]) != "test data" {
				t.Errorf("Received wrong data: %s", string(buf[:n]))
			}
		}
	}()

	// Create handler and writer
	handler := TCPWriteHandler{}
	addr, _ := net.ResolveTCPAddr("tcp", serverAddr)
	writer := TCPWriter{Addr: addr}
	handler.AttachWriter(writer)

	// Flush data
	data := []byte("test data")
	retryCount := 1
	handler.FlushData(&data, &retryCount)

	// Detach to cover closeConnection
	handler.DetachWriter(writer)
}
