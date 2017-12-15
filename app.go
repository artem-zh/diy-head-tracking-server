// MIT License
//
// Copyright (c) 2017 Artem Zhuravsky
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

type Entry struct {
	x, y, z float64
}

type Subscriber struct {
	ch     chan Entry
	active bool
}

var subscribers = make([]*Subscriber, 0, 30)

func multiply_matrices(a []float64, b [][]float64) []float64 {
	var num_cols = len(a)
	var num_rows = len(b[0])

	res := make([]float64, num_cols)
	for c := 0; c < num_cols; c++ {
		res[c] = 0
		for r := 0; r < num_rows; r++ {
			res[c] += a[r] * b[c][r]
		}
	}
	return res
}

func multiply_matrices_2d(a [][]float64, b [][]float64) [][]float64 {
	var num_cols = len(a)
	var num_rows = len(b[0])

	res := make([][]float64, num_cols)
	for c := 0; c < num_cols; c++ {
		res[c] = make([]float64, num_rows)
		for r := 0; r < num_rows; r++ {
			res[c][r] = 0
			for k := 0; k < len(b[r]); k++ {
				res[c][r] += a[k][c] * b[r][k]
			}

		}
	}
	return res
}

func transpose(mtx [][]float64) [][]float64 {
	var rows = len(mtx)
	var cols = len(mtx[0])

	res := make([][]float64, cols)
	for c := 0; c < cols; c++ {
		res[c] = make([]float64, rows)
		for r := 0; r < rows; r++ {
			res[c][r] = mtx[r][c]
		}
	}
	return res
}

func float64ToBuf(f float64, buf *bytes.Buffer) {
	err := binary.Write(buf, binary.LittleEndian, f)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
}

func handleTcpConnection(c net.Conn, sync chan bool) {
	subs := Subscriber{ch: make(chan Entry, 30), active: true}
	subscribers = append(subscribers, &subs)

	for {
		e := <-subs.ch

		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, uint8(3*8))
		float64ToBuf(e.x, &buf)
		float64ToBuf(e.y, &buf)
		float64ToBuf(e.z, &buf)

		_, err := c.Write(buf.Bytes())
		if err != nil {
			fmt.Println("Couldn't write TCP: ", err)
			break
		}
	}
	sync <- false
	subs.active = false
}

func tcpServer(input chan Entry, sync chan bool) {
	ln, err := net.Listen("tcp", ":3001")
	if err != nil {
		fmt.Println("Coudln't start listening TCP: ", err)
	}
	log.Println("Listening TCP on 3001...")
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		}
		log.Println("Accepted new TCP connection...")
		sync <- true
		go handleTcpConnection(conn, sync)
	}
}

func main() {
	data := make(chan Entry, 10)
	sync := make(chan bool, 10)

	//var prevTime = time.Unix(0, 0)
	//var elapsed, _ = time.ParseDuration("0ms")

	go func() {
		for {
			e := <-data
			for _, subs := range subscribers {
				if subs.active {
					subs.ch <- e
				}
			}
		}
	}()
	go tcpServer(data, sync)
	StartPhoneGyroWebServer(data, sync) // It blocks, hence it must be the last line.
}
