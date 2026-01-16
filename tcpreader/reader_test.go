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

package tcpreader

import (
	"bytes"
	"net"
	"sync"
	"testing"
)

func TestSplitByZeroByte(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		atEOF    bool
		expected struct {
			advance int
			token   []byte
			err     error
		}
	}{
		{
			name:  "empty data at EOF",
			data:  []byte{},
			atEOF: true,
			expected: struct {
				advance int
				token   []byte
				err     error
			}{0, nil, nil},
		},
		{
			name:  "data with zero byte",
			data:  []byte{1, 2, 3, 0, 4, 5},
			atEOF: false,
			expected: struct {
				advance int
				token   []byte
				err     error
			}{4, []byte{1, 2, 3, 0}, nil},
		},
		{
			name:  "data without zero byte at EOF",
			data:  []byte{1, 2, 3},
			atEOF: true,
			expected: struct {
				advance int
				token   []byte
				err     error
			}{3, []byte{1, 2, 3}, nil},
		},
		{
			name:  "data without zero byte not at EOF",
			data:  []byte{1, 2, 3},
			atEOF: false,
			expected: struct {
				advance int
				token   []byte
				err     error
			}{0, nil, nil},
		},
		{
			name:  "zero byte at start",
			data:  []byte{0, 1, 2},
			atEOF: false,
			expected: struct {
				advance int
				token   []byte
				err     error
			}{1, []byte{0}, nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advance, token, err := splitByZeroByte(tt.data, tt.atEOF)
			if advance != tt.expected.advance {
				t.Errorf("advance: got %d, want %d", advance, tt.expected.advance)
			}
			if !bytes.Equal(token, tt.expected.token) {
				t.Errorf("token: got %v, want %v", token, tt.expected.token)
			}
			if err != tt.expected.err {
				t.Errorf("err: got %v, want %v", err, tt.expected.err)
			}
		})
	}
}

func TestTCPReader_Listen(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "empty address",
			addr:    "",
			wantErr: true,
		},
		{
			name:    "valid address",
			addr:    "127.0.0.1:0", // Use port 0 for automatic assignment
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := TCPReader{Addr: tt.addr}
			err := reader.Listen()
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPReader.Listen() error = %v, wantErr %v", err, tt.wantErr)
			}
			if reader.listener != nil {
				reader.listener.Close()
			}
		})
	}
}

func TestTCPReader_Read(t *testing.T) {
	// Use net.Pipe to create connected pipes for testing
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := TCPReader{}
	var mu sync.Mutex
	var data []byte

	// Write test data to clientConn
	testData := []byte("hello\x00world\x00")
	go func() {
		clientConn.Write(testData)
		clientConn.Close()
	}()

	// Read from serverConn
	reader.Read(serverConn, &mu, &data)

	mu.Lock()
	if !bytes.Equal(data, testData) {
		t.Errorf("Read data = %v, want %v", data, testData)
	}
	mu.Unlock()
}

func TestTCPReader_AcceptConn(t *testing.T) {
	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	reader := TCPReader{listener: listener}

	// Connect in a goroutine
	go func() {
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
	}()

	// Accept connection
	conn, err := reader.AcceptConn()
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
