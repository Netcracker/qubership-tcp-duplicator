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

package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Netcracker/qubership-tcp-duplicator/converters"
	"github.com/Netcracker/qubership-tcp-duplicator/tcpreader"
	"github.com/Netcracker/qubership-tcp-duplicator/tcpwriter"
)

var (
	listenPort = func() string {
		value := os.Getenv("LISTEN_PORT")
		if len(value) == 0 {
			return ""
		}
		return ":" + value
	}()
	flushInterval = func() time.Duration {
		value := os.Getenv("FLUSH_INTERVAL")
		if len(value) == 0 {
			defaultDuration, _ := time.ParseDuration("10s")
			return defaultDuration
		}
		duration, err := time.ParseDuration(value)
		if err != nil {
			log.Fatal("[ERROR] Incorrect format for \"FLUSH_INTERVAL\" variable: ", err.Error())
		}
		return duration
	}()
	bufferLimitSize = func() uint64 {
		value := os.Getenv("BUFFER_LIMIT_SIZE")
		if len(value) == 0 {
			defaultBufferLimit, _ := converters.ToBytes("64MB")
			return defaultBufferLimit
		}
		bufferLimit, err := converters.ToBytes(value)
		if err != nil {
			log.Fatal("[ERROR] Incorrect format for \"BUFFER_LIMIT_SIZE\" variable: ", err.Error())
		}
		return bufferLimit
	}()
	retryCount = func() int {
		value := os.Getenv("RETRY_COUNT")
		if len(value) == 0 {
			return 3
		}
		number, err := strconv.Atoi(value)
		if err != nil {
			log.Fatal("[ERROR] Incorrect format for \"RETRY_COUNT\" variable: ", err.Error())
		} else if number <= 0 {
			log.Fatal("[ERROR] The value of the \"RETRY_COUNT\" variable must be positive/non-zero")
		}
		return number
	}()
	checkInterval = flushInterval / 5
)

type Payload struct {
	sync.Mutex
	data []byte
}

func NewPayload() *Payload {
	return &Payload{
		data: make([]byte, 0),
	}
}

func NewTCPReader(addr string) tcpreader.TCPReader {
	return tcpreader.TCPReader{
		Addr: addr,
	}
}

func NewTCPWriteHandler() tcpwriter.TCPWriteHandler {
	tcpWriteHandler := tcpwriter.TCPWriteHandler{}
	addresses := os.Getenv("TCP_ADDRESSES")
	if len(addresses) == 0 {
		log.Fatal("[ERROR] Incorrect value for \"TCP_ADDRESSES\" variable:")
	}
	for _, address := range strings.Split(addresses, ",") {
		writer := tcpwriter.TCPWriter{Addr: resolveTCPAddr(strings.TrimSpace(address))}
		tcpWriteHandler.AttachWriter(writer)
	}
	return tcpWriteHandler
}

func resolveTCPAddr(address string) *net.TCPAddr {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.Fatal(err.Error())
	}
	return addr
}

func main() {
	if listenPort == "" {
		log.Fatal("[ERROR] Incorrect value for \"LISTEN_PORT\" variable:")
	}

	payload := NewPayload()
	reader := NewTCPReader(listenPort)

	err := reader.Listen()
	if err != nil {
		log.Fatal("[ERROR] Listen error", err.Error())
	}

	tcpWriteHandler := NewTCPWriteHandler()

	go func() {
		for {
			if conn, err := reader.AcceptConn(); err == nil {
				go func() {
					reader.Read(conn, &payload.Mutex, &payload.data)
					err := conn.Close()
					log.Print("[INFO] Closing connection: ", conn.RemoteAddr())
					if err != nil {
						log.Print("[WARN] Closing error: ", err.Error())
					}
				}()
			} else {
				log.Print("[WARN] Connections accepting error: ", err.Error())
			}
		}
	}()

	go func() {
		nextFlush := time.Now().Add(flushInterval)
		for {
			payload.Lock()
			if dataLen := uint64(len(payload.data)); (time.Now().After(nextFlush) || dataLen >= bufferLimitSize) && dataLen > 0 {
				tcpWriteHandler.FlushData(&payload.data, &retryCount)
				payload.data = payload.data[:0]
				nextFlush = time.Now().Add(flushInterval)
			}
			payload.Unlock()
			time.Sleep(checkInterval)
		}
	}()

	select {}
}
