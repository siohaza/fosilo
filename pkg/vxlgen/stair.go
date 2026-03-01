package vxlgen

type Stair struct {
	Start     Vec3i
	Direction Vec3i
	Color1    Vec3f
	Color2    Vec3f
}

func NewStair(start, direction Vec3i, color1, color2 Vec3f) *Stair {
	return &Stair{
		Start:     start,
		Direction: direction,
		Color1:    color1,
		Color2:    color2,
	}
}

func (s *Stair) CellPosition() Vec3i {
	return s.Start
}

func (s *Stair) BuildCells(rng *Rng, grid *Grid) {
	a := grid.CellAt(s.Start)
	bpos := s.Start.Add(s.Direction)
	b := grid.CellAt(bpos)
	cpos := s.Start.Sub(s.Direction)
	c := grid.CellAt(cpos)

	a.Type = CellStairBody
	b.Type = CellStairBody
	c.Type = CellStairEndLow

	a.HasFloor = true
	b.HasFloor = true
	c.HasFloor = true

	grid.ConnectWith(s.Start, s.Direction)
	grid.ConnectWith(s.Start, s.Direction.Neg())

	if grid.Contains(bpos.Add(s.Direction)) {
		grid.DisconnectWith(bpos, s.Direction)
	}

	dirSide1 := Vec3i{s.Direction.Y, -s.Direction.X, 0}
	dirSide2 := Vec3i{-s.Direction.Y, s.Direction.X, 0}
	grid.TryDisconnectWith(s.Start, dirSide1)
	grid.TryDisconnectWith(s.Start, dirSide2)
	grid.TryDisconnectWith(bpos, dirSide1)
	grid.TryDisconnectWith(bpos, dirSide2)

	aboveA := s.Start.Add(Vec3i{0, 0, 1})
	aboveB := bpos.Add(Vec3i{0, 0, 1})
	aboveC := cpos.Add(Vec3i{0, 0, 1})

	grid.CellAt(aboveA).Type = CellAir
	grid.ConnectWith(s.Start, Vec3i{0, 0, 1})

	grid.ConnectWith(aboveA, aboveB.Sub(aboveA))
	grid.CellAt(aboveB).Type = CellAir
	grid.ConnectWith(bpos, Vec3i{0, 0, 1})

	aboveD := aboveB.Add(s.Direction)
	grid.CellAt(aboveD).Type = CellStairEndHigh
	grid.ConnectWith(aboveB, s.Direction)
	grid.DisconnectWith(aboveD, Vec3i{0, 0, -1})

	if grid.Contains(aboveA.Add(dirSide1)) {
		grid.CellAt(aboveA.Add(dirSide1)).Balcony = BalconySimple
	}
	if grid.Contains(aboveA.Add(dirSide2)) {
		grid.CellAt(aboveA.Add(dirSide2)).Balcony = BalconySimple
	}
	if grid.Contains(aboveB.Add(dirSide1)) {
		grid.CellAt(aboveB.Add(dirSide1)).Balcony = BalconySimple
	}
	if grid.Contains(aboveB.Add(dirSide2)) {
		grid.CellAt(aboveB.Add(dirSide2)).Balcony = BalconySimple
	}
	if grid.Contains(aboveC) {
		grid.CellAt(aboveC).Balcony = BalconySimple
	}
}

func (s *Stair) BuildBlocks(rng *Rng, grid *Grid, base Vec3i, m *MapWrapper) {
	centerA := base.Add(Vec3i{2, 2, 0})

	remapCenter := func(input Vec3i) Vec3i {
		diff := input.Sub(centerA)
		return centerA.Add(Rotate(diff, s.Direction))
	}

	for j := 1; j < 4; j++ {
		for i := 2; i < 8; i++ {
			height := i - 1
			for k := 1; k <= 6; k++ {
				p := remapCenter(base.Add(Vec3i{i, j, k}))
				if k <= height {
					if k&1 != 0 {
						m.SetBlockF(p.X, p.Y, p.Z, s.Color1)
					} else {
						m.SetBlockF(p.X, p.Y, p.Z, s.Color2)
					}
				} else {
					m.EmptyBlock(p.X, p.Y, p.Z)
				}
			}
		}
	}
}
