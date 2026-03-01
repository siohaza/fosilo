package vxlgen

type MetaStructure int

const (
	MetaNormal MetaStructure = iota
	MetaArch
	MetaPyramid
	MetaFourPillars
	MetaCube
	MetaCross
)

type Level struct {
	GroundColorLight Vec3f
	GroundColorDark  Vec3f
	WallColor        Vec3f
	GroundPattern    PatternEx
	Balcony          BalconyType
}

func NewLevel(lvl int, rng *Rng, isRoof bool) *Level {
	return NewLevelWithColor(lvl, rng, rng.RandomColor(), isRoof)
}

func NewLevelWithColor(lvl int, rng *Rng, color Vec3f, isRoof bool) *Level {
	l := &Level{}
	l.GroundColorLight = Lerp3f(color, Vec3f{1, 1, 1}, float32(0.4+0.2*rng.Uniform()))
	l.GroundColorDark = Lerp3f(color, Vec3f{0, 0, 0}, float32(0.4+0.2*rng.Uniform()))
	l.WallColor = Lerp3f(color, Vec3f{0.5, 0.5, 0.5}, float32(0.4+0.2*rng.Uniform()))

	if lvl == 0 {
		l.GroundColorLight = l.GroundColorLight.Mulf(0.3)
		l.GroundColorDark = l.GroundColorDark.Mulf(0.3)
		l.WallColor = l.WallColor.Mulf(0.3)
	}

	if isRoof {
		l.GroundPattern = PatternEx{Pattern: PatternOnlyOne, NoiseAmount: 0.003}
		l.Balcony = BalconyBattlement
	} else {
		l.GroundPattern = PatternEx{
			Pattern:     Pattern(rng.Dice(int(PatternOnlyOne), int(patternCount))),
			SwapIJ:      rng.Bool(),
			SwapColors:  rng.Bool(),
			NoiseAmount: 0.008,
		}
		l.Balcony = BalconySimple
	}
	return l
}

type Tower struct {
	Position         Vec3i
	NumCells         Vec3i
	CellSize         Vec3i
	Dimension        Vec3i
	entranceRoomSize int
	BlueEntrance     Box3i
	GreenEntrance    Box3i
}

func NewTower(position, numCells Vec3i) *Tower {
	numCells.Z += 1 // for roof
	t := &Tower{
		Position: position,
		NumCells: numCells,
		CellSize: Vec3i{4, 4, 6},
	}
	t.Dimension = Vec3i{
		numCells.X*t.CellSize.X + 1,
		numCells.Y*t.CellSize.Y + 1,
		numCells.Z*t.CellSize.Z + 1,
	}

	minDim := numCells.X
	if numCells.Y < minDim {
		minDim = numCells.Y
	}
	if minDim >= 1 {
		t.entranceRoomSize = 1
	}
	if minDim >= 13 {
		t.entranceRoomSize = 3
	}
	if minDim >= 23 {
		t.entranceRoomSize = 5
	}
	return t
}

func (t *Tower) BuildBlocks(rng *Rng, m *MapWrapper) {
	levels := make([]*Level, 0, t.NumCells.Z+2)
	for l := 0; l < t.NumCells.Z-1; l++ {
		levels = append(levels, NewLevel(l, rng, false))
	}
	levels = append(levels, NewLevelWithColor(t.NumCells.Z-1, rng, Vec3f{0.85, 0.85, 0.85}, true))
	levels = append(levels, NewLevelWithColor(t.NumCells.Z, rng, Vec3f{0.85, 0.85, 0.85}, true))

	grid := NewGrid(t.NumCells)

	// generate rough map
	for i := 0; i < t.NumCells.X; i++ {
		for j := 0; j < t.NumCells.Y; j++ {
			for k := 0; k < t.NumCells.Z; k++ {
				c := grid.CellAtXYZ(i, j, k)
				c.HasLeftWall = rng.Uniform() < 0.6
				c.HasTopWall = rng.Uniform() < 0.6

				floorThreshold := 0.95
				if k == 0 {
					floorThreshold = 0.9
				}
				if k == t.NumCells.Z-1 {
					floorThreshold = 1.0
				}
				c.HasFloor = rng.Uniform() < floorThreshold
			}
		}
	}

	t.buildExternalCells(grid, levels)
	t.buildMetastructure(rng, grid)

	rooms := t.addRooms(rng, grid)
	stairs := t.addStairs(rng, grid, levels)

	t.ensureEachFloorConnected(rng, grid)
	t.removeUninterestingPatterns(rng, grid)

	// red water
	d := Vec3i{
		t.NumCells.X*t.CellSize.X + 1,
		t.NumCells.Y*t.CellSize.Y + 1,
		0,
	}
	red := Vec3f{80.0 / 255.0, 0, 0}
	for y := 0; y < d.Y; y++ {
		for x := 0; x < d.X; x++ {
			m.SetBlockF(t.Position.X+x, t.Position.Y+y, 0, red)
		}
	}

	// clear cells
	for lvl := 0; lvl < t.NumCells.Z; lvl++ {
		for cellX := 0; cellX < t.NumCells.X; cellX++ {
			for cellY := 0; cellY < t.NumCells.Y; cellY++ {
				t.clearCell(rng, grid, m, Vec3i{cellX, cellY, lvl}, levels[lvl])
			}
		}
	}

	// render cells
	for lvl := 0; lvl < t.NumCells.Z; lvl++ {
		for cellX := 0; cellX < t.NumCells.X; cellX++ {
			for cellY := 0; cellY < t.NumCells.Y; cellY++ {
				t.renderCell(rng, grid, m, Vec3i{cellX, cellY, lvl}, levels)
			}
		}
	}

	// build block structures
	for _, room := range rooms {
		cellPos := room.CellPosition()
		spos := t.Position.Add(Vec3i{cellPos.X * t.CellSize.X, cellPos.Y * t.CellSize.Y, cellPos.Z * t.CellSize.Z})
		room.BuildBlocks(rng, grid, spos, m)
	}
	for _, stair := range stairs {
		cellPos := stair.CellPosition()
		spos := t.Position.Add(Vec3i{cellPos.X * t.CellSize.X, cellPos.Y * t.CellSize.Y, cellPos.Z * t.CellSize.Z})
		stair.BuildBlocks(rng, grid, spos, m)
	}
}

func (t *Tower) buildExternalCells(grid *Grid, levels []*Level) {
	for x := 0; x < t.NumCells.X; x++ {
		for y := 0; y < t.NumCells.Y; y++ {
			for z := 0; z < t.NumCells.Z; z++ {
				p := Vec3i{x, y, z}
				c := grid.CellAt(p)

				if grid.IsExternal(p) {
					if x == 0 {
						c.HasLeftWall = false
					}
					if y == 0 {
						c.HasTopWall = false
					}
					if z <= 1 {
						c.Type = CellFull
					} else {
						c.Balcony = levels[z].Balcony
					}
				}

				if z+1 == t.NumCells.Z {
					c.HasLeftWall = false
					c.HasTopWall = false
				}
			}
		}
	}
}

func (t *Tower) buildMetastructure(rng *Rng, grid *Grid) {
	m := rng.Uniform()
	var mt MetaStructure
	switch {
	case m < 0.6:
		mt = MetaNormal
	case m < 0.7:
		mt = MetaCube
	case m < 0.8:
		mt = MetaCross
	case m < 0.9:
		mt = MetaFourPillars
	default:
		mt = MetaArch
	}

	nx := grid.NumCells.X
	ny := grid.NumCells.Y
	nz := grid.NumCells.Z
	x3 := (nx + 1) / 3
	y3 := (ny + 1) / 3
	x25 := (nx*2 + 2) / 5
	y25 := (ny*2 + 2) / 5
	x37 := (nx*3 + 3) / 7
	y37 := (ny*3 + 3) / 7
	z3 := (nz + 1) / 3
	z4 := (nz + 1) / 3

	switch mt {
	case MetaNormal:
		// no-op
	case MetaArch:
		var bb Box3i
		if rng.Bool() {
			bb.Min = Vec3i{0, y25, 1}
			bb.Max = Vec3i{nx, ny - y25, nz - z4}
		} else {
			bb.Min = Vec3i{x25, 0, 1}
			bb.Max = Vec3i{nx - x25, ny, nz - z4}
		}
		grid.ClearArea(bb)
	case MetaPyramid:
		// not implemented in original
	case MetaFourPillars:
		grid.ClearArea(Box3i{Min: Vec3i{0, y37, 1}, Max: Vec3i{nx, ny - y37, nz - z4}})
		grid.ClearArea(Box3i{Min: Vec3i{x37, 0, 1}, Max: Vec3i{nx - x37, ny, nz - z4}})
	case MetaCube:
		grid.ClearArea(Box3i{Min: Vec3i{x3, 0, z3}, Max: Vec3i{nx - x3, ny, nz - z3}})
		grid.ClearArea(Box3i{Min: Vec3i{0, y3, z3}, Max: Vec3i{nx, ny - y3, nz - z3}})
		grid.ClearArea(Box3i{Min: Vec3i{x3, y3, 1}, Max: Vec3i{nx - x3, ny - y3, nz}})
	case MetaCross:
		grid.ClearArea(Box3i{Min: Vec3i{0, 0, 1}, Max: Vec3i{x3, y3, nz}})
		grid.ClearArea(Box3i{Min: Vec3i{nx - x3, 0, 1}, Max: Vec3i{nx, y3, nz}})
		grid.ClearArea(Box3i{Min: Vec3i{nx - x3, ny - y3, 1}, Max: Vec3i{nx, ny, nz}})
		grid.ClearArea(Box3i{Min: Vec3i{0, ny - y3, 1}, Max: Vec3i{x3, ny, nz}})
	}
	_ = z4
}

func (t *Tower) ensureEachFloorConnected(rng *Rng, grid *Grid) {
	stack := make([]Vec3i, t.NumCells.X*t.NumCells.Y)
	directions := [4]Vec3i{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}}

	for lvl := 0; lvl < t.NumCells.Z; lvl++ {
		// reset colors
		for cellX := 0; cellX < t.NumCells.X; cellX++ {
			for cellY := 0; cellY < t.NumCells.Y; cellY++ {
				c := grid.CellAtXYZ(cellX, cellY, lvl)
				if c.Type.ShouldBeConnected() {
					c.Color = -1
				} else {
					c.Color = -2
				}
			}
		}

		numColors := 0
		var colorLookup []int
		stackIndex := 0

		// flood fill to assign colors
		for {
			foundUncolored := false
			for cellX := 0; cellX < t.NumCells.X; cellX++ {
				for cellY := 0; cellY < t.NumCells.Y; cellY++ {
					c := grid.CellAtXYZ(cellX, cellY, lvl)
					if c.Color != -1 {
						continue
					}
					foundUncolored = true
					color := numColors
					numColors++
					c.Color = color
					colorLookup = append(colorLookup, color)

					stack[0] = Vec3i{cellX, cellY, lvl}
					stackIndex = 1

					for stackIndex > 0 {
						stackIndex--
						p := stack[stackIndex]
						grid.CellAt(p).Color = color

						for _, dir := range directions {
							if grid.IsConnectedWith(p, dir) {
								other := grid.CellAt(p.Add(dir))
								if other.Color == -1 {
									stack[stackIndex] = p.Add(dir)
									stackIndex++
								}
							}
						}
					}
				}
			}
			if !foundUncolored {
				break
			}
		}

		// merge colors until single connected region
		coloursToEliminate := len(colorLookup) - 1
		firstX := 0

	eliminateColors:
		for coloursToEliminate > 0 {
			found := false
			for cellX := firstX; cellX < t.NumCells.X; cellX++ {
				for cellY := 0; cellY < t.NumCells.Y; cellY++ {
					p := Vec3i{cellX, cellY, lvl}
					c := grid.CellAt(p)

					tryOrder := rng.Bool()

					for k := 0; k < 2; k++ {
						tryRight := (k == 0) != tryOrder
						var dir Vec3i
						if tryRight {
							dir = Vec3i{1, 0, 0}
						} else {
							dir = Vec3i{0, 1, 0}
						}

						if c.Color != -2 && grid.Contains(p.Add(dir)) {
							other := grid.CellAt(p.Add(dir))
							if other.Color != -2 {
								colorA := colorLookup[c.Color]
								colorB := colorLookup[other.Color]
								if colorA != colorB {
									grid.ConnectWith(p, dir)
									minColor := colorA
									maxColor := colorB
									if colorB < colorA {
										minColor = colorB
										maxColor = colorA
									}
									firstX = cellX
									for i := range colorLookup {
										if colorLookup[i] == maxColor {
											colorLookup[i] = minColor
										}
									}
									coloursToEliminate--
									found = true
									continue eliminateColors
								}
							}
						}
					}
				}
			}
			if !found {
				break
			}
		}
	}
}

func (t *Tower) removeUninterestingPatterns(rng *Rng, grid *Grid) {
	for {
		found := false
		for z := 0; z < t.NumCells.Z; z++ {
			for x := 0; x < t.NumCells.X; x++ {
				for y := 0; y < t.NumCells.Y; y++ {
					if grid.NumConnections(x, y, z) == 1 {
						c := grid.CellAtXYZ(x, y, z)
						if c.Type == CellRegular {
							found = true
							grid.Close(Vec3i{x, y, z})
						}
					}
				}
			}
		}
		if !found {
			break
		}
	}
}

func (t *Tower) addRooms(rng *Rng, grid *Grid) []*Room {
	var rooms []*Room
	roomProportion := 0.09

	suitableCells := 0
	for x := 0; x < t.NumCells.X; x++ {
		for y := 0; y < t.NumCells.Y; y++ {
			for z := 0; z < t.NumCells.Z; z++ {
				if grid.CellAtXYZ(x, y, z).Type.AvailableForRoom() {
					suitableCells++
				}
			}
		}
	}

	roomCells := float64(suitableCells) * roomProportion

	tryRoom := func(bb Box3i, isEntrance bool) {
		if grid.CanBuildRoom(bb) {
			room := NewRoom(bb, isEntrance, t.CellSize)
			room.BuildCells(rng, grid)
			rooms = append(rooms, room)
			roomCells -= float64(bb.Volume())
		}
	}

	// build entrances
	if t.NumCells.X > 7 && t.NumCells.Y > 7 && t.NumCells.Z > 3 {
		entranceSize := Vec3i{4, 5, 3}
		middle := Vec3i{
			(t.NumCells.X - entranceSize.X) / 2,
			(t.NumCells.Y - entranceSize.Y) / 2,
			0,
		}

		east := Vec3i{t.NumCells.X - entranceSize.X, middle.Y, 1}
		west := Vec3i{0, middle.Y, 1}

		t.BlueEntrance = Box3i{Min: west, Max: west.Add(entranceSize)}
		t.GreenEntrance = Box3i{Min: east, Max: east.Add(entranceSize)}

		tryRoom(t.GreenEntrance, true)
		tryRoom(t.BlueEntrance, true)
	}

	for roomCells > 0 {
		maxWidth := MinI(t.NumCells.X, 7)
		maxDepth := MinI(t.NumCells.Y, 7)
		maxHeight := MinI(t.NumCells.Z, 10)

		roomWidth := rng.Dice(3, maxWidth)
		roomDepth := rng.Dice(3, maxDepth)
		roomHeight := 1
		for roomHeight < maxHeight && rng.Uniform() < 0.5 {
			roomHeight++
		}

		roomSize := Vec3i{roomWidth, roomDepth, roomHeight}
		pos := Vec3i{
			rng.Dice(0, 1+t.NumCells.X-roomSize.X),
			rng.Dice(0, 1+t.NumCells.Y-roomSize.Y),
			rng.Dice(0, 1+t.NumCells.Z-roomSize.Z),
		}
		tryRoom(Box3i{Min: pos, Max: pos.Add(roomSize)}, false)
	}

	return rooms
}

func (t *Tower) addStairs(rng *Rng, grid *Grid, levels []*Level) []*Stair {
	var stairs []*Stair

	for lvl := 0; lvl < t.NumCells.Z-1; lvl++ {
		suitableCells := 0
		for x := 0; x < t.NumCells.X; x++ {
			for y := 0; y < t.NumCells.Y; y++ {
				if grid.CellAtXYZ(x, y, lvl).Type.AvailableForStair() {
					suitableCells++
				}
			}
		}

		numStairInLevels := int(0.5 + 32*float64(suitableCells)/(63.0*63))
		stairRemaining := numStairInLevels
		maxAttempts := stairRemaining * 200

		for stairRemaining > 0 && maxAttempts > 0 {
			maxAttempts--

			var direction Vec3i
			if rng.Uniform() < 0.5 {
				direction = Vec3i{1, 0, 0}
			} else {
				direction = Vec3i{0, 1, 0}
			}
			if rng.Uniform() < 0.5 {
				direction = direction.Neg()
			}

			posA := Vec3i{rng.Dice(0, t.NumCells.X), rng.Dice(0, t.NumCells.Y), lvl}
			posB := posA.Add(direction)
			posC := posA.Sub(direction)
			posAboveD := posB.Add(direction).Add(Vec3i{0, 0, 1})

			tooNear := false
			for _, other := range stairs {
				diff := other.Start.Sub(posA)
				if AbsI(diff.X)+AbsI(diff.Y) < 2 {
					tooNear = true
					break
				}
			}

			if !tooNear && grid.Contains(posA) && grid.Contains(posB) && grid.Contains(posC) && grid.Contains(posAboveD) {
				if grid.CanBuildStair(posA) && grid.CanBuildStair(posB) && grid.CanBuildStair(posC) && grid.CanBuildStair(posAboveD) {
					stair := NewStair(posA, direction, levels[lvl+1].GroundColorDark, levels[lvl+1].GroundColorLight)
					stair.BuildCells(rng, grid)
					stairs = append(stairs, stair)
					stairRemaining--
				}
			}
		}
	}

	return stairs
}

func (t *Tower) clearCell(rng *Rng, grid *Grid, m *MapWrapper, cellPos Vec3i, level *Level) {
	bp := t.Position.Add(Vec3i{cellPos.X * t.CellSize.X, cellPos.Y * t.CellSize.Y, cellPos.Z * t.CellSize.Z})

	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			for k := 0; k < 7; k++ {
				px, py, pz := bp.X+i, bp.Y+j, bp.Z+k
				if m.Contains(px, py, pz) {
					m.EmptyBlock(px, py, pz)
				}
			}
		}
	}
}

func (t *Tower) renderCell(rng *Rng, grid *Grid, m *MapWrapper, cellPos Vec3i, levels []*Level) {
	bp := t.Position.Add(Vec3i{cellPos.X * t.CellSize.X, cellPos.Y * t.CellSize.Y, cellPos.Z * t.CellSize.Z})
	cellX := cellPos.X
	cellY := cellPos.Y
	lvl := cellPos.Z
	x := bp.X
	y := bp.Y
	z := bp.Z

	cell := grid.CellAt(cellPos)
	isBalcony, isBalconyLeft, isBalconyRight, isBalconyTop, isBalconyBottom := grid.GetBalconyMask(cellPos)
	canSeeInside := grid.CanSeeInside(cellPos)

	lightColor := levels[lvl].GroundColorLight
	darkColor := levels[lvl].GroundColorDark
	if cell.Type.IsStairPart() {
		lightColor = levels[lvl+1].GroundColorLight
		darkColor = levels[lvl+1].GroundColorDark
	}

	// cell ground
	if !grid.IsConnectedWith(cellPos, Vec3i{0, 0, -1}) {
		for i := 0; i < 5; i++ {
			for j := 0; j < 5; j++ {
				color := PatternColor(rng, levels[lvl].GroundPattern,
					i+cellX*4, j+cellY*4, lightColor, darkColor)
				m.SetBlockF(x+i, y+j, z, color)
			}
		}
	} else if cell.Type != CellAir {
		// create hole
		holePatterns := [5][25]int{
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0},
			{0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			{1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1},
			{0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		}

		p := rng.Uniform()
		var patIdx int
		switch {
		case p < 0.2:
			patIdx = 0
		case p < 0.4:
			patIdx = 1
		case p < 0.6:
			patIdx = 2
		case p < 0.8:
			patIdx = 3
		default:
			patIdx = 4
		}

		swapIJ := rng.Bool()
		reverseI := rng.Bool()
		reverseJ := rng.Bool()

		for i := 0; i < 5; i++ {
			for j := 0; j < 5; j++ {
				ii, jj := i, j
				if reverseI {
					ii = 4 - ii
				}
				if reverseJ {
					jj = 4 - jj
				}
				if swapIJ {
					ii, jj = jj, ii
				}

				if holePatterns[patIdx][ii*5+jj] != 0 {
					color := PatternColor(rng, levels[lvl].GroundPattern,
						i+cellX*4, j+cellY*4, lightColor, darkColor)
					m.SetBlockF(x+i, y+j, z, color)
				}
			}
		}
	}

	// no connections -> full block
	if grid.NumConnections(cellX, cellY, lvl) == 0 {
		for i := 1; i < 4; i++ {
			for j := 1; j < 4; j++ {
				for k := 1; k < 7; k++ {
					m.SetBlockF(x+i, y+j, z+k, levels[lvl].WallColor)
				}
			}
		}
	}

	wallBase := 1
	if lvl == 0 {
		wallBase = 0
	}

	// left wall
	if cell.HasLeftWall {
		wallColor := levels[lvl].WallColor
		leftCell := cellPos.Sub(Vec3i{1, 0, 0})
		if cell.Type.IsStairPart() || (grid.Contains(leftCell) && grid.CellAt(leftCell).Type.IsStairPart()) {
			wallColor = levels[lvl+1].WallColor
		}
		if cellX == 1 {
			wallColor = Grey(wallColor, 0.6)
		}
		for j := 0; j < 5; j++ {
			for k := wallBase; k < 7; k++ {
				m.SetBlockF(x, y+j, z+k, wallColor)
			}
		}
		if canSeeInside && grid.CanSeeInside(leftCell) {
			t.addWindows(rng, m, x, y, z, true)
		}
	}

	// top wall
	if cell.HasTopWall {
		wallColor := levels[lvl].WallColor
		topCell := cellPos.Sub(Vec3i{0, 1, 0})
		if cell.Type.IsStairPart() || (grid.Contains(topCell) && grid.CellAt(topCell).Type.IsStairPart()) {
			wallColor = levels[lvl+1].WallColor
		}
		if cellY == 1 {
			wallColor = Grey(wallColor, 0.6)
		}
		for i := 0; i < 5; i++ {
			for k := wallBase; k < 7; k++ {
				m.SetBlockF(x+i, y, z+k, wallColor)
			}
		}
		if canSeeInside && grid.CanSeeInside(topCell) {
			t.addWindows(rng, m, x, y, z, false)
		}
	}

	// full block
	if cell.Type == CellFull {
		fullColor := Grey(levels[lvl].WallColor, 0.7)
		for i := 0; i < 5; i++ {
			for j := 0; j < 5; j++ {
				for k := 0; k < 6; k++ {
					m.SetBlockF(x+i, y+j, z+k, fullColor)
				}
			}
		}
	}

	// balcony
	if isBalcony {
		balconyColorLight := Lerp3f(Grey(levels[lvl].WallColor, 0.4), Vec3f{1, 1, 1}, 0.6)
		balconyColorDark := Lerp3f(Grey(levels[lvl].WallColor, 0.7), Vec3f{0, 0, 0}, 0.6)

		for i := 0; i < 5; i++ {
			for j := 0; j < 5; j++ {
				wallSize := -1

				if !grid.IsConnectedWith(cellPos, Vec3i{0, 0, -1}) {
					if i == 0 && isBalconyLeft {
						wallSize = 1
					}
					if j == 0 && isBalconyTop {
						wallSize = 1
					}
					if i+1 == 5 && isBalconyRight {
						wallSize = 1
					}
					if j+1 == 5 && isBalconyBottom {
						wallSize = 1
					}
				}

				if cell.Balcony == BalconyBattlement {
					if (i^j)&1 != 0 {
						wallSize = -1
					}
				}

				for k := 0; k <= wallSize; k++ {
					if k == 0 {
						m.SetBlockF(x+i, y+j, z+k, balconyColorDark)
					} else {
						m.SetBlockF(x+i, y+j, z+k, balconyColorLight)
					}
				}
			}
		}

		if lvl%2 == 0 {
			if grid.NumConnections(cellX, cellY, lvl) == 0 {
				for i := 0; i < 5; i++ {
					for j := 0; j < 5; j++ {
						for k := 1; k < 6; k++ {
							m.SetBlockF(x+i, y+j, z+k, balconyColorDark)
						}
					}
				}
			}
		}
	}
}

func (t *Tower) addWindows(rng *Rng, m *MapWrapper, x, y, z int, isLeft bool) {
	// Window generation for walls
	if isLeft {
		if rng.Uniform() < 0.16 {
			m.EmptyBlock(x, y+2, z+3)
		} else if rng.Uniform() < 0.08 {
			m.EmptyBlock(x, y+1, z+3)
			m.EmptyBlock(x, y+3, z+3)
		} else if rng.Uniform() < 0.08 {
			m.EmptyBlock(x, y+1, z+3)
			m.EmptyBlock(x, y+2, z+3)
			m.EmptyBlock(x, y+3, z+3)
		} else if rng.Uniform() < 0.02 {
			m.EmptyBlock(x, y+1, z+3)
			m.EmptyBlock(x, y+2, z+2)
			m.EmptyBlock(x, y+3, z+3)
		} else if rng.Uniform() < 0.02 {
			m.EmptyBlock(x, y+1, z+2)
			m.EmptyBlock(x, y+2, z+3)
			m.EmptyBlock(x, y+3, z+2)
		} else if rng.Uniform() < 0.002 {
			m.EmptyBlock(x, y+1, z+1)
			m.EmptyBlock(x, y+3, z+1)
			if rng.Uniform() < 0.01 {
				m.EmptyBlock(x, y+2, z+2)
			}
			m.EmptyBlock(x, y+1, z+3)
			m.EmptyBlock(x, y+3, z+3)
		}
	} else {
		if rng.Uniform() < 0.16 {
			m.EmptyBlock(x+2, y, z+3)
		} else if rng.Uniform() < 0.08 {
			m.EmptyBlock(x+1, y, z+3)
			m.EmptyBlock(x+3, y, z+3)
		} else if rng.Uniform() < 0.08 {
			m.EmptyBlock(x+1, y, z+3)
			m.EmptyBlock(x+2, y, z+3)
			m.EmptyBlock(x+3, y, z+3)
		} else if rng.Uniform() < 0.02 {
			m.EmptyBlock(x+1, y, z+3)
			m.EmptyBlock(x+2, y, z+2)
			m.EmptyBlock(x+3, y, z+3)
		} else if rng.Uniform() < 0.02 {
			m.EmptyBlock(x+1, y, z+2)
			m.EmptyBlock(x+2, y, z+3)
			m.EmptyBlock(x+3, y, z+2)
		} else if rng.Uniform() < 0.002 {
			m.EmptyBlock(x+1, y, z+1)
			m.EmptyBlock(x+3, y, z+1)
			if rng.Uniform() < 0.01 {
				m.EmptyBlock(x+2, y, z+2)
			}
			m.EmptyBlock(x+1, y, z+3)
			m.EmptyBlock(x+3, y, z+3)
		}
	}
}
