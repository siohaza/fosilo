package gamestate

import (
	"math/rand"
	"sync"
	"time"

	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
	"github.com/siohaza/fosilo/pkg/config"
	"github.com/siohaza/fosilo/pkg/vxl"
)

type GameState struct {
	Map              *vxl.Map
	MapConfig        *config.MapConfig
	Config           *config.Config
	Gamemode         config.GamemodeID
	Players          *player.Manager
	Team1Score       uint8
	Team2Score       uint8
	CaptureLimit     uint8
	Intel            [2]Intel
	IntelSpawnPos    [2]protocol.Vector3f
	Base             [2]protocol.Vector3f
	Grenades         []*Grenade
	RoundStartTime   time.Time
	TimeLimitReached bool
	rng              *rand.Rand
	mu               sync.RWMutex
}

type Intel struct {
	Position  protocol.Vector3f
	Held      bool
	CarrierID uint8
	Team      uint8
}

type Grenade struct {
	Position    protocol.Vector3f
	Velocity    protocol.Vector3f
	FuseLength  float32
	TimeCreated float64
	PlayerID    uint8
}

func New(cfg *config.Config, mapCfg *config.MapConfig, vxlMap *vxl.Map) *GameState {
	gamemodeID, err := config.ParseGamemode(cfg.Server.Gamemode)
	if err != nil {
		gamemodeID = config.GamemodeCTF
	}

	captureLimit := uint8(cfg.Server.CaptureLimit)
	if mapCfg.Extensions.CapLimit != nil {
		captureLimit = uint8(*mapCfg.Extensions.CapLimit)
	}

	gs := &GameState{
		Map:            vxlMap,
		MapConfig:      mapCfg,
		Config:         cfg,
		Players:        player.NewManager(),
		Gamemode:       gamemodeID,
		CaptureLimit:   captureLimit,
		Grenades:       make([]*Grenade, 0),
		RoundStartTime: time.Now(),
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	gs.initializeGamemode()
	return gs
}

func (gs *GameState) initializeGamemode() {
	switch gs.Gamemode {
	case config.GamemodeCTF:
		gs.initCTF()
	case config.GamemodeBabel:
		gs.initBabel()
	case config.GamemodeTDM, config.GamemodeTC, config.GamemodeArena:
		gs.initTDM()
	}
}

func (gs *GameState) initCTF() {
	gs.Team1Score = 0
	gs.Team2Score = 0

	team1IntelX := float32(gs.MapConfig.Intel.Team1Position[0])
	team1IntelY := float32(gs.MapConfig.Intel.Team1Position[1])
	team1IntelZ := float32(gs.MapConfig.Intel.Team1Position[2])

	team2IntelX := float32(gs.MapConfig.Intel.Team2Position[0])
	team2IntelY := float32(gs.MapConfig.Intel.Team2Position[1])
	team2IntelZ := float32(gs.MapConfig.Intel.Team2Position[2])

	team1BaseX := float32(gs.MapConfig.Intel.Team1Base[0])
	team1BaseY := float32(gs.MapConfig.Intel.Team1Base[1])
	team1BaseZ := float32(gs.MapConfig.Intel.Team1Base[2])

	team2BaseX := float32(gs.MapConfig.Intel.Team2Base[0])
	team2BaseY := float32(gs.MapConfig.Intel.Team2Base[1])
	team2BaseZ := float32(gs.MapConfig.Intel.Team2Base[2])

	team1IntelPos := protocol.Vector3f{X: team1IntelX, Y: team1IntelY, Z: team1IntelZ}
	team2IntelPos := protocol.Vector3f{X: team2IntelX, Y: team2IntelY, Z: team2IntelZ}

	gs.Intel[0] = Intel{
		Position: team1IntelPos,
		Held:     false,
		Team:     0,
	}

	gs.Intel[1] = Intel{
		Position: team2IntelPos,
		Held:     false,
		Team:     1,
	}

	gs.IntelSpawnPos[0] = team1IntelPos
	gs.IntelSpawnPos[1] = team2IntelPos

	gs.Base[0] = protocol.Vector3f{X: team1BaseX, Y: team1BaseY, Z: team1BaseZ}
	gs.Base[1] = protocol.Vector3f{X: team2BaseX, Y: team2BaseY, Z: team2BaseZ}
}

func (gs *GameState) initBabel() {
	gs.Team1Score = 0
	gs.Team2Score = 0

	// babel has a single center flag
	// use team1 position as center flag position
	centerFlagX := float32(gs.MapConfig.Intel.Team1Position[0])
	centerFlagY := float32(gs.MapConfig.Intel.Team1Position[1])
	centerFlagZ := float32(gs.MapConfig.Intel.Team1Position[2])

	// set bases for capture zones
	team1BaseX := float32(gs.MapConfig.Intel.Team1Base[0])
	team1BaseY := float32(gs.MapConfig.Intel.Team1Base[1])
	team1BaseZ := float32(gs.MapConfig.Intel.Team1Base[2])

	team2BaseX := float32(gs.MapConfig.Intel.Team2Base[0])
	team2BaseY := float32(gs.MapConfig.Intel.Team2Base[1])
	team2BaseZ := float32(gs.MapConfig.Intel.Team2Base[2])

	centerFlagPos := protocol.Vector3f{X: centerFlagX, Y: centerFlagY, Z: centerFlagZ}

	// intel[0] is the center flag
	gs.Intel[0] = Intel{
		Position: centerFlagPos,
		Held:     false,
		Team:     0, // center flag belongs to "team 0" for tracking purposes
	}

	// intel[1] is not used in babel, hide it far outside map bounds
	hiddenPos := protocol.Vector3f{X: 1e9, Y: 1e9, Z: 128}
	gs.Intel[1] = Intel{
		Position: hiddenPos,
		Held:     true, // mark as held so it's never picked up
		Team:     1,
	}

	gs.IntelSpawnPos[0] = centerFlagPos
	gs.IntelSpawnPos[1] = hiddenPos

	gs.Base[0] = protocol.Vector3f{X: team1BaseX, Y: team1BaseY, Z: team1BaseZ}
	gs.Base[1] = protocol.Vector3f{X: team2BaseX, Y: team2BaseY, Z: team2BaseZ}
}

func (gs *GameState) initTDM() {
	gs.Team1Score = 0
	gs.Team2Score = 0

	// tdm can have optional intel
	intelEnabled := gs.intelEnabled()
	if intelEnabled {
		// initialize intel like ctf
		team1IntelX := float32(gs.MapConfig.Intel.Team1Position[0])
		team1IntelY := float32(gs.MapConfig.Intel.Team1Position[1])
		team1IntelZ := float32(gs.MapConfig.Intel.Team1Position[2])

		team2IntelX := float32(gs.MapConfig.Intel.Team2Position[0])
		team2IntelY := float32(gs.MapConfig.Intel.Team2Position[1])
		team2IntelZ := float32(gs.MapConfig.Intel.Team2Position[2])

		team1IntelPos := protocol.Vector3f{X: team1IntelX, Y: team1IntelY, Z: team1IntelZ}
		team2IntelPos := protocol.Vector3f{X: team2IntelX, Y: team2IntelY, Z: team2IntelZ}

		gs.Intel[0] = Intel{
			Position: team1IntelPos,
			Held:     false,
			Team:     0,
		}

		gs.Intel[1] = Intel{
			Position: team2IntelPos,
			Held:     false,
			Team:     1,
		}

		gs.IntelSpawnPos[0] = team1IntelPos
		gs.IntelSpawnPos[1] = team2IntelPos
	} else {
		// no intel in tdm hide it far outside map bounds (thx piqueserver for this idea)
		hiddenPos := protocol.Vector3f{X: 1e9, Y: 1e9, Z: 128}
		gs.Intel[0] = Intel{Position: hiddenPos, Held: false, Team: 0}
		gs.Intel[1] = Intel{Position: hiddenPos, Held: false, Team: 1}
		gs.IntelSpawnPos[0] = hiddenPos
		gs.IntelSpawnPos[1] = hiddenPos
	}

	// set bases
	team1BaseX := float32(gs.MapConfig.Intel.Team1Base[0])
	team1BaseY := float32(gs.MapConfig.Intel.Team1Base[1])
	team1BaseZ := float32(gs.MapConfig.Intel.Team1Base[2])

	team2BaseX := float32(gs.MapConfig.Intel.Team2Base[0])
	team2BaseY := float32(gs.MapConfig.Intel.Team2Base[1])
	team2BaseZ := float32(gs.MapConfig.Intel.Team2Base[2])

	gs.Base[0] = protocol.Vector3f{X: team1BaseX, Y: team1BaseY, Z: team1BaseZ}
	gs.Base[1] = protocol.Vector3f{X: team2BaseX, Y: team2BaseY, Z: team2BaseZ}
}

func (gs *GameState) isValidSpawnPoint(x, y, z int) bool {
	if x < 0 || x >= gs.Map.Width() || y < 0 || y >= gs.Map.Height() {
		return false
	}

	if z < 0 || z >= gs.Map.Depth()-1 {
		return false
	}

	if z+1 < gs.Map.Depth() && !gs.Map.IsSolid(x, y, z+1) {
		return false
	}

	if gs.Map.IsSolid(x, y, z) {
		return false
	}
	if z-1 >= 0 && gs.Map.IsSolid(x, y, z-1) {
		return false
	}
	if z-2 >= 0 && gs.Map.IsSolid(x, y, z-2) {
		return false
	}

	return true
}

func (gs *GameState) findValidSpawnZ(x, y, groundZ int) int {
	spawnZ := groundZ - 2

	if gs.isValidSpawnPoint(x, y, spawnZ) {
		return spawnZ
	}

	for offset := 1; offset <= 10; offset++ {
		if groundZ-2-offset >= 0 && gs.isValidSpawnPoint(x, y, groundZ-2-offset) {
			return groundZ - 2 - offset
		}
		if groundZ-2+offset < gs.Map.Depth()-1 && gs.isValidSpawnPoint(x, y, groundZ-2+offset) {
			return groundZ - 2 + offset
		}
	}

	return groundZ - 2
}

func (gs *GameState) GetSpawnPosition(team uint8) protocol.Vector3f {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	var spawnPoints [][]float64
	if team == 0 {
		spawnPoints = gs.MapConfig.SpawnPoints.Team1Points
	} else {
		spawnPoints = gs.MapConfig.SpawnPoints.Team2Points
	}

	if len(spawnPoints) > 0 {
		// randomly select a spawn point
		point := spawnPoints[gs.rng.Intn(len(spawnPoints))]
		if len(point) >= 3 {
			pos := protocol.Vector3f{
				X: float32(point[0]) + 0.5,
				Y: float32(point[1]) + 0.5,
				Z: float32(point[2]) - 2.4,
			}
			return pos
		}
	}

	// fallback to spawn areas
	var spawnArea config.SpawnArea
	if team == 0 {
		spawnArea = gs.MapConfig.SpawnPoints.Team1
	} else {
		spawnArea = gs.MapConfig.SpawnPoints.Team2
	}

	waterLevel := float32(63.0)
	if gs.MapConfig.Water.Enabled {
		waterLevel = gs.MapConfig.Water.Level
	}

	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		x := float32(gs.rng.Intn(spawnArea.End[0]-spawnArea.Start[0]+1) + spawnArea.Start[0])
		y := float32(gs.rng.Intn(spawnArea.End[1]-spawnArea.Start[1]+1) + spawnArea.Start[1])
		groundZ := gs.Map.FindGroundLevel(int(x), int(y))
		spawnZ := gs.findValidSpawnZ(int(x), int(y), groundZ)

		if float32(groundZ) < waterLevel {
			return protocol.Vector3f{X: x + 0.5, Y: y + 0.5, Z: float32(spawnZ) - 0.4}
		}
	}

	x := float32(gs.rng.Intn(spawnArea.End[0]-spawnArea.Start[0]+1) + spawnArea.Start[0])
	y := float32(gs.rng.Intn(spawnArea.End[1]-spawnArea.Start[1]+1) + spawnArea.Start[1])
	groundZ := gs.Map.FindGroundLevel(int(x), int(y))
	spawnZ := gs.findValidSpawnZ(int(x), int(y), groundZ)

	return protocol.Vector3f{X: x + 0.5, Y: y + 0.5, Z: float32(spawnZ) - 0.4}
}

func (gs *GameState) PickupIntel(playerID uint8, team uint8) bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if team >= 2 {
		return false
	}

	oppositeTeam := 1 - team

	if gs.Intel[oppositeTeam].Held {
		return false
	}

	p, ok := gs.Players.Get(playerID)
	if !ok {
		return false
	}

	p.RLock()
	alive := p.Alive
	playerTeam := p.Team
	p.RUnlock()

	if !alive || playerTeam != team {
		return false
	}

	gs.Intel[oppositeTeam].Held = true
	gs.Intel[oppositeTeam].CarrierID = playerID

	p.Lock()
	p.HasIntel = true
	p.Unlock()

	return true
}

func (gs *GameState) DropIntel(team uint8, position protocol.Vector3f) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if team >= 2 {
		return
	}

	x := int(position.X)
	y := int(position.Y)
	z := int(position.Z)

	if x < 0 || x >= gs.Map.Width() {
		x = max(0, min(x, gs.Map.Width()-1))
		position.X = float32(x) + 0.5
	}
	if y < 0 || y >= gs.Map.Height() {
		y = max(0, min(y, gs.Map.Height()-1))
		position.Y = float32(y) + 0.5
	}
	if z < 0 || z >= gs.Map.Depth() {
		z = max(0, min(z, gs.Map.Depth()-1))
		position.Z = float32(z)
	}

	gs.Intel[team].Held = false
	gs.Intel[team].Position = position
	gs.Intel[team].CarrierID = 0
}

func (gs *GameState) CaptureIntel(playerID uint8, team uint8) bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if team >= 2 {
		return false
	}

	p, ok := gs.Players.Get(playerID)
	if !ok {
		return false
	}

	// for babel, check intel[0] (center flag)
	// for ctf/tdm, check opposite team's intel
	var intelToCheck uint8
	gm := gs.Gamemode
	if gm == config.GamemodeBabel {
		intelToCheck = 0
	} else {
		intelToCheck = 1 - team
	}

	if !gs.Intel[intelToCheck].Held || gs.Intel[intelToCheck].CarrierID != playerID {
		return false
	}

	// reset intel to spawn position
	gs.Intel[intelToCheck].Held = false
	gs.Intel[intelToCheck].Position = gs.IntelSpawnPos[intelToCheck]
	gs.Intel[intelToCheck].CarrierID = 0

	p.Lock()
	p.HasIntel = false
	p.Unlock()

	if team == 0 {
		gs.Team1Score++
	} else {
		gs.Team2Score++
	}

	return true
}

func (gs *GameState) IsIntelAtBase(team uint8) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if team >= 2 {
		return false
	}

	intel := gs.Intel[team]
	base := gs.Base[team]

	if intel.Held {
		return false
	}

	dx := intel.Position.X - base.X
	dy := intel.Position.Y - base.Y
	dz := intel.Position.Z - base.Z

	return dx*dx+dy*dy+dz*dz < 2.0
}

func (gs *GameState) AddGrenade(g *Grenade) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.Grenades = append(gs.Grenades, g)
}

func (gs *GameState) RemoveGrenade(index int) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if index < 0 || index >= len(gs.Grenades) {
		return
	}

	gs.Grenades = append(gs.Grenades[:index], gs.Grenades[index+1:]...)
}

func (gs *GameState) GetGrenades() []*Grenade {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	grenades := make([]*Grenade, len(gs.Grenades))
	copy(grenades, gs.Grenades)
	return grenades
}

func (gs *GameState) UpdateGrenades(updateFunc func(grenades []*Grenade) []int) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	toRemove := updateFunc(gs.Grenades)

	for i := len(toRemove) - 1; i >= 0; i-- {
		idx := toRemove[i]
		if idx >= 0 && idx < len(gs.Grenades) {
			gs.Grenades = append(gs.Grenades[:idx], gs.Grenades[idx+1:]...)
		}
	}
}

func (gs *GameState) GetStateData(playerID uint8) protocol.PacketStateData {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	protoGamemode := protocolGamemodeFor(gs.Gamemode)

	stateData := protocol.PacketStateData{
		PacketID: uint8(protocol.PacketTypeStateData),
		PlayerID: playerID,
		FogColor: protocol.Color3b{
			B: uint8(gs.MapConfig.Map.FogColor[2]),
			G: uint8(gs.MapConfig.Map.FogColor[1]),
			R: uint8(gs.MapConfig.Map.FogColor[0]),
		},
		Team1Color: protocol.Color3b{
			B: uint8(gs.Config.Teams.Team1.Color[2]),
			G: uint8(gs.Config.Teams.Team1.Color[1]),
			R: uint8(gs.Config.Teams.Team1.Color[0]),
		},
		Team2Color: protocol.Color3b{
			B: uint8(gs.Config.Teams.Team2.Color[2]),
			G: uint8(gs.Config.Teams.Team2.Color[1]),
			R: uint8(gs.Config.Teams.Team2.Color[0]),
		},
		Gamemode: protoGamemode,
	}

	copy(stateData.Team1Name[:], gs.Config.Teams.Team1.Name)
	copy(stateData.Team2Name[:], gs.Config.Teams.Team2.Name)

	if protoGamemode == protocol.GamemodeTypeCTF {
		intelEnabled := gs.intelEnabled()
		heldIntels := uint8(0)
		if intelEnabled && gs.Intel[0].Held {
			heldIntels |= 1
		}
		if intelEnabled && gs.Intel[1].Held {
			heldIntels |= 2
		}

		stateData.CTFState = protocol.CTFStateData{
			Team1Score:   gs.Team1Score,
			Team2Score:   gs.Team2Score,
			CaptureLimit: gs.CaptureLimit,
			HeldIntels:   heldIntels,
			CarrierIDs:   [2]uint8{255, 255},
			Team1Intel:   gs.Intel[0].Position,
			Team2Intel:   gs.Intel[1].Position,
			Team1Base:    gs.Base[0],
			Team2Base:    gs.Base[1],
		}
		if intelEnabled {
			stateData.CTFState.CarrierIDs[0] = gs.Intel[0].CarrierID
			stateData.CTFState.CarrierIDs[1] = gs.Intel[1].CarrierID
		}
	}
	if protoGamemode == protocol.GamemodeTypeTC {
		stateData.TCState.TerritoryCount = 0
	}

	return stateData
}

func protocolGamemodeFor(gm config.GamemodeID) protocol.GamemodeType {
	switch gm {
	case config.GamemodeTC:
		return protocol.GamemodeTypeTC
	default:
		return protocol.GamemodeTypeCTF
	}
}

func (gs *GameState) intelEnabled() bool {
	gm, _ := config.ParseGamemode(int(gs.Gamemode))
	if gm == config.GamemodeTDM && gs.Config.Server.RemoveIntel {
		return false
	}
	return true
}

func (gs *GameState) GetTeamScore(team uint8) uint8 {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if team == 0 {
		return gs.Team1Score
	}
	return gs.Team2Score
}

func (gs *GameState) GetIntelState(team uint8) (protocol.Vector3f, bool) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if team >= 2 {
		return protocol.Vector3f{}, false
	}

	return gs.Intel[team].Position, gs.Intel[team].Held
}

func (gs *GameState) GetBase(team uint8) protocol.Vector3f {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if team >= 2 {
		return protocol.Vector3f{}
	}

	return gs.Base[team]
}

func (gs *GameState) ResetScores() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Team1Score = 0
	gs.Team2Score = 0
}

func (gs *GameState) ResetIntel() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Intel[0].Held = false
	gs.Intel[0].Position = gs.IntelSpawnPos[0]
	gs.Intel[0].CarrierID = 0

	gs.Intel[1].Held = false
	gs.Intel[1].Position = gs.IntelSpawnPos[1]
	gs.Intel[1].CarrierID = 0
}

func (gs *GameState) HasWon(team uint8) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	score := gs.Team1Score
	if team == 1 {
		score = gs.Team2Score
	}

	return score >= gs.CaptureLimit
}

func (gs *GameState) IsTimeLimitReached() bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.TimeLimitReached {
		return false
	}

	if gs.MapConfig.Extensions.TimeLimit == nil {
		return false
	}

	timeLimit := time.Duration(*gs.MapConfig.Extensions.TimeLimit) * time.Second
	if time.Since(gs.RoundStartTime) >= timeLimit {
		gs.TimeLimitReached = true
		return true
	}
	return false
}

func (gs *GameState) GetRoundTime() time.Duration {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return time.Since(gs.RoundStartTime)
}
