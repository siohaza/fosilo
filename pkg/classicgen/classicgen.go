// classicgen generator
// original by Tom Dobrowolski and Ken Silverman
// https://web.archive.org/web/20170223015419/http://moonedit.com/tom/vox1_en.htm#genland

package classicgen

import (
	"math"

	"github.com/siohaza/fosilo/pkg/vxl"
)

const (
	octMax = 10
	eps    = 0.1
	vsid   = 512
	vdepth = 64
	pi     = math.Pi
)

type noiseContext struct {
	noisep   [512]uint8
	noisep15 [512]uint8
	seed     uint32
}

type vcol struct {
	r, g, b, a uint8
}

func (nc *noiseContext) getRandom() uint32 {
	nc.seed = nc.seed*214013 + 2531011
	return (nc.seed >> 16) & 0x7FFF
}

func (nc *noiseContext) initNoise() {
	for i := 255; i >= 0; i-- {
		nc.noisep[i] = uint8(i)
	}
	for i := 255; i > 0; i-- {
		j := (nc.getRandom() * uint32(i+1)) >> 15
		nc.noisep[i], nc.noisep[j] = nc.noisep[j], nc.noisep[i]
	}
	for i := 255; i >= 0; i-- {
		nc.noisep[i+256] = nc.noisep[i]
	}
	for i := 511; i >= 0; i-- {
		nc.noisep15[i] = nc.noisep[i] & 15
	}
}

func fgrad(h int, x, y, z float64) float64 {
	switch h {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x + z
	case 5:
		return -x + z
	case 6:
		return x - z
	case 7:
		return -x - z
	case 8:
		return y + z
	case 9:
		return -y + z
	case 10:
		return y - z
	case 11:
		return -y - z
	case 12:
		return x + y
	case 13:
		return -x + y
	case 14:
		return y - z
	case 15:
		return -y - z
	}
	return 0
}

func (nc *noiseContext) noise3d(fx, fy, fz float64, mask int) float64 {
	var l [6]int
	var p [3]float64
	var f [8]float64
	var a [4]int

	l[0] = int(math.Floor(fx))
	p[0] = fx - float64(l[0])
	l[0] &= mask
	l[3] = (l[0] + 1) & mask

	l[1] = int(math.Floor(fy))
	p[1] = fy - float64(l[1])
	l[1] &= mask
	l[4] = (l[1] + 1) & mask

	l[2] = int(math.Floor(fz))
	p[2] = fz - float64(l[2])
	l[2] &= mask
	l[5] = (l[2] + 1) & mask

	i := nc.noisep[l[0]]
	a[0] = int(nc.noisep[int(i)+l[1]])
	a[2] = int(nc.noisep[int(i)+l[4]])
	i = nc.noisep[l[3]]
	a[1] = int(nc.noisep[int(i)+l[1]])
	a[3] = int(nc.noisep[int(i)+l[4]])

	f[0] = fgrad(int(nc.noisep15[a[0]+l[2]]), p[0], p[1], p[2])
	f[1] = fgrad(int(nc.noisep15[a[1]+l[2]]), p[0]-1, p[1], p[2])
	f[2] = fgrad(int(nc.noisep15[a[2]+l[2]]), p[0], p[1]-1, p[2])
	f[3] = fgrad(int(nc.noisep15[a[3]+l[2]]), p[0]-1, p[1]-1, p[2])
	p[2]--
	f[4] = fgrad(int(nc.noisep15[a[0]+l[5]]), p[0], p[1], p[2])
	f[5] = fgrad(int(nc.noisep15[a[1]+l[5]]), p[0]-1, p[1], p[2])
	f[6] = fgrad(int(nc.noisep15[a[2]+l[5]]), p[0], p[1]-1, p[2])
	f[7] = fgrad(int(nc.noisep15[a[3]+l[5]]), p[0]-1, p[1]-1, p[2])
	p[2]++

	p[2] = (3.0 - 2.0*p[2]) * p[2] * p[2]
	p[1] = (3.0 - 2.0*p[1]) * p[1] * p[1]
	p[0] = (3.0 - 2.0*p[0]) * p[0] * p[0]

	f[0] = (f[4]-f[0])*p[2] + f[0]
	f[1] = (f[5]-f[1])*p[2] + f[1]
	f[2] = (f[6]-f[2])*p[2] + f[2]
	f[3] = (f[7]-f[3])*p[2] + f[3]
	f[0] = (f[2]-f[0])*p[1] + f[0]
	f[1] = (f[3]-f[1])*p[1] + f[1]

	return (f[1]-f[0])*p[0] + f[0]
}

func getHeightPos(x, y int) int {
	return y*vsid + x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clamp(v, minv, maxv float64) float64 {
	if v < minv {
		return minv
	}
	if v > maxv {
		return maxv
	}
	return v
}

func Generate(seed uint32) (*vxl.Map, error) {
	nc := &noiseContext{seed: seed}
	nc.initNoise()

	buf := make([]vcol, vsid*vsid)
	amb := make([]vcol, vsid*vsid)

	var amplut [octMax]float64
	var msklut [octMax]int

	d := 1.0
	for i := 0; i < octMax; i++ {
		amplut[i] = d
		d *= 0.4
		msklut[i] = minInt((1<<(i+2))-1, 255)
	}

	k := 0
	for y := 0; y < vsid; y++ {
		for x := 0; x < vsid; x++ {
			var samp, csamp [3]float64

			for i := 0; i < 3; i++ {
				dx := (float64(x)*(256.0/float64(vsid)) + float64(i&1)*eps) * (1.0 / 64.0)
				dy := (float64(y)*(256.0/float64(vsid)) + float64(i>>1)*eps) * (1.0 / 64.0)
				d := 0.0
				river := 0.0

				for o := 0; o < octMax; o++ {
					d += nc.noise3d(dx, dy, 9.5, msklut[o]) * amplut[o] * (d*1.6 + 1.0)
					river += nc.noise3d(dx, dy, 13.2, msklut[o]) * amplut[o]
					dx *= 2
					dy *= 2
				}

				samp[i] = d*-20.0 + 28.0
				d = math.Sin(float64(x)*(pi/256.0)+river*4.0)*(0.5+0.02) + (0.5 - 0.02)
				if d > 1 {
					d = 1
				}
				csamp[i] = samp[i] * d
				if d < 0 {
					d = 0
				}
				samp[i] *= d
				if csamp[i] < samp[i] {
					csamp[i] = -math.Log(1.0 - csamp[i])
				}
			}

			nx := csamp[1] - csamp[0]
			ny := csamp[2] - csamp[0]
			nz := -eps
			d = 1.0 / math.Sqrt(nx*nx+ny*ny+nz*nz)
			nx *= d
			ny *= d
			nz *= d

			gr := 140.0
			gg := 125.0
			gb := 115.0

			g := min(math.Max(math.Max(-nz, 0)*1.4-csamp[0]/32.0+nc.noise3d(float64(x)*(1.0/64.0), float64(y)*(1.0/64.0), 0.3, 15)*0.3, 0), 1)
			gr += (72 - gr) * g
			gg += (80 - gg) * g
			gb += (32 - gb) * g

			g2 := (1 - math.Abs(g-0.5)*2) * 0.7
			gr += (68 - gr) * g2
			gg += (78 - gg) * g2
			gb += (40 - gb) * g2

			g2 = math.Max(min((samp[0]-csamp[0])*1.5, 1), 0)
			g = 1 - g2*0.2
			gr += (60*g - gr) * g2
			gg += (100*g - gg) * g2
			gb += (120*g - gb) * g2

			d = 0.3
			amb[k].r = uint8(clamp(gr*d, 0, 255))
			amb[k].g = uint8(clamp(gg*d, 0, 255))
			amb[k].b = uint8(clamp(gb*d, 0, 255))
			maxa := max(max(int(amb[k].r), int(amb[k].g)), int(amb[k].b))

			d = (nx*0.5 + ny*0.25 - nz) / math.Sqrt(0.5*0.5+0.25*0.25+1.0*1.0)
			d *= 1.2
			buf[k].a = uint8(63 - samp[0])
			buf[k].r = uint8(clamp(gr*d, 0, float64(255-maxa)))
			buf[k].g = uint8(clamp(gg*d, 0, float64(255-maxa)))
			buf[k].b = uint8(clamp(gb*d, 0, float64(255-maxa)))

			k++
		}
	}

	for y := 0; y < vsid; y++ {
		for x := 0; x < vsid; x++ {
			k := getHeightPos(x, y)
			buf[k].r += amb[k].r
			buf[k].g += amb[k].g
			buf[k].b += amb[k].b
		}
	}

	m, err := vxl.NewEmpty(vsid, vsid, vdepth)
	if err != nil {
		return nil, err
	}

	for y := 0; y < vsid; y++ {
		for x := 0; x < vsid; x++ {
			k := getHeightPos(x, y)
			height := int(buf[k].a)
			color := uint32(buf[k].b) | uint32(buf[k].g)<<8 | uint32(buf[k].r)<<16

			for z := 63; z >= height; z-- {
				m.SetNoOptimize(x, y, z, color)
			}
		}
	}

	return m, nil
}
