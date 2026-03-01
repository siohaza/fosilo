package vxlgen

type Room struct {
	Pos        Box3i
	IsEntrance bool
	CellSize   Vec3i
}

func NewRoom(pos Box3i, isEntrance bool, cellSize Vec3i) *Room {
	return &Room{
		Pos:        pos,
		IsEntrance: isEntrance,
		CellSize:   cellSize,
	}
}

func (r *Room) CellPosition() Vec3i {
	return r.Pos.Min
}

func (r *Room) BuildCells(rng *Rng, grid *Grid) {
	for x := r.Pos.Min.X; x < r.Pos.Max.X; x++ {
		for y := r.Pos.Min.Y; y < r.Pos.Max.Y; y++ {
			for z := r.Pos.Min.Z; z < r.Pos.Max.Z; z++ {
				p := Vec3i{x, y, z}
				c := grid.CellAt(p)

				if z == r.Pos.Min.Z {
					c.Type = CellRoomFloor
				} else {
					c.Type = CellAir
				}

				if z == r.Pos.Min.Z && grid.IsExternal(p) && !r.IsEntrance {
					c.Balcony = BalconySimple
				}

				if r.IsEntrance && z == r.Pos.Min.Z {
					c.HasFloor = true
				}

				if x+1 < r.Pos.Max.X {
					grid.ConnectWith(p, Vec3i{1, 0, 0})
				}
				if y+1 < r.Pos.Max.Y {
					grid.ConnectWith(p, Vec3i{0, 1, 0})
				}
				if z+1 < r.Pos.Max.Z {
					grid.ConnectWith(p, Vec3i{0, 0, 1})
				}
			}
		}
	}

	for z := r.Pos.Min.Z + 1; z < r.Pos.Max.Z; z++ {
		for x := r.Pos.Min.X - 1; x < r.Pos.Max.X+1; x++ {
			for y := r.Pos.Min.Y - 1; y < r.Pos.Max.Y+1; y++ {
				if grid.ContainsXYZ(x, y, z) {
					grid.CellAtXYZ(x, y, z).Balcony = BalconySimple
				}
			}
		}
	}
}

func (r *Room) BuildBlocks(rng *Rng, grid *Grid, base Vec3i, m *MapWrapper) {

}
