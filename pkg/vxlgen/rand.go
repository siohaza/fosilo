package vxlgen

import (
	"math/rand/v2"
)

type Rng struct {
	r *rand.Rand
}

func NewRng(seed uint64) *Rng {
	return &Rng{r: rand.New(rand.NewPCG(seed, seed^0xa5a5a5a5a5a5a5a5))}
}

func (rng *Rng) Dice(min, max int) int {
	return min + rng.r.IntN(max-min)
}

func (rng *Rng) Uniform() float64 {
	return rng.r.Float64()
}

func (rng *Rng) Bool() bool {
	return rng.r.IntN(2) != 0
}

func (rng *Rng) NormalPerturbation() Vec3f {
	return Vec3f{
		float32(rng.r.NormFloat64()),
		float32(rng.r.NormFloat64()),
		float32(rng.r.NormFloat64()),
	}
}

func (rng *Rng) RandomColor() Vec3f {
	return Vec3f{
		float32(rng.r.Float64()),
		float32(rng.r.Float64()),
		float32(rng.r.Float64()),
	}
}

func (rng *Rng) RandomDirection() Vec2i {
	dir := rng.Dice(0, 4)
	switch dir {
	case 0:
		return Vec2i{1, 0}
	case 1:
		return Vec2i{-1, 0}
	case 2:
		return Vec2i{0, 1}
	default:
		return Vec2i{0, -1}
	}
}
