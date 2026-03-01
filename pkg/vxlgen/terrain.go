package vxlgen

import "math"

type Tree struct {
	Pos          Vec3i
	Height       int
	TrunkColor   Vec3f
	FoliageColor Vec3f
	Conifere     float32
}

func (tr *Tree) BuildBlocks(rng *Rng, m *MapWrapper) {
	trunkSize := tr.Height / 2
	if trunkSize > 3 {
		trunkSize = 3
	}

	maxRadius := 1.0 + float64(tr.Height-trunkSize)*0.33
	minRadius := 0.2
	skewNess := 0.8 + rng.Uniform()*0.4

	// cast shadow
	iradius := int(maxRadius) + 1
	for i := -3; i < tr.Height; i++ {
		for x := -iradius; x <= iradius; x++ {
			for y := -iradius; y <= iradius; y++ {
				dist := float64(x*x + y*y)
				if dist < maxRadius*maxRadius {
					px, py, pz := tr.Pos.X+x, tr.Pos.Y+y, tr.Pos.Z+i
					if m.Contains(px, py, pz) && m.IsSolid(px, py, pz) {
						r, g, b := m.GetColor(px, py, pz)
						m.SetColor(px, py, pz,
							uint8(float32(r)*0.7),
							uint8(float32(g)*0.7),
							uint8(float32(b)*0.7),
						)
					}
				}
			}
		}
	}

	// trunk
	for i := 0; i < trunkSize; i++ {
		m.SetBlockF(tr.Pos.X, tr.Pos.Y, tr.Pos.Z+i, tr.TrunkColor)
	}

	// foliage
	for i := trunkSize; i < tr.Height; i++ {
		t := float64(i-trunkSize) / float64(tr.Height-1-trunkSize)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		radius := 1.4 * lerp64(maxRadius, minRadius, math.Pow(t, skewNess))
		ir := int(radius) + 1
		for x := -ir; x <= ir; x++ {
			for y := -ir; y <= ir; y++ {
				pp := Vec3i{tr.Pos.X + x, tr.Pos.Y + y, tr.Pos.Z + i}
				dist := math.Sqrt(float64(x*x + y*y))
				dist *= 0.8 + 0.4*rng.Uniform()
				if m.ContainsV(pp) && dist < radius {
					m.SetBlockF(pp.X, pp.Y, pp.Z, tr.FoliageColor)
				}
			}
		}
	}
}

type Terrain struct {
	HeightMap  []float32
	Vegetation []float32
	Trees      []*Tree
	MapDim     Vec2i
}

func NewTerrain(mapDim Vec2i, rng *Rng) *Terrain {
	t := &Terrain{MapDim: mapDim}
	t.makeHeightMap(rng)
	t.makeVegetation(rng)
	return t
}

func (t *Terrain) makeHeightMap(rng *Rng) {
	numOct := 8
	noises := make([]*SimplexNoise, numOct)
	for i := 0; i < numOct; i++ {
		noises[i] = NewSimplexNoise(rng)
	}

	t.HeightMap = make([]float32, t.MapDim.X*t.MapDim.Y)

	for y := 0; y < t.MapDim.Y; y++ {
		for x := 0; x < t.MapDim.X; x++ {
			fx := float64(x) / float64(t.MapDim.X)
			fy := float64(y) / float64(t.MapDim.Y)
			z := 3.0

			for oct := 0; oct < numOct; oct++ {
				freq := math.Pow(2.0, float64(oct))
				zo := noises[oct].Noise2D(fx*freq, fy*freq)
				amplitude := 44 * math.Pow(2.0, float64(-oct))
				z += zo * amplitude
			}

			if z > 62 {
				z = 62
			}
			if z >= 1 {
				z = 1 + math.Pow((z-1)/62.0, 2.0)*62.0
			}
			if z > 54 {
				z = 54 + math.Log2(z-53)
			}

			dx := float64(x) - 255
			dy := float64(y) - 255
			distanceToCenter := math.Sqrt(dx*dx + dy*dy)
			heightIdeal := 7.0
			blend := 2 - distanceToCenter*0.012
			if blend < 0 {
				blend = 0
			}
			if blend > 1 {
				blend = 1
			}
			z = lerp64(z, heightIdeal, blend)

			t.HeightMap[y*t.MapDim.X+x] = float32(z)
		}
	}
}

func (t *Terrain) makeVegetation(rng *Rng) {
	numOct := 4
	noises := make([]*SimplexNoise, numOct)
	for i := 0; i < numOct; i++ {
		noises[i] = NewSimplexNoise(rng)
	}

	t.Vegetation = make([]float32, t.MapDim.X*t.MapDim.Y)

	for y := 0; y < t.MapDim.Y; y++ {
		for x := 0; x < t.MapDim.X; x++ {
			fx := float64(x) / float64(t.MapDim.X)
			fy := float64(y) / float64(t.MapDim.Y)
			veg := 0.0

			for oct := 0; oct < numOct; oct++ {
				freq := math.Pow(2.0, float64(oct))
				amplitude := math.Pow(2.0, float64(-oct))
				noise := noises[oct].Noise2D(fx*freq, fy*freq)
				veg += noise * amplitude
			}

			h := float64(t.HeightMap[y*t.MapDim.X+x])
			if h < 1 {
				veg = 0
			}
			if h > 24 {
				factor := 1.0 - (veg-24.0)/24.0
				if factor < 0 {
					factor = 0
				}
				if factor > 1 {
					factor = 1
				}
				veg *= factor
			}

			dx := float64(x) - 255
			dy := float64(y) - 255
			distanceToCenter := math.Sqrt(dx*dx + dy*dy)
			blend := 2 - distanceToCenter*0.012
			if blend < 0 {
				blend = 0
			}
			if blend > 1 {
				blend = 1
			}
			veg = lerp64(veg, 0, blend)

			if veg < 0 {
				veg = 0
			}
			if veg > 1 {
				veg = 1
			}
			t.Vegetation[y*t.MapDim.X+x] = float32(veg)
		}
	}

	// add trees
	for y := 0; y < t.MapDim.Y; y++ {
		for x := 0; x < t.MapDim.X; x++ {
			h := int(0.5 + float64(t.HeightMap[y*t.MapDim.X+x]))
			if h < 1 {
				h = 0
			}
			if h > 62 {
				h = 62
			}

			if h > 1 && rng.Uniform() < float64(t.Vegetation[y*t.MapDim.X+x])*0.05 {
				treeHeight := rng.Dice(5, 10)
				if h+treeHeight < 62 {
					trunkColor := Vec3f{116.0 / 255, 84.0 / 255, 52.0 / 255}
					green := Vec3f{81.0 / 255, 137.0 / 255, 56.0 / 255}
					yellow := Vec3f{175.0 / 255, 171.0 / 255, 3.0 / 255}
					darkGreen := Vec3f{54.0 / 255, 103.0 / 255, 37.0 / 255}

					trunkColor = Lerp3f(trunkColor, Vec3f{60.0 / 255, 0, 17.0 / 255}, float32(rng.Uniform()))

					a := float32(rng.Uniform()) + 0.1
					b := float32(rng.Uniform())
					c := float32(rng.Uniform())
					sum := a + b + c

					foliageColor := green.Mulf(a).Add(yellow.Mulf(b)).Add(darkGreen.Mulf(c)).Divf(sum)

					trunkColor = trunkColor.Add(rng.NormalPerturbation().Mulf(0.02))
					foliageColor = foliageColor.Add(rng.NormalPerturbation().Mulf(0.04))

					if h > 40 {
						white := Vec3f{1, 1, 1}
						tt := float32(h-40) / float32(48-32)
						if tt > 1 {
							tt = 1
						}
						trunkColor = Lerp3f(trunkColor, white, tt)
						foliageColor = Lerp3f(foliageColor, white, tt)
					}

					t.Trees = append(t.Trees, &Tree{
						Pos:          Vec3i{x, y, h + 1},
						Height:       treeHeight,
						FoliageColor: foliageColor,
						TrunkColor:   trunkColor,
					})

					// clear vegetation around tree
					for i := -4; i < 5; i++ {
						for j := -4; j < 5; j++ {
							ix := ((y + j) + t.MapDim.Y) % t.MapDim.Y
							iy := ((x + i) + t.MapDim.X) % t.MapDim.X
							t.Vegetation[ix*t.MapDim.X+iy] = 0
						}
					}
				}
			}
		}
	}
}

func (t *Terrain) BuildBlocks(rng *Rng, m *MapWrapper) {
	for y := 0; y < t.MapDim.Y; y++ {
		for x := 0; x < t.MapDim.X; x++ {
			z := float64(t.HeightMap[y*t.MapDim.X+x])
			h := int(0.5 + z)
			if h < 1 {
				h = 0
			}
			if h > 62 {
				h = 62
			}

			for k := 0; k <= h; k++ {
				var color Vec3f

				switch {
				case k == 0:
					lightBlue := Vec3f{90.0 / 255, 148.0 / 255, 237.0 / 255}
					darkBlue := Vec3f{32.0 / 255, 38.0 / 255, 119.0 / 255}
					tt := ClampF(float32(-(z-0.5)*0.1), 0, 1)
					color = Lerp3f(lightBlue, darkBlue, tt)
				case k == 1:
					color = Vec3f{0.9, 0.9, 0.9}
					color = color.Add(rng.NormalPerturbation().Mulf(0.015))
				case k == 2:
					sand := Vec3f{0.9, 0.9, 0.9}
					greenC := Vec3f{168.0 / 255, 194.0 / 255, 75.0 / 255}
					color = sand.Add(greenC).Divf(2)
					color = color.Add(rng.NormalPerturbation().Mulf(0.015))
				case k < 16:
					greenC := Vec3f{168.0 / 255, 194.0 / 255, 75.0 / 255}
					marron := Vec3f{118.0 / 255, 97.0 / 255, 56.0 / 255}
					color = Lerp3f(greenC, marron, float32(k-2)/float32(16-2))
					color = color.Add(rng.NormalPerturbation().Mulf(0.025))
				case k < 32:
					marron := Vec3f{118.0 / 255, 97.0 / 255, 56.0 / 255}
					grey := Vec3f{0.6, 0.6, 0.6}
					tt := float32(k-16) / float32(32-16)
					color = Lerp3f(marron, grey, tt)
					color = color.Add(rng.NormalPerturbation().Mulf(0.02 - tt*0.01))
				case k < 48:
					grey := Vec3f{0.6, 0.6, 0.6}
					white := Vec3f{1, 1, 1}
					color = Lerp3f(grey, white, float32(k-32)/float32(48-32))
					color = color.Add(rng.NormalPerturbation().Mulf(0.01))
				default:
					color = Vec3f{1, 1, 1}
					color = color.Add(rng.NormalPerturbation().Mulf(0.01))
				}

				m.SetBlockF(x, y, k, color)
			}
		}
	}

	// render trees
	for _, tree := range t.Trees {
		tree.BuildBlocks(rng, m)
	}
}

func lerp64(a, b, t float64) float64 {
	return a*(1-t) + b*t
}
