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
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type RawDataEntry struct {
	alpha, beta, gamma float64
    ts_delta uint64
}

const (
	degToRad = math.Pi / 180.0
)

var (
	thePoint = []float64{0, 0, 1}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

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

func dataProcessor(raw_data chan RawDataEntry, sync chan bool, out_data chan Entry) {
	var latest_rotation_mtx [][]float64
	var rotation_ref_mtx [][]float64
	var rot_mtx_1 [][]float64
	var initialized = false
	var mtx1_initialized = false
//    var mirrorCoef float64 = 1.0
	for {
		select {
		case rd := <-raw_data:
			rot_mtx := buildRotationMatrix(rd.beta, rd.gamma, rd.alpha)
			latest_rotation_mtx = rot_mtx
			if !mtx1_initialized {
				rot_mtx_1 = buildRotationMatrix(rd.beta, 90*degToRad, rd.alpha)
				mtx1_initialized = true
			}

			//the_pointA := []float64{0, 0, 1}

			var pointRotated []float64
			//pointA3 := pointRotated

			if initialized {
				// It was an attempt to fix the issue with starting orientation...
				mtx := multiply_matrices_2d(transpose(rotation_ref_mtx), rot_mtx)
				mtx = multiply_matrices_2d(mtx, rot_mtx_1)
				pointRotated = multiply_matrices(thePoint, mtx)
			} else {
				pointRotated = multiply_matrices(thePoint, rot_mtx)
			}

			// Ignore result made in 'if initialized'...
			//pointRotated = multiply_matrices(thePoint, rot_mtx)

			c := math.Sqrt(pointRotated[0]*pointRotated[0] + pointRotated[1]*pointRotated[1])
			heading := math.Asin(pointRotated[1]/c) / math.Pi * 180

			c2 := math.Sqrt(pointRotated[0]*pointRotated[0] + pointRotated[2]*pointRotated[2])
			pitch := math.Asin(pointRotated[2]/c2) / math.Pi * 180

			//fmt.Printf("  Point: %.2f, %.2f, %.2f\n", pointRotated[0], pointRotated[1], pointRotated[2])
			//fmt.Printf("   Heading: %.3f Pitch: %.3f\n", heading, pitch)

			out_data <- Entry{x: heading, y: pitch, z: 0}

		case synced := <-sync:
			initialized = synced
			if synced {
				rotation_ref_mtx = latest_rotation_mtx
                //pointRotated := multiply_matrices(thePoint, rotation_ref_mtx)
                //mirrorCoef = 1.0
			    //if pointRotated[0] > 0 {
				//    mirrorCoef = -1.0
			    //}
				log.Println("Synced!")
			} else {
				log.Println("Unsynced")
			}
		}
	}
}

func StartPhoneGyroWebServer(outData chan Entry, sync chan bool) {
	//var signalStr string
	raw_gyro_data := make(chan RawDataEntry, 10)
    var prev_timestamp uint64 = 0

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
                unix_ts_s := str_values[4]

				alpha, _ := strconv.ParseFloat(alpha_s, 64)
				beta, _ := strconv.ParseFloat(beta_s, 64)
				gamma, _ := strconv.ParseFloat(gamma_s, 64)
                unix_ts, _ := strconv.ParseUint(unix_ts_s, 10, 64)
				//orient, _ := strconv.ParseFloat(orient_s, 64)

                var ts_delta uint64 = 0
                if prev_timestamp > 0 {
                    ts_delta = unix_ts - prev_timestamp
                }
                prev_timestamp = unix_ts

				fmt.Printf("   raw: %.1f, %.1f, %.1f [%d]\n", alpha, beta, gamma, ts_delta)

				e := RawDataEntry{alpha: alpha * degToRad, beta: beta * degToRad, gamma: gamma * degToRad, ts_delta: ts_delta}
				raw_gyro_data <- e
			}
		}
	})

	go dataProcessor(raw_gyro_data, sync, outData)

	log.Println("Listening http on 3000...")
	// TODO !! It blocks, and it's the only reason the app is working. Refactor it!
	http.ListenAndServe(":3000", nil)
}
