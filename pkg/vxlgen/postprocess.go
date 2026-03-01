package vxlgen

func ColorBleed(w *MapWrapper) {
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			for z := 1; z < 63; z++ {
				if !w.IsSolid(x, y, z) {
					continue
				}

				fr, fg, fb := w.GetColor(x, y, z)
				luminance := 0.6*float32(fg) + 0.3*float32(fr) + 0.1*float32(fb)

				var count float32
				var sumR, sumG, sumB float32

				tryBlock := func(bx, by, bz, weight int) {
					if !w.Contains(bx, by, bz) || !w.IsSolid(bx, by, bz) {
						return
					}
					r, g, b := w.GetColor(bx, by, bz)
					sumR += float32(r) * float32(weight)
					sumG += float32(g) * float32(weight)
					sumB += float32(b) * float32(weight)
					count += float32(weight)
				}

				tryBlock(x, y, z, 20)
				tryBlock(x-1, y, z, 1)
				tryBlock(x+1, y, z, 1)
				tryBlock(x, y-1, z, 1)
				tryBlock(x, y+1, z, 1)
				tryBlock(x, y, z-1, 1)
				tryBlock(x, y, z+1, 1)

				if count > 0 {
					inv := 1.0 / count
					avgR := sumR * inv
					avgG := sumG * inv
					avgB := sumB * inv
					lum2 := 0.6*avgG + 0.1*avgB + 0.3*avgR
					scale := luminance / (lum2 + 0.01)
					w.SetColor(x, y, z,
						uint8(ClampF(avgR*scale+0.5, 0, 255)),
						uint8(ClampF(avgG*scale+0.5, 0, 255)),
						uint8(ClampF(avgB*scale+0.5, 0, 255)),
					)
				}
			}
		}
	}
}

func BetterAO(w *MapWrapper) {
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			for z := 1; z < 63; z++ {
				if !w.IsSolid(x, y, z) {
					continue
				}

				occlusion := 0
				for i := -1; i <= 1; i++ {
					for j := -1; j <= 1; j++ {
						for k := 0; k <= 1; k++ {
							bx, by, bz := x+j, y+i, z+k
							if w.Contains(bx, by, bz) && w.IsSolid(bx, by, bz) {
								occlusion++
							}
						}
					}
				}

				occluded := ClampF(float32(occlusion)/18.0, 0, 1)
				scale := 1.4 - occluded*0.8

				r, g, b := w.GetColor(x, y, z)
				w.SetColor(x, y, z,
					uint8(ClampF(float32(r)*scale+0.5, 0, 255)),
					uint8(ClampF(float32(g)*scale+0.5, 0, 255)),
					uint8(ClampF(float32(b)*scale+0.5, 0, 255)),
				)
			}
		}
	}
}

func ReverseClientAO(w *MapWrapper) {
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			for z := 1; z < 63; z++ {
				if !w.IsSolid(x, y, z) {
					continue
				}

				obstruction := 0
				for i := 1; i <= 9; i++ {
					px := x
					py := y - i
					pz := z + i
					if w.Contains(px, py, pz) && w.IsSolid(px, py, pz) {
						obstruction++
					}
				}

				fact := 1.0 - 0.5*float32(obstruction)/9.0
				invFact := 1.0 / fact

				r, g, b := w.GetColor(x, y, z)
				w.SetColor(x, y, z,
					uint8(ClampF(float32(r)*invFact+0.5, 0, 255)),
					uint8(ClampF(float32(g)*invFact+0.5, 0, 255)),
					uint8(ClampF(float32(b)*invFact+0.5, 0, 255)),
				)
			}
		}
	}
}
