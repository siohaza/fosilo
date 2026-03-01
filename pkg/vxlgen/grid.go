package vxlgen

type Grid struct {
	NumCells Vec3i
	Cells    []Cell
}

func NewGrid(numCells Vec3i) *Grid {
	g := &Grid{
		NumCells: numCells,
		Cells:    make([]Cell, numCells.X*numCells.Y*numCells.Z),
	}
	for i := range g.Cells {
		g.Cells[i].Type = CellRegular
		g.Cells[i].Balcony = BalconyNone
	}
	return g
}

func (g *Grid) Contains(v Vec3i) bool {
	return g.ContainsXYZ(v.X, v.Y, v.Z)
}

func (g *Grid) ContainsXYZ(x, y, z int) bool {
	if x < 0 || y < 0 || z < 0 {
		return false
	}
	return x < g.NumCells.X && y < g.NumCells.Y && z < g.NumCells.Z
}

func (g *Grid) cellIndex(x, y, z int) int {
	return z*(g.NumCells.X*g.NumCells.Y) + y*g.NumCells.X + x
}

func (g *Grid) CellAt(v Vec3i) *Cell {
	return g.CellAtXYZ(v.X, v.Y, v.Z)
}

func (g *Grid) CellAtXYZ(x, y, z int) *Cell {
	return &g.Cells[g.cellIndex(x, y, z)]
}

func (g *Grid) IsExternal(v Vec3i) bool {
	if v.X == 0 || v.Y == 0 {
		return true
	}
	if v.X+1 == g.NumCells.X || v.Y+1 == g.NumCells.Y {
		return true
	}
	return false
}

func (g *Grid) IsConnectedWith(v, dir Vec3i) bool {
	if !g.Contains(v.Add(dir)) {
		return false
	}
	other := g.CellAt(v.Add(dir))
	it := g.CellAt(v)

	switch {
	case dir.X == -1:
		return !it.HasLeftWall
	case dir.Y == -1:
		return !it.HasTopWall
	case dir.Z == -1:
		if !it.HasFloor && other.Type == CellFull {
			return false
		}
		return !it.HasFloor
	case dir.X == 1:
		return !other.HasLeftWall
	case dir.Y == 1:
		return !other.HasTopWall
	case dir.Z == 1:
		return !other.HasFloor
	}
	return false
}

func (g *Grid) setWall(v, dir Vec3i, enabled bool) {
	it := g.CellAt(v)
	switch {
	case dir.X == -1:
		it.HasLeftWall = enabled
	case dir.Y == -1:
		it.HasTopWall = enabled
	case dir.Z == -1:
		it.HasFloor = enabled
	}

	other := g.CellAt(v.Add(dir))
	switch {
	case dir.X == 1:
		other.HasLeftWall = enabled
	case dir.Y == 1:
		other.HasTopWall = enabled
	case dir.Z == 1:
		other.HasFloor = enabled
	}
}

func (g *Grid) ConnectWith(v, dir Vec3i) {
	g.setWall(v, dir, false)
}

func (g *Grid) DisconnectWith(v, dir Vec3i) {
	g.setWall(v, dir, true)
}

func (g *Grid) TryConnectWith(v, dir Vec3i) {
	if g.Contains(v.Add(dir)) {
		g.setWall(v, dir, false)
	}
}

func (g *Grid) TryDisconnectWith(v, dir Vec3i) {
	if g.Contains(v.Add(dir)) {
		g.setWall(v, dir, true)
	}
}

func (g *Grid) NumConnections(x, y, z int) int {
	return g.numConnectionsImpl(x, y, z, true)
}

func (g *Grid) NumConnectionsSameLevel(x, y, z int) int {
	return g.numConnectionsImpl(x, y, z, false)
}

func (g *Grid) numConnectionsImpl(x, y, z int, countZ bool) int {
	it := g.CellAtXYZ(x, y, z)
	if it.Type == CellFull {
		return 0
	}

	res := 0
	if z > 0 && !it.HasFloor && countZ {
		res++
	}
	if x > 0 && !it.HasLeftWall {
		res++
	}
	if y > 0 && !it.HasTopWall {
		res++
	}

	if x+1 < g.NumCells.X {
		right := g.CellAtXYZ(x+1, y, z)
		if !right.HasLeftWall && right.Type != CellFull {
			res++
		}
	}
	if y+1 < g.NumCells.Y {
		bottom := g.CellAtXYZ(x, y+1, z)
		if !bottom.HasTopWall && bottom.Type != CellFull {
			res++
		}
	}
	if countZ && z+1 < g.NumCells.Z {
		above := g.CellAtXYZ(x, y, z+1)
		if !above.HasFloor && above.Type != CellFull {
			res++
		}
	}
	return res
}

func (g *Grid) CanBuildStair(pos Vec3i) bool {
	return g.CellAt(pos).Type.AvailableForStair()
}

func (g *Grid) CanBuildRoom(pos Box3i) bool {
	for x := pos.Min.X; x < pos.Max.X; x++ {
		for y := pos.Min.Y; y < pos.Max.Y; y++ {
			for z := pos.Min.Z; z < pos.Max.Z; z++ {
				if !g.CellAtXYZ(x, y, z).Type.AvailableForRoom() {
					return false
				}
			}
		}
	}
	return true
}

func (g *Grid) Close(v Vec3i) {
	g.TryDisconnectWith(v, Vec3i{1, 0, 0})
	g.TryDisconnectWith(v, Vec3i{-1, 0, 0})
	g.TryDisconnectWith(v, Vec3i{0, 1, 0})
	g.TryDisconnectWith(v, Vec3i{0, -1, 0})
	g.TryDisconnectWith(v, Vec3i{0, 0, 1})
	g.TryDisconnectWith(v, Vec3i{0, 0, -1})
}

func (g *Grid) Open(v Vec3i) {
	g.TryConnectWith(v, Vec3i{1, 0, 0})
	g.TryConnectWith(v, Vec3i{-1, 0, 0})
	g.TryConnectWith(v, Vec3i{0, 1, 0})
	g.TryConnectWith(v, Vec3i{0, -1, 0})
	g.TryConnectWith(v, Vec3i{0, 0, 1})
	g.TryConnectWith(v, Vec3i{0, 0, -1})
}

func (g *Grid) ClearArea(pos Box3i) {
	for x := pos.Min.X; x < pos.Max.X; x++ {
		for y := pos.Min.Y; y < pos.Max.Y; y++ {
			for z := pos.Min.Z; z < pos.Max.Z; z++ {
				p := Vec3i{x, y, z}
				c := g.CellAt(p)

				if z == pos.Min.Z {
					c.Type = CellRoomFloor
				} else {
					c.Type = CellAir
				}

				if z == pos.Min.Z && z > 1 {
					c.Balcony = BalconySimple
				}

				if z == pos.Min.Z {
					c.HasFloor = true
				}

				if x+1 < pos.Max.X {
					g.ConnectWith(p, Vec3i{1, 0, 0})
				}
				if y+1 < pos.Max.Y {
					g.ConnectWith(p, Vec3i{0, 1, 0})
				}
				if z+1 < pos.Max.Z {
					g.ConnectWith(p, Vec3i{0, 0, 1})
				}
			}
		}
	}

	for z := pos.Min.Z + 1; z < pos.Max.Z; z++ {
		for x := pos.Min.X - 1; x < pos.Max.X+1; x++ {
			for y := pos.Min.Y - 1; y < pos.Max.Y+1; y++ {
				if g.ContainsXYZ(x, y, z) {
					g.CellAtXYZ(x, y, z).Balcony = BalconySimple
				}
			}
		}
	}
}

func (g *Grid) GetBalconyMask(cellPos Vec3i) (isBalcony, left, right, top, bottom bool) {
	c := g.CellAt(cellPos)
	isBalcony = c.Balcony != BalconyNone
	if !isBalcony {
		return
	}

	if cellPos.X == 0 {
		left = true
	} else {
		l := g.CellAt(cellPos.Add(Vec3i{-1, 0, 0}))
		if l.Type == CellAir && c.Type != CellStairEndHigh {
			left = true
		}
	}

	if cellPos.X+1 == g.NumCells.X {
		right = true
	} else {
		r := g.CellAt(cellPos.Add(Vec3i{1, 0, 0}))
		if r.Type == CellAir && c.Type != CellStairEndHigh {
			right = true
		}
	}

	if cellPos.Y == 0 {
		top = true
	} else {
		t := g.CellAt(cellPos.Add(Vec3i{0, -1, 0}))
		if t.Type == CellAir && c.Type != CellStairEndHigh {
			top = true
		}
	}

	if cellPos.Y+1 == g.NumCells.Y {
		bottom = true
	} else {
		b := g.CellAt(cellPos.Add(Vec3i{0, 1, 0}))
		if b.Type == CellAir && c.Type != CellStairEndHigh {
			bottom = true
		}
	}

	return
}

func (g *Grid) CanSeeInside(pos Vec3i) bool {
	if !g.Contains(pos) {
		return true
	}
	c := g.CellAt(pos)
	if c.Type == CellFull {
		return false
	}
	if g.NumConnections(pos.X, pos.Y, pos.Z) == 0 {
		return false
	}
	return true
}
