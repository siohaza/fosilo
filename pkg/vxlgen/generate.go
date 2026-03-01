package vxlgen

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/siohaza/fosilo/pkg/vxl"
)

type Config struct {
	Seed   uint64
	Floors int // 0 = random [7, 11)
	CellsX int // 0 = random [21, 41)
	CellsY int // 0 = random [21, 41)
}

type Result struct {
	Map            *vxl.Map
	Seed           uint64
	TowerPos       Vec3i
	NumCells       Vec3i
	CellSize       Vec3i
	BlueSpawnArea  Box3i
	GreenSpawnArea Box3i
}

func Generate(cfg Config) *Result {
	seed := cfg.Seed
	if seed == 0 {
		var b [8]byte
		rand.Read(b[:])
		seed = binary.LittleEndian.Uint64(b[:])
	}

	rng := NewRng(seed)

	floors := cfg.Floors
	if floors == 0 {
		floors = rng.Dice(7, 11)
	}
	cellsX := cfg.CellsX
	if cellsX == 0 {
		cellsX = rng.Dice(21, 41)
	}
	cellsY := cfg.CellsY
	if cellsY == 0 {
		cellsY = rng.Dice(21, 41)
	}

	wrapper := NewMapWrapper()

	// generate terrain
	terrain := NewTerrain(Vec2i{MapSize, MapSize}, rng)
	terrain.BuildBlocks(rng, wrapper)

	// compute tower position
	cellSize := Vec3i{4, 4, 6}
	numCells := Vec3i{cellsX, cellsY, floors}
	dimensions := Vec3i{
		numCells.X*cellSize.X + 1,
		numCells.Y*cellSize.Y + 1,
		numCells.Z*cellSize.Z + 1,
	}
	towerPos := Vec3i{
		254 - dimensions.X/2,
		254 - dimensions.Y/2,
		1,
	}

	// generate tower
	tower := NewTower(towerPos, numCells)
	tower.BuildBlocks(rng, wrapper)

	// post-processing
	ColorBleed(wrapper)
	BetterAO(wrapper)
	ReverseClientAO(wrapper)

	blueSpawn := Box3i{
		Min: towerPos.Add(Vec3i{
			tower.BlueEntrance.Min.X * cellSize.X,
			tower.BlueEntrance.Min.Y * cellSize.Y,
			tower.BlueEntrance.Min.Z * cellSize.Z,
		}),
		Max: towerPos.Add(Vec3i{
			tower.BlueEntrance.Max.X * cellSize.X,
			tower.BlueEntrance.Max.Y * cellSize.Y,
			tower.BlueEntrance.Max.Z * cellSize.Z,
		}),
	}
	greenSpawn := Box3i{
		Min: towerPos.Add(Vec3i{
			tower.GreenEntrance.Min.X * cellSize.X,
			tower.GreenEntrance.Min.Y * cellSize.Y,
			tower.GreenEntrance.Min.Z * cellSize.Z,
		}),
		Max: towerPos.Add(Vec3i{
			tower.GreenEntrance.Max.X * cellSize.X,
			tower.GreenEntrance.Max.Y * cellSize.Y,
			tower.GreenEntrance.Max.Z * cellSize.Z,
		}),
	}

	return &Result{
		Map:            wrapper.M,
		Seed:           seed,
		TowerPos:       towerPos,
		NumCells:       numCells,
		CellSize:       cellSize,
		BlueSpawnArea:  blueSpawn,
		GreenSpawnArea: greenSpawn,
	}
}
