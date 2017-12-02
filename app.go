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
	"github.com/gorilla/websocket"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Entry struct {
	x, y, z float64
}

type Subscriber struct {
	ch     chan Entry
	active bool
}

const (
	degToRad = math.Pi / 180.0
)

var (
	thePoint = []float64{0, 0, 1}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var subscribers = make([]*Subscriber, 0, 30)

func buildRotationMatrix(x, y, z float64) [][]float64 {
	cx := math.Cos(x)
	cy := math.Cos(y)
	cz := math.Cos(z)
	sx := math.Sin(x)
	sy := math.Sin(y)
	sz := math.Sin(z)

	m11 := cz*cy - sz*sx*sy
	m12 := -cx * sz
	m13 := cy*sz*sx + cz*sy

	m21 := cy*sz + cz*sx*sy
	m22 := cz * cx
	m23 := sz*sy - cz*cy*sx

	m31 := -cx * sy
	m32 := sx
	m33 := cx * cy

	return [][]float64{
		{m11, m12, m13},
		{m21, m22, m23},
		{m31, m32, m33},
	}
}

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

		//out_buf_size := buf.Len()
		out_buf := buf.Bytes()

		_, err := c.Write(out_buf)
		if err != nil {
			fmt.Println("Couldn't write TCP: ", err)
			break
		} //else {
		//	fmt.Printf("Wrote %d bytes to TCP out of %d\n", num, out_buf_size)
		//}
	}
	sync <- false
	subs.active = false
}

func tcpServer(input chan Entry, sync chan bool) {
	ln, err := net.Listen("tcp", ":3001")
	if err != nil {
		fmt.Println("Coudln't listen TCP: ", err)
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

func dataProcessor(raw_data chan Entry, sync chan bool, out_data chan Entry) {
	var latest_rotation_mtx [][]float64
	var rotation_ref_mtx [][]float64
	var rot_mtx_1 [][]float64
	var initialized = false
	var mtx1_initialized = false
	for {
		select {
		case rd := <-raw_data:
			rot_mtx := buildRotationMatrix(rd.x, rd.y, rd.z)
			latest_rotation_mtx = rot_mtx
			if !mtx1_initialized {
				rot_mtx_1 = buildRotationMatrix(rd.x, 90*degToRad, rd.z)
				mtx1_initialized = true
			}

			//the_pointA := []float64{0, 0, 1}

			var pointA2 []float64
			//pointA3 := pointA2
			if initialized {
				//ref_mtx := *rotation_ref_mtx
				//ref_t := transpose(rotation_ref_mtx)
				//fmt.Printf("    %v", rotation_ref_mtx)
				//rot_mtx = multiply_matrices_2d(rot_mtx, transpose(rot_mtx))
				//fmt.Printf("    %v", rot_mtx)
				//rot_mtx = rotation_ref_mtx

				//pointA2 = multiply_matrices(the_pointA, transpose(rotation_ref_mtx))
				//pointA2 = multiply_matrices(pointA2, transpose(rot_mtx_1))
				//pointA2 = multiply_matrices(pointA2, rot_mtx)
				mtx := multiply_matrices_2d(transpose(rotation_ref_mtx), rot_mtx)
				mtx = multiply_matrices_2d(mtx, rot_mtx_1)
				pointA2 = multiply_matrices(thePoint, mtx)
				//pointA2 = multiply_matrices(pointA2, transpose(rotation_ref_mtx))
				//pointA2 = multiply_matrices(pointA2, rot_mtx_1)

				//rot_mtx = multiply_matrices_2d(transpose(rot_mtx), rotation_ref_mtx)
			} else {
				pointA2 = multiply_matrices(thePoint, rot_mtx)
			}

			//pointA2 = pointA3

			c := math.Sqrt(pointA2[0]*pointA2[0] + pointA2[1]*pointA2[1])
			heading := math.Asin(pointA2[1]/c) / math.Pi * 180

			c2 := math.Sqrt(pointA2[0]*pointA2[0] + pointA2[2]*pointA2[2])
			pitch := math.Asin(pointA2[2]/c2) / math.Pi * 180

			//fmt.Printf("  Point: A: %.2f, %.2f, %.2f\n", pointA2[0], pointA2[1], pointA2[2])
			fmt.Printf("   Heading: %.3f Pitch: %.3f\n", heading, pitch)

			out_data <- Entry{x: heading, y: pitch, z: 0}

		case synced := <-sync:
			initialized = synced
			if synced {
				rotation_ref_mtx = latest_rotation_mtx
				log.Println("Synced!")
			} else {
				log.Println("Unsynced")
			}
		}
	}
}

func main() {
	data := make(chan Entry, 10)
	raw_gyro_data := make(chan Entry, 10)
	sync := make(chan bool, 10)

	//var prevTime = time.Unix(0, 0)
	//var elapsed, _ = time.ParseDuration("0ms")

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		log.Println("WS Connected!!")
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println(err)
				return
			}
			if string(msg) == "close" {
				conn.Close()
				fmt.Println("Stopping WS")
				return
			} else {
				str_msg := string(msg)
				str_values := strings.Split(str_msg, "/")
				beta_s, gamma_s, alpha_s := str_values[0], str_values[1], str_values[2]
				alpha, _ := strconv.ParseFloat(alpha_s, 64)
				beta, _ := strconv.ParseFloat(beta_s, 64)
				gamma, _ := strconv.ParseFloat(gamma_s, 64)
				//orient, _ := strconv.ParseFloat(orient_s, 64)

				//fmt.Printf("   raw: %.1f, %.1f, %.1f\n", alpha, beta, gamma)

				x := beta * degToRad
				y := gamma * degToRad
				z := alpha * degToRad

				e := Entry{x: x, y: y, z: z}
				raw_gyro_data <- e
			}
		}
	})

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
	go dataProcessor(raw_gyro_data, sync, data)

	log.Println("Listening...")
	http.ListenAndServe(":3000", nil)
}
