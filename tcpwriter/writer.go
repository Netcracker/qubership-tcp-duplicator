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

package tcpwriter

import (
	"log"
	"net"
	"sync"
)

type TCPWriter struct {
	Addr *net.TCPAddr // Represents the address of a TCP endpoint

	connection *net.TCPConn
}

func (tcpWriter *TCPWriter) openConnection() {
	conn, err := net.DialTCP("tcp", nil, tcpWriter.Addr)
	if err != nil {
		log.Print("[WARN] Dial failed: ", err.Error())
	} else {
		log.Print("[INFO] Connection established successfully for ", tcpWriter.Addr)
		tcpWriter.connection = conn
	}
}

func (tcpWriter *TCPWriter) closeConnection() {
	if tcpWriter.connection != nil {
		log.Print("[INFO] Closing connection: ", tcpWriter.Addr)
		err := tcpWriter.connection.Close()
		if err != nil {
			log.Print("[WARN] Connection close error: ", err.Error())
		}
	}
}

func (tcpWriter *TCPWriter) write(bytes *[]byte, retryCount *int, wg *sync.WaitGroup) {
	defer wg.Done()
	if tcpWriter.connection == nil {
		tcpWriter.openConnection()
	}
	if tcpWriter.connection != nil {
		for i := 0; i < *retryCount; i++ {
			err := tcpWriter.doWrite(bytes)
			if err != nil {
				log.Print("[WARN] Write to server failed: ", err.Error())
				tcpWriter.openConnection()
			} else {
				break
			}
		}
	}
}

func (tcpWriter *TCPWriter) doWrite(bytes *[]byte) error {
	_, err := tcpWriter.connection.Write(*bytes)
	if err != nil {
		return err
	}
	return nil
}

type TCPWriteHandler struct {
	_writers []*TCPWriter
}

// Function to attach a new TCPWriter.
func (tcpWriteHandler *TCPWriteHandler) AttachWriter(tcpWriter TCPWriter) {
	tcpWriteHandler._writers = append(tcpWriteHandler._writers, &tcpWriter)
}

// Function to detach a TCPWriter.
func (tcpWriteHandler *TCPWriteHandler) DetachWriter(tcpWriter TCPWriter) {
	for i, writer := range tcpWriteHandler._writers {
		if writer.Addr == tcpWriter.Addr {
			if writer.connection != nil {
				writer.closeConnection()
			}
			tcpWriteHandler._writers = append(tcpWriteHandler._writers[0:i], tcpWriteHandler._writers[i+1:]...)
			break
		}
	}
}

// Flush writes a buffered data to the TCPWriter's.
func (tcpWriteHandler *TCPWriteHandler) FlushData(bytes *[]byte, retryCount *int) {
	var wg sync.WaitGroup
	for _, writer := range tcpWriteHandler._writers {
		wg.Add(1)
		go writer.write(bytes, retryCount, &wg)
	}
	wg.Wait()
}
