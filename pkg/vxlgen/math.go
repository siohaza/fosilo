package vxlgen

import "math"

type Vec2i struct{ X, Y int }
type Vec3i struct{ X, Y, Z int }
type Vec3f struct{ X, Y, Z float32 }
type Box3i struct{ Min, Max Vec3i }

func (v Vec3i) Add(o Vec3i) Vec3i { return Vec3i{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3i) Sub(o Vec3i) Vec3i { return Vec3i{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3i) Scale(s int) Vec3i { return Vec3i{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3i) Neg() Vec3i        { return Vec3i{-v.X, -v.Y, -v.Z} }
func (v Vec3i) Eq(o Vec3i) bool   { return v.X == o.X && v.Y == o.Y && v.Z == o.Z }
func (v Vec3i) ToVec3f() Vec3f    { return Vec3f{float32(v.X), float32(v.Y), float32(v.Z)} }

func (v Vec3f) Add(o Vec3f) Vec3f    { return Vec3f{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3f) Sub(o Vec3f) Vec3f    { return Vec3f{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3f) Mulf(s float32) Vec3f { return Vec3f{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3f) Divf(s float32) Vec3f { return Vec3f{v.X / s, v.Y / s, v.Z / s} }

func (b Box3i) Contains(v Vec3i) bool {
	return v.X >= b.Min.X && v.X < b.Max.X &&
		v.Y >= b.Min.Y && v.Y < b.Max.Y &&
		v.Z >= b.Min.Z && v.Z < b.Max.Z
}

func (b Box3i) Volume() int {
	dx := b.Max.X - b.Min.X
	dy := b.Max.Y - b.Min.Y
	dz := b.Max.Z - b.Min.Z
	if dx <= 0 || dy <= 0 || dz <= 0 {
		return 0
	}
	return dx * dy * dz
}

func Lerp3f(a, b Vec3f, t float32) Vec3f {
	return Vec3f{
		a.X*(1-t) + b.X*t,
		a.Y*(1-t) + b.Y*t,
		a.Z*(1-t) + b.Z*t,
	}
}

func ClampF(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func Grey(color Vec3f, fraction float32) Vec3f {
	g := (color.X + color.Y + color.Z) / 3
	return Lerp3f(color, Vec3f{g, g, g}, fraction)
}

func Rotate(v, direction Vec3i) Vec3i {
	switch {
	case direction.Eq(Vec3i{1, 0, 0}):
		return v
	case direction.Eq(Vec3i{-1, 0, 0}):
		return Vec3i{-v.X, -v.Y, v.Z}
	case direction.Eq(Vec3i{0, 1, 0}):
		return Vec3i{-v.Y, v.X, v.Z}
	case direction.Eq(Vec3i{0, -1, 0}):
		return Vec3i{v.Y, -v.X, v.Z}
	default:
		return v
	}
}

func AbsI(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func MinI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func FloorToInt(x float64) int {
	return int(math.Floor(x))
}
