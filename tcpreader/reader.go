// Copyright 2024-2025 NetCracker Technology Corporation
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
	"bufio"
	"bytes"
	"errors"
	"log"
	"net"
	"sync"
)

type TCPReader struct {
	Addr string // Represents the address of an endpoint

	listener net.Listener
}

// Listen announces on the local network address.
//
// The network must be "tcp".
//
// If the host in the address parameter is empty, an error is returned.
// If the port in the address parameter is "0", as in
// "127.0.0.1:" or "[::1]:0", a port number is automatically chosen.
func (tcpReader *TCPReader) Listen() error {
	addr := tcpReader.Addr
	if addr == "" {
		return errors.New("the listen address must not be empty")
	}
	log.Print("[INFO] Starting listen on: ", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	} else {
		tcpReader.listener = listener
		return nil
	}
}

// Waits for the next call, accepts the incoming call
// and returns a generic net.Conn.
func (tcpReader *TCPReader) AcceptConn() (net.Conn, error) {
	newConn, err := tcpReader.listener.Accept()
	if newConn != nil {
		log.Print("[INFO] Accepted connection from: ", newConn.RemoteAddr())
		return newConn, nil
	}
	return nil, err
}

// Reads a packet from the connection,
// copying the payload into *buff.
func (tcpReader *TCPReader) Read(newConn net.Conn, mu *sync.Mutex, buff *[]byte) {
	scanner := bufio.NewScanner(bufio.NewReader(newConn))
	if err := scanner.Err(); err != nil {
		log.Print("[WARN] Read error: ", err.Error())
	}
	scanner.Split(splitByZeroByte)
	for scanner.Scan() {
		mu.Lock()
		*buff = append(*buff, scanner.Bytes()...)
		mu.Unlock()
	}
}

func splitByZeroByte(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	} else if i := bytes.Index(data, []byte{0}); i >= 0 {
		return i + 1, data[0 : i+1], nil
	} else if atEOF {
		return len(data), data, nil
	} else {
		return
	}
}
