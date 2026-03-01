package vxlgen

import "github.com/siohaza/fosilo/pkg/vxl"

const (
	MapSize  = 512
	MapDepth = 64
)

type MapWrapper struct {
	M *vxl.Map
}

func NewMapWrapper() *MapWrapper {
	m, _ := vxl.NewEmpty(MapSize, MapSize, MapDepth)
	return &MapWrapper{M: m}
}

func (w *MapWrapper) flipZ(z int) int {
	return 63 - z
}

func (w *MapWrapper) SetBlock(x, y, z int, r, g, b uint8) {
	vz := w.flipZ(z)
	color := uint32(b) | uint32(g)<<8 | uint32(r)<<16
	w.M.SetNoOptimize(x, y, vz, color)
}

func (w *MapWrapper) SetBlockF(x, y, z int, c Vec3f) {
	r := uint8(ClampF(c.X*255+0.5, 0, 255))
	g := uint8(ClampF(c.Y*255+0.5, 0, 255))
	b := uint8(ClampF(c.Z*255+0.5, 0, 255))
	w.SetBlock(x, y, z, r, g, b)
}

func (w *MapWrapper) SetBlockI(x, y, z int, r, g, b uint8) {
	w.SetBlock(x, y, z, r, g, b)
}

func (w *MapWrapper) ClearBlock(x, y, z int) {
	vz := w.flipZ(z)
	w.M.SetAir(x, y, vz)
}

func (w *MapWrapper) EmptyBlock(x, y, z int) {
	vz := w.flipZ(z)
	if w.M.IsSolid(x, y, vz) {
		w.M.SetAir(x, y, vz)
	}
}

func (w *MapWrapper) IsSolid(x, y, z int) bool {
	vz := w.flipZ(z)
	return w.M.IsSolid(x, y, vz)
}

func (w *MapWrapper) Contains(x, y, z int) bool {
	return x >= 0 && x < MapSize && y >= 0 && y < MapSize && z >= 0 && z < MapDepth
}

func (w *MapWrapper) ContainsV(v Vec3i) bool {
	return w.Contains(v.X, v.Y, v.Z)
}

func (w *MapWrapper) GetColor(x, y, z int) (uint8, uint8, uint8) {
	vz := w.flipZ(z)
	color := w.M.Get(x, y, vz)
	r := uint8((color >> 16) & 0xFF)
	g := uint8((color >> 8) & 0xFF)
	b := uint8(color & 0xFF)
	return r, g, b
}

func (w *MapWrapper) SetColor(x, y, z int, r, g, b uint8) {
	w.SetBlock(x, y, z, r, g, b)
}

func (w *MapWrapper) ClearBox(b Box3i) {
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for z := b.Min.Z; z < b.Max.Z; z++ {
				w.EmptyBlock(x, y, z)
			}
		}
	}
}
