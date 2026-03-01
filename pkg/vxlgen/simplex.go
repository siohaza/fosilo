package vxlgen

import "math"

type SimplexNoise struct {
	perm [256]int
}

func NewSimplexNoise(rng *Rng) *SimplexNoise {
	s := &SimplexNoise{}
	for i := 0; i < 256; i++ {
		s.perm[i] = i
	}
	for i := 0; i < 256; i++ {
		j := rng.Dice(0, 256)
		s.perm[i], s.perm[j] = s.perm[j], s.perm[i]
	}
	return s
}

func (s *SimplexNoise) p(n int) int {
	return s.perm[n&255]
}

func (s *SimplexNoise) Noise2D(xin, yin float64) float64 {
	F2 := 0.5 * (math.Sqrt(3.0) - 1.0)
	G2 := (3.0 - math.Sqrt(3.0)) / 6.0

	ss := (xin + yin) * F2
	i := fastfloor(xin + ss)
	j := fastfloor(yin + ss)

	t := float64(i+j) * G2
	x0 := xin - (float64(i) - t)
	y0 := yin - (float64(j) - t)

	var i1, j1 int
	if x0 > y0 {
		i1, j1 = 1, 0
	} else {
		i1, j1 = 0, 1
	}

	x1 := x0 - float64(i1) + G2
	y1 := y0 - float64(j1) + G2
	x2 := x0 - 1.0 + 2.0*G2
	y2 := y0 - 1.0 + 2.0*G2

	ii := i & 255
	jj := j & 255
	gi0 := s.p(ii+s.p(jj)) % 12
	gi1 := s.p(ii+i1+s.p(jj+j1)) % 12
	gi2 := s.p(ii+1+s.p(jj+1)) % 12

	var n0, n1, n2 float64

	t0 := 0.5 - x0*x0 - y0*y0
	if t0 >= 0 {
		t0 *= t0
		n0 = t0 * t0 * dot2(grad3[gi0], x0, y0)
	}

	t1 := 0.5 - x1*x1 - y1*y1
	if t1 >= 0 {
		t1 *= t1
		n1 = t1 * t1 * dot2(grad3[gi1], x1, y1)
	}

	t2 := 0.5 - x2*x2 - y2*y2
	if t2 >= 0 {
		t2 *= t2
		n2 = t2 * t2 * dot2(grad3[gi2], x2, y2)
	}

	return 70.0 * (n0 + n1 + n2)
}

func (s *SimplexNoise) Noise3D(xin, yin, zin float64) float64 {
	const F3 = 1.0 / 3.0
	const G3 = 1.0 / 6.0

	ss := (xin + yin + zin) * F3
	i := fastfloor(xin + ss)
	j := fastfloor(yin + ss)
	k := fastfloor(zin + ss)

	t := float64(i+j+k) * G3
	x0 := xin - (float64(i) - t)
	y0 := yin - (float64(j) - t)
	z0 := zin - (float64(k) - t)

	var i1, j1, k1, i2, j2, k2 int
	if x0 >= y0 {
		if y0 >= z0 {
			i1, j1, k1, i2, j2, k2 = 1, 0, 0, 1, 1, 0
		} else if x0 >= z0 {
			i1, j1, k1, i2, j2, k2 = 1, 0, 0, 1, 0, 1
		} else {
			i1, j1, k1, i2, j2, k2 = 0, 0, 1, 1, 0, 1
		}
	} else {
		if y0 < z0 {
			i1, j1, k1, i2, j2, k2 = 0, 0, 1, 0, 1, 1
		} else if x0 < z0 {
			i1, j1, k1, i2, j2, k2 = 0, 1, 0, 0, 1, 1
		} else {
			i1, j1, k1, i2, j2, k2 = 0, 1, 0, 1, 1, 0
		}
	}

	x1 := x0 - float64(i1) + G3
	y1 := y0 - float64(j1) + G3
	z1 := z0 - float64(k1) + G3
	x2 := x0 - float64(i2) + 2.0*G3
	y2 := y0 - float64(j2) + 2.0*G3
	z2 := z0 - float64(k2) + 2.0*G3
	x3 := x0 - 1.0 + 3.0*G3
	y3 := y0 - 1.0 + 3.0*G3
	z3 := z0 - 1.0 + 3.0*G3

	ii := i & 255
	jj := j & 255
	kk := k & 255
	gi0 := s.p(ii+s.p(jj+s.p(kk))) % 12
	gi1 := s.p(ii+i1+s.p(jj+j1+s.p(kk+k1))) % 12
	gi2 := s.p(ii+i2+s.p(jj+j2+s.p(kk+k2))) % 12
	gi3 := s.p(ii+1+s.p(jj+1+s.p(kk+1))) % 12

	var n0, n1, n2, n3 float64

	t0 := 0.6 - x0*x0 - y0*y0 - z0*z0
	if t0 >= 0 {
		t0 *= t0
		n0 = t0 * t0 * dot3(grad3[gi0], x0, y0, z0)
	}
	t1 := 0.6 - x1*x1 - y1*y1 - z1*z1
	if t1 >= 0 {
		t1 *= t1
		n1 = t1 * t1 * dot3(grad3[gi1], x1, y1, z1)
	}
	t2 := 0.6 - x2*x2 - y2*y2 - z2*z2
	if t2 >= 0 {
		t2 *= t2
		n2 = t2 * t2 * dot3(grad3[gi2], x2, y2, z2)
	}
	t3 := 0.6 - x3*x3 - y3*y3 - z3*z3
	if t3 >= 0 {
		t3 *= t3
		n3 = t3 * t3 * dot3(grad3[gi3], x3, y3, z3)
	}

	return 32.0 * (n0 + n1 + n2 + n3)
}

func (s *SimplexNoise) Noise4D(x, y, z, w float64) float64 {
	F4 := (math.Sqrt(5.0) - 1.0) / 4.0
	G4 := (5.0 - math.Sqrt(5.0)) / 20.0

	ss := (x + y + z + w) * F4
	i := fastfloor(x + ss)
	j := fastfloor(y + ss)
	k := fastfloor(z + ss)
	l := fastfloor(w + ss)

	t := float64(i+j+k+l) * G4
	x0 := x - (float64(i) - t)
	y0 := y - (float64(j) - t)
	z0 := z - (float64(k) - t)
	w0 := w - (float64(l) - t)

	c1 := boolToInt(x0 > y0) * 32
	c2 := boolToInt(x0 > z0) * 16
	c3 := boolToInt(y0 > z0) * 8
	c4 := boolToInt(x0 > w0) * 4
	c5 := boolToInt(y0 > w0) * 2
	c6 := boolToInt(z0 > w0)
	c := c1 + c2 + c3 + c4 + c5 + c6

	i1 := boolToInt(simplex4[c][0] >= 3)
	j1 := boolToInt(simplex4[c][1] >= 3)
	k1 := boolToInt(simplex4[c][2] >= 3)
	l1 := boolToInt(simplex4[c][3] >= 3)

	i2 := boolToInt(simplex4[c][0] >= 2)
	j2 := boolToInt(simplex4[c][1] >= 2)
	k2 := boolToInt(simplex4[c][2] >= 2)
	l2 := boolToInt(simplex4[c][3] >= 2)

	i3 := boolToInt(simplex4[c][0] >= 1)
	j3 := boolToInt(simplex4[c][1] >= 1)
	k3 := boolToInt(simplex4[c][2] >= 1)
	l3 := boolToInt(simplex4[c][3] >= 1)

	x1 := x0 - float64(i1) + G4
	y1 := y0 - float64(j1) + G4
	z1 := z0 - float64(k1) + G4
	w1 := w0 - float64(l1) + G4
	x2 := x0 - float64(i2) + 2.0*G4
	y2 := y0 - float64(j2) + 2.0*G4
	z2 := z0 - float64(k2) + 2.0*G4
	w2 := w0 - float64(l2) + 2.0*G4
	x3 := x0 - float64(i3) + 3.0*G4
	y3 := y0 - float64(j3) + 3.0*G4
	z3 := z0 - float64(k3) + 3.0*G4
	w3 := w0 - float64(l3) + 3.0*G4
	x4 := x0 - 1.0 + 4.0*G4
	y4 := y0 - 1.0 + 4.0*G4
	z4 := z0 - 1.0 + 4.0*G4
	w4 := w0 - 1.0 + 4.0*G4

	ii := i & 255
	jj := j & 255
	kk := k & 255
	ll := l & 255
	gi0 := s.p(ii+s.p(jj+s.p(kk+s.p(ll)))) % 32
	gi1 := s.p(ii+i1+s.p(jj+j1+s.p(kk+k1+s.p(ll+l1)))) % 32
	gi2 := s.p(ii+i2+s.p(jj+j2+s.p(kk+k2+s.p(ll+l2)))) % 32
	gi3 := s.p(ii+i3+s.p(jj+j3+s.p(kk+k3+s.p(ll+l3)))) % 32
	gi4 := s.p(ii+1+s.p(jj+1+s.p(kk+1+s.p(ll+1)))) % 32

	var n0, n1, n2, n3, n4 float64

	t0 := 0.6 - x0*x0 - y0*y0 - z0*z0 - w0*w0
	if t0 >= 0 {
		t0 *= t0
		n0 = t0 * t0 * dot4(grad4[gi0], x0, y0, z0, w0)
	}
	t1 := 0.6 - x1*x1 - y1*y1 - z1*z1 - w1*w1
	if t1 >= 0 {
		t1 *= t1
		n1 = t1 * t1 * dot4(grad4[gi1], x1, y1, z1, w1)
	}
	t2 := 0.6 - x2*x2 - y2*y2 - z2*z2 - w2*w2
	if t2 >= 0 {
		t2 *= t2
		n2 = t2 * t2 * dot4(grad4[gi2], x2, y2, z2, w2)
	}
	t3 := 0.6 - x3*x3 - y3*y3 - z3*z3 - w3*w3
	if t3 >= 0 {
		t3 *= t3
		n3 = t3 * t3 * dot4(grad4[gi3], x3, y3, z3, w3)
	}
	t4 := 0.6 - x4*x4 - y4*y4 - z4*z4 - w4*w4
	if t4 >= 0 {
		t4 *= t4
		n4 = t4 * t4 * dot4(grad4[gi4], x4, y4, z4, w4)
	}

	return 27.0 * (n0 + n1 + n2 + n3 + n4)
}

func fastfloor(x float64) int {
	ix := int(x)
	if x < 0 {
		return ix - 1
	}
	return ix
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func dot2(g [3]int, x, y float64) float64 {
	return float64(g[0])*x + float64(g[1])*y
}

func dot3(g [3]int, x, y, z float64) float64 {
	return float64(g[0])*x + float64(g[1])*y + float64(g[2])*z
}

func dot4(g [4]int, x, y, z, w float64) float64 {
	return float64(g[0])*x + float64(g[1])*y + float64(g[2])*z + float64(g[3])*w
}

var grad3 = [12][3]int{
	{1, 1, 0}, {-1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
	{1, 0, 1}, {-1, 0, 1}, {1, 0, -1}, {-1, 0, -1},
	{0, 1, 1}, {0, -1, 1}, {0, 1, -1}, {0, -1, -1},
}

var grad4 = [32][4]int{
	{0, 1, 1, 1}, {0, 1, 1, -1}, {0, 1, -1, 1}, {0, 1, -1, -1},
	{0, -1, 1, 1}, {0, -1, 1, -1}, {0, -1, -1, 1}, {0, -1, -1, -1},
	{1, 0, 1, 1}, {1, 0, 1, -1}, {1, 0, -1, 1}, {1, 0, -1, -1},
	{-1, 0, 1, 1}, {-1, 0, 1, -1}, {-1, 0, -1, 1}, {-1, 0, -1, -1},
	{1, 1, 0, 1}, {1, 1, 0, -1}, {1, -1, 0, 1}, {1, -1, 0, -1},
	{-1, 1, 0, 1}, {-1, 1, 0, -1}, {-1, -1, 0, 1}, {-1, -1, 0, -1},
	{1, 1, 1, 0}, {1, 1, -1, 0}, {1, -1, 1, 0}, {1, -1, -1, 0},
	{-1, 1, 1, 0}, {-1, 1, -1, 0}, {-1, -1, 1, 0}, {-1, -1, -1, 0},
}

var simplex4 = [64][4]int{
	{0, 1, 2, 3}, {0, 1, 3, 2}, {0, 0, 0, 0}, {0, 2, 3, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {1, 2, 3, 0},
	{0, 2, 1, 3}, {0, 0, 0, 0}, {0, 3, 1, 2}, {0, 3, 2, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {1, 3, 2, 0},
	{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0},
	{1, 2, 0, 3}, {0, 0, 0, 0}, {1, 3, 0, 2}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {2, 3, 0, 1}, {2, 3, 1, 0},
	{1, 0, 2, 3}, {1, 0, 3, 2}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {2, 0, 3, 1}, {0, 0, 0, 0}, {2, 1, 3, 0},
	{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0},
	{2, 0, 1, 3}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {3, 0, 1, 2}, {3, 0, 2, 1}, {0, 0, 0, 0}, {3, 1, 2, 0},
	{2, 1, 0, 3}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {3, 1, 0, 2}, {0, 0, 0, 0}, {3, 2, 0, 1}, {3, 2, 1, 0},
}
