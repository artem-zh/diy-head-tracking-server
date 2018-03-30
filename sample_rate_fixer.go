package main

import ()

// http://math.oregonstate.edu/~restrepo/475A/Notes/sourcea-/node35.html
func interpolate(data [][]float64) [][]float64 {
	n := len(data) - 1
	a := make([]float64, n+1)
	alpha := make([]float64, n+1)
	b := make([]float64, n+1)
	c := make([]float64, n+1)
	d := make([]float64, n+1)
	for i := 0; i <= n; i++ {
		a[i] = data[i][1]
	}
	// 1.
	h := make([]float64, n+1)
	for i := 0; i < n; i++ {
		h[i] = data[i+1][0] - data[i][0]
	}
	// 2.
	for i := 1; i < n; i++ {
		//a[i] = 3.0*(a[i+1]-a[i])/h[i] - 3.0*(a[i]-a[i-1])/h[i-1]
		alpha[i] = 3.0*(a[i+1]-a[i])/h[i] - 3.0*(a[i]-a[i-1])/h[i-1]
	}
	// 3.
	l := make([]float64, n+1)
	nu := make([]float64, n+1)
	z := make([]float64, n+1)
	l[0] = 1
	nu[0] = 0
	z[0] = 0
	// 4.
	for i := 1; i < n; i++ {
		l[i] = 2.0*(data[i+1][0]-data[i-1][0]) - h[i-1]*nu[i-1]
		nu[i] = h[i] / l[i]
		z[i] = (alpha[i] - h[i-1]*z[i-1]) / l[i]
	}
	// 5.
	l[n] = 1
	//z[n] = 0 // ???
	c[n] = 0
	// 6.
	for j := n - 1; j >= 0; j-- {
		c[j] = z[j] - nu[j]*c[j+1]
		b[j] = (a[j+1]-a[j])/h[j] - h[j]*(c[j+1]+2*c[j])/3
		d[j] = (c[j+1] - c[j]) / (3 * h[j])
	}

	num_steps := 24

	output := make([][]float64, num_steps-1)

	step := (data[n][0] - data[0][0]) / float64(num_steps)
	cur := 0
	cur_x := data[cur][0]

	for i := 0; i < (num_steps - 1); i++ {
		hc := cur_x - data[cur][0]
		s := a[cur] + b[cur]*hc + c[cur]*hc*hc + d[cur]*hc*hc*hc
		output[i] = make([]float64, 2)
		output[i][0] = cur_x
		output[i][1] = s
		cur_x += step
		// adjusting, if needed, to corresponding spline
		for cur < n && data[cur+1][0] < cur_x {
			cur++
		}
	}
	return output
}
