package vxlgen

type Pattern int

const (
	PatternOnlyOne Pattern = iota
	PatternBorder
	PatternTiles1x1
	PatternTiles2x2
	PatternTiles4x4
	PatternTilesZebra
	PatternTilesL
	PatternTiles2x2_1x1
	PatternCross
	PatternHoles
	patternCount
)

type PatternEx struct {
	Pattern     Pattern
	SwapIJ      bool
	SwapColors  bool
	NoiseAmount float32
}

var crossBitmap = [100]int{
	0, 1, 1, 1, 0, 1, 0, 0, 0, 1,
	0, 0, 1, 0, 1, 1, 1, 0, 1, 0,
	0, 1, 0, 0, 0, 1, 0, 1, 1, 1,
	1, 1, 1, 0, 1, 0, 0, 0, 1, 0,
	0, 1, 0, 1, 1, 1, 0, 1, 0, 0,
	1, 0, 0, 0, 1, 0, 1, 1, 1, 0,
	1, 1, 0, 1, 0, 0, 0, 1, 0, 1,
	1, 0, 1, 1, 1, 0, 1, 0, 0, 0,
	0, 0, 0, 1, 0, 1, 1, 1, 0, 1,
	1, 0, 1, 0, 0, 0, 1, 0, 1, 1,
}

var holesBitmap = [36]int{
	0, 0, 0, 1, 1, 1,
	0, 1, 0, 1, 0, 1,
	0, 0, 0, 1, 1, 1,
	1, 1, 1, 0, 0, 0,
	1, 0, 1, 0, 1, 0,
	1, 1, 1, 0, 0, 0,
}

var tiles2x2_1x1Bitmap = [9]int{
	0, 1, 1,
	1, 0, 0,
	1, 0, 0,
}

func subColor(i, j int, pattern Pattern, light, dark Vec3f) Vec3f {
	switch pattern {
	case PatternOnlyOne:
		return light

	case PatternBorder:
		if i%4 == 0 || j%4 == 0 {
			return light
		}
		return dark

	case PatternTiles1x1:
		if (i^j)&1 != 0 {
			return light
		}
		return dark

	case PatternTiles2x2:
		if ((i/2)^(j/2))&1 != 0 {
			return dark
		}
		return light

	case PatternTiles4x4:
		if i%4 == 0 || j%4 == 0 {
			return dark
		}
		if i%4 == 2 || j%4 == 2 {
			tileIsLight := ((i / 4) ^ (j / 4)) & 1
			if tileIsLight != 0 {
				return dark
			}
			return light
		}
		if ((i/4)^(j/4))&1 != 0 {
			return light
		}
		return dark

	case PatternTilesZebra:
		if ((i+j)/2)&1 != 0 {
			return dark
		}
		return light

	case PatternTilesL:
		if (i&j)&1 != 0 {
			return light
		}
		return dark

	case PatternTiles2x2_1x1:
		im := i % 3
		if im < 0 {
			im += 3
		}
		jm := j % 3
		if jm < 0 {
			jm += 3
		}
		if tiles2x2_1x1Bitmap[im*3+jm] != 0 {
			return light
		}
		return dark

	case PatternCross:
		im := i % 10
		if im < 0 {
			im += 10
		}
		jm := j % 10
		if jm < 0 {
			jm += 10
		}
		if crossBitmap[im*10+jm] != 0 {
			return light
		}
		return dark

	case PatternHoles:
		im := i % 6
		if im < 0 {
			im += 6
		}
		jm := j % 6
		if jm < 0 {
			jm += 6
		}
		if holesBitmap[im*6+jm] != 0 {
			return light
		}
		return dark

	default:
		return light
	}
}

func PatternColor(rng *Rng, pat PatternEx, i, j int, light, dark Vec3f) Vec3f {
	if pat.SwapIJ {
		i, j = j, i
	}
	if pat.SwapColors {
		light, dark = dark, light
	}
	c := subColor(i, j, pat.Pattern, light, dark)
	pert := rng.NormalPerturbation()
	return c.Add(pert.Mulf(pat.NoiseAmount))
}
