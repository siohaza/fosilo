package server

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/siohaza/fosilo/internal/bans"
	"github.com/siohaza/fosilo/internal/callbacks"
	"github.com/siohaza/fosilo/internal/gamemode"
	"github.com/siohaza/fosilo/internal/gamestate"
	"github.com/siohaza/fosilo/internal/masterserver"
	"github.com/siohaza/fosilo/internal/network"
	"github.com/siohaza/fosilo/internal/physics"
	"github.com/siohaza/fosilo/internal/ping"
	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
	"github.com/siohaza/fosilo/internal/vote"
	"github.com/siohaza/fosilo/pkg/classicgen"
	"github.com/siohaza/fosilo/pkg/config"
	"github.com/siohaza/fosilo/pkg/lua"
	"github.com/siohaza/fosilo/pkg/vxl"

	"github.com/codecat/go-enet"
)

const (
	fallbackTerrainColor  uint32 = 0x674028
	spectatorTeamID       uint8  = 255
	spectatorClientTeamID uint8  = 2
)

func toInternalTeamID(team uint8) (uint8, bool) {
	switch team {
	case 0, 1:
		return team, true
	case spectatorClientTeamID, spectatorTeamID:
		return spectatorTeamID, true
	default:
		return 0, false
	}
}

func toNetworkTeamID(team uint8) uint8 {
	if team == spectatorTeamID {
		return spectatorClientTeamID
	}
	return team
}

type Server struct {
	config          *config.Config
	network         *network.Server
	gameState       *gamestate.GameState
	gameMode        gamemode.GameMode
	logger          *slog.Logger
	running         bool
	tickRate        time.Duration
	startTime       time.Time
	luaCommands     *lua.CommandManager
	voteManager     *vote.Manager
	banManager      *bans.Manager
	masterServers   []*masterserver.Client
	pingHandler     *ping.Handler
	currentMap      int
	activeMapName   string
	reportedMapName string
	callbacks       *callbacks.CallbackChain
	ctx             context.Context
	cancel          context.CancelFunc
}

func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	net, err := network.NewServer(cfg.Server.Port, cfg.Server.MaxPlayers, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create network server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	srv := &Server{
		config:   cfg,
		network:  net,
		logger:   logger,
		tickRate: time.Second / 60,
		ctx:      ctx,
		cancel:   cancel,
	}

	srv.banManager = bans.NewManager("data/bans.json")
	if err := srv.banManager.Load(); err != nil {
		logger.Warn("failed to load bans", "error", err)
	}

	srv.voteManager = vote.NewManager()
	srv.luaCommands = lua.NewCommandManager(logger)
	srv.callbacks = callbacks.NewCallbackChain()

	pingPort := cfg.Server.Port + 1
	listenAddr := fmt.Sprintf(":%d", pingPort)
	srv.pingHandler = ping.NewHandler(listenAddr, &ping.ServerInfo{
		Name:           cfg.Server.Name,
		PlayersCurrent: 0,
		PlayersMax:     cfg.Server.MaxPlayers,
		Map:            "",
		GameMode:       "",
		GameVersion:    "0.75",
	}, logger)

	return srv, nil
}

func (s *Server) Start() error {
	if err := s.loadMap(s.config.Server.Maps[0]); err != nil {
		return fmt.Errorf("failed to load map: %w", err)
	}

	gm, err := config.ParseGamemode(s.config.Server.Gamemode)
	if err != nil {
		return fmt.Errorf("invalid gamemode: %w", err)
	}

	api := lua.NewGameAPI(s.gameState)
	api.SetBanManager(s.banManager)
	api.SetServer(s)
	api.SetCommandManager(s.luaCommands)

	luaGamemodePath := fmt.Sprintf("scripts/gamemodes/%s.lua", gm.String())
	luaMode, err := gamemode.NewLuaGameMode(luaGamemodePath, s.gameState, api, s.logger)
	if err != nil {
		return fmt.Errorf("failed to load Lua gamemode: %w", err)
	}
	s.gameMode = luaMode
	s.logger.Info("loaded Lua game mode", "path", luaGamemodePath, "mode", s.gameMode.Name())

	if s.luaCommands != nil {
		if err := s.luaCommands.LoadCommands("scripts/commands", api); err != nil {
			s.logger.Warn("failed to load lua commands", "error", err)
		}
	}

	if err := s.network.Start(); err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	if err := s.pingHandler.Start(); err != nil {
		s.logger.Warn("failed to start ping handler", "error", err)
	} else {
		s.logger.Info("ping handler started", "port", s.config.Server.Port+1)
	}

	s.updatePingServerInfo()

	s.startTime = time.Now()
	s.running = true

	s.syncIntelPositions()

	s.logger.Info("server started", "name", s.config.Server.Name)

	if s.config.Server.Master {
		for _, host := range s.config.Server.MasterHosts {
			ms, err := masterserver.New(
				host.Host,
				host.Port,
				s.config.Server.Name,
				gm.String(),
				s.getReportedMapName(),
				s.config.Server.Port,
				s.config.Server.MaxPlayers,
				s.logger,
			)
			if err != nil {
				s.logger.Error("failed to create master server client", "host", host.Host, "error", err)
			} else {
				s.masterServers = append(s.masterServers, ms)
				ms.Enable()
				ms.Start()
				s.logger.Info("master server integration enabled", "host", host.Host)
			}
		}
	}

	go s.run()
	go s.startPeriodicAnnouncements()

	return nil
}

func (s *Server) Stop() {
	s.logger.Info("stopping server")

	if s.cancel != nil {
		s.cancel()
	}

	s.running = false

	if s.voteManager != nil {
		s.voteManager.Stop()
	}

	s.network.Stop()

	if s.pingHandler != nil {
		s.pingHandler.Stop()
	}

	for _, ms := range s.masterServers {
		ms.Disable()
		ms.Destroy()
	}

	s.logger.Info("server stopped")
}

func (s *Server) RegisterCallbacks(cb callbacks.Callbacks) {
	s.callbacks.Register(cb)
}

func (s *Server) ReloadCommands() error {
	if s.luaCommands == nil {
		return fmt.Errorf("command manager not initialized")
	}

	api := lua.NewGameAPI(s.gameState)
	api.SetBanManager(s.banManager)
	api.SetServer(s)
	api.SetCommandManager(s.luaCommands)

	if err := s.luaCommands.Reload("scripts/commands", api); err != nil {
		return fmt.Errorf("failed to reload commands: %w", err)
	}

	s.logger.Info("reloaded lua commands")
	return nil
}

func (s *Server) ReloadGamemode() error {
	gm, err := config.ParseGamemode(s.config.Server.Gamemode)
	if err != nil {
		return fmt.Errorf("invalid gamemode: %w", err)
	}

	api := lua.NewGameAPI(s.gameState)
	api.SetBanManager(s.banManager)
	api.SetServer(s)
	api.SetCommandManager(s.luaCommands)

	luaGamemodePath := fmt.Sprintf("scripts/gamemodes/%s.lua", gm.String())
	luaMode, err := gamemode.NewLuaGameMode(luaGamemodePath, s.gameState, api, s.logger)
	if err != nil {
		return fmt.Errorf("failed to load Lua gamemode: %w", err)
	}

	s.gameMode = luaMode
	s.logger.Info("reloaded Lua game mode", "path", luaGamemodePath, "mode", s.gameMode.Name())
	return nil
}

func (s *Server) GetConfigPassword(role string) string {
	switch role {
	case "manager":
		return s.config.Passwords.Manager
	case "admin":
		return s.config.Passwords.Admin
	case "moderator", "mod":
		return s.config.Passwords.Moderator
	case "guard":
		return s.config.Passwords.Guard
	case "trusted":
		return s.config.Passwords.Trusted
	default:
		return ""
	}
}

func (s *Server) GetCurrentMapName() string {
	if s.activeMapName != "" {
		return s.activeMapName
	}
	if len(s.config.Server.Maps) == 0 {
		return ""
	}
	if s.currentMap >= len(s.config.Server.Maps) {
		return ""
	}
	return s.config.Server.Maps[s.currentMap]
}

func (s *Server) getReportedMapName() string {
	if s.reportedMapName != "" {
		return s.reportedMapName
	}
	if len(s.config.Server.Maps) == 0 {
		return ""
	}
	if s.currentMap >= len(s.config.Server.Maps) {
		return ""
	}
	base, _ := splitMapSpec(s.config.Server.Maps[s.currentMap])
	if base != "" {
		return base
	}
	return s.config.Server.Maps[s.currentMap]
}

func (s *Server) GetServerName() string {
	return s.config.Server.Name
}

func (s *Server) GetUptime() time.Duration {
	if s.startTime.IsZero() {
		return 0
	}
	return time.Since(s.startTime)
}

func (s *Server) loadMap(mapName string) error {
	vxlMap, mapCfg, displayName, reportedName, err := s.prepareMapResources(mapName)
	if err != nil {
		return err
	}

	s.gameState = gamestate.New(s.config, mapCfg, vxlMap)
	s.activeMapName = displayName
	s.reportedMapName = reportedName
	s.logger.Info("map loaded", "spec", mapName, "display", displayName)

	return nil
}

func (s *Server) prepareMapResources(mapSpec string) (*vxl.Map, *config.MapConfig, string, string, error) {
	base, param := splitMapSpec(mapSpec)
	reportedName := base
	if reportedName == "" {
		reportedName = mapSpec
	}
	if strings.EqualFold(base, "classicgen") {
		seed, display := s.resolveClassicgenSeed(param)
		vxlMap, err := classicgen.Generate(seed)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to generate classicgen map (seed=%d): %w", seed, err)
		}
		mapCfg := config.DefaultMapConfig()
		mapCfg.Map.Author = "ClassicGen"
		mapCfg.Map.Description = fmt.Sprintf("Procedurally generated map (seed %d)", seed)

		team1Top := vxlMap.FindGroundLevel(56, 56)
		team2Top := vxlMap.FindGroundLevel(456, 456)

		if team1Top < 0 {
			team1Top = 32
		}
		if team2Top < 0 {
			team2Top = 32
		}

		mapCfg.Tents.Team1 = config.TentArea{
			Start: [3]int{32, 32, max(0, team1Top-5)},
			End:   [3]int{80, 80, min(63, team1Top+10)},
		}
		mapCfg.Tents.Team2 = config.TentArea{
			Start: [3]int{432, 432, max(0, team2Top-5)},
			End:   [3]int{480, 480, min(63, team2Top+10)},
		}

		mapCfg.SpawnPoints.Team1 = config.SpawnArea{
			Start: [3]int{32, 32, max(0, team1Top-5)},
			End:   [3]int{80, 80, min(63, team1Top+10)},
		}
		mapCfg.SpawnPoints.Team2 = config.SpawnArea{
			Start: [3]int{432, 432, max(0, team2Top-5)},
			End:   [3]int{480, 480, min(63, team2Top+10)},
		}

		setClassicgenIntelPositions(vxlMap, mapCfg)

		return vxlMap, mapCfg, display, "classicgen", nil
	}

	mapPath := fmt.Sprintf("maps/%s.vxl", mapSpec)

	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to read map file: %w", err)
	}

	vxlMap, err := vxl.Create(512, 512, 64, mapData)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to create VXL map: %w", err)
	}

	mapCfg, err := config.LoadMapConfig(mapPath)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to load map config: %w", err)
	}

	return vxlMap, mapCfg, mapSpec, reportedName, nil
}

type tentPlacement struct {
	intel [3]float64
	base  [3]float64
	x     int
	y     int
	z     int
}

func (tp tentPlacement) spawnPoints(vxlMap *vxl.Map) [][]float64 {
	offsets := []struct{ dx, dy int }{
		{0, 0}, {2, 0}, {-2, 0}, {0, 2}, {0, -2},
		{2, 2}, {2, -2}, {-2, 2}, {-2, -2},
	}

	points := make([][]float64, 0, len(offsets))
	for _, o := range offsets {
		px := clampInt(tp.x+o.dx, 0, vxlMap.Width()-1)
		py := clampInt(tp.y+o.dy, 0, vxlMap.Height()-1)
		points = append(points, []float64{
			float64(px),
			float64(py),
			float64(tp.z) + 2.0,
		})
	}
	return points
}

func setClassicgenIntelPositions(vxlMap *vxl.Map, mapCfg *config.MapConfig) {
	if vxlMap == nil || mapCfg == nil {
		return
	}

	placeTent := func(area config.TentArea) tentPlacement {
		x, y, top, slope := findTentPlacement(vxlMap, area)
		flattenTentArea(vxlMap, x, y, top)

		baseHeight := clampInt(top, 1, vxlMap.Depth()-2)
		heightOffset := 1 + min(slope, 2)

		intel := [3]float64{
			float64(x) + 0.5,
			float64(y) + 0.5,
			float64(baseHeight + heightOffset),
		}
		base := [3]float64{
			float64(x) + 0.5,
			float64(y) + 0.5,
			float64(baseHeight),
		}

		return tentPlacement{
			intel: intel,
			base:  base,
			x:     x,
			y:     y,
			z:     baseHeight,
		}
	}

	setSpawnArea := func(area *config.SpawnArea, placement tentPlacement) {
		if area == nil {
			return
		}
		r := 4
		startX := clampInt(placement.x-r, 0, vxlMap.Width()-1)
		endX := clampInt(placement.x+r, 0, vxlMap.Width()-1)
		startY := clampInt(placement.y-r, 0, vxlMap.Height()-1)
		endY := clampInt(placement.y+r, 0, vxlMap.Height()-1)
		area.Start = [3]int{
			startX,
			startY,
			max(0, placement.z-3),
		}
		area.End = [3]int{
			endX,
			endY,
			min(vxlMap.Depth()-1, placement.z+1),
		}
	}

	setTentArea := func(area *config.TentArea, placement tentPlacement) {
		if area == nil {
			return
		}
		r := 5
		area.Start = [3]int{
			clampInt(placement.x-r, 0, vxlMap.Width()-1),
			clampInt(placement.y-r, 0, vxlMap.Height()-1),
			max(0, placement.z-2),
		}
		area.End = [3]int{
			clampInt(placement.x+r, 0, vxlMap.Width()-1),
			clampInt(placement.y+r, 0, vxlMap.Height()-1),
			min(vxlMap.Depth()-1, placement.z+2),
		}
	}

	team1 := placeTent(mapCfg.Tents.Team1)
	team2 := placeTent(mapCfg.Tents.Team2)

	mapCfg.Intel.Team1Position = team1.intel
	mapCfg.Intel.Team1Base = team1.base
	mapCfg.Intel.Team2Position = team2.intel
	mapCfg.Intel.Team2Base = team2.base

	setSpawnArea(&mapCfg.SpawnPoints.Team1, team1)
	setSpawnArea(&mapCfg.SpawnPoints.Team2, team2)
	mapCfg.SpawnPoints.Team1Points = team1.spawnPoints(vxlMap)
	mapCfg.SpawnPoints.Team2Points = team2.spawnPoints(vxlMap)
	setTentArea(&mapCfg.Tents.Team1, team1)
	setTentArea(&mapCfg.Tents.Team2, team2)
}

func findTentPlacement(vxlMap *vxl.Map, area config.TentArea) (int, int, int, int) {
	width := vxlMap.Width()
	height := vxlMap.Height()

	startX := clampInt(min(area.Start[0], area.End[0]), 0, width-1)
	endX := clampInt(max(area.Start[0], area.End[0]), 0, width-1)
	startY := clampInt(min(area.Start[1], area.End[1]), 0, height-1)
	endY := clampInt(max(area.Start[1], area.End[1]), 0, height-1)

	centerX := (startX + endX) / 2
	centerY := (startY + endY) / 2

	bestScore := math.MaxInt32
	bestX, bestY := centerX, centerY
	bestZ := vxlMap.FindGroundLevel(centerX, centerY)
	bestSlope := math.MaxInt32
	if bestZ < 0 {
		bestZ = 32
	}

	sampleRadius := 2
	neighborOffsets := make([][2]int, 0, (sampleRadius*2+1)*(sampleRadius*2+1)-1)
	for dx := -sampleRadius; dx <= sampleRadius; dx++ {
		for dy := -sampleRadius; dy <= sampleRadius; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			neighborOffsets = append(neighborOffsets, [2]int{dx, dy})
		}
	}

	for x := startX; x <= endX; x++ {
		for y := startY; y <= endY; y++ {
			top := vxlMap.FindGroundLevel(x, y)
			if top < 0 {
				continue
			}

			maxDiff := 0
			for _, n := range neighborOffsets {
				nx := clampInt(x+n[0], 0, width-1)
				ny := clampInt(y+n[1], 0, height-1)
				ntop := vxlMap.FindGroundLevel(nx, ny)
				if ntop < 0 {
					continue
				}
				diff := abs(top - ntop)
				if diff > maxDiff {
					maxDiff = diff
				}
			}

			slopePenalty := maxDiff * 1000
			distancePenalty := (abs(x-centerX) + abs(y-centerY)) * 10
			score := slopePenalty + distancePenalty

			if score < bestScore || (score == bestScore && maxDiff < bestSlope) || (score == bestScore && maxDiff == bestSlope && top > bestZ) {
				bestScore = score
				bestX = x
				bestY = y
				bestZ = top
				bestSlope = maxDiff
			}
		}
	}

	if bestSlope == math.MaxInt32 {
		bestSlope = 0
	}
	return bestX, bestY, bestZ, bestSlope
}

func flattenTentArea(vxlMap *vxl.Map, centerX, centerY, centerZ int) {
	if vxlMap == nil {
		return
	}

	width := vxlMap.Width()
	height := vxlMap.Height()
	depth := vxlMap.Depth()

	targetHeight := clampInt(centerZ, 1, depth-2)
	radius := 4

	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			if abs(dx)+abs(dy) > radius {
				continue
			}

			cx := clampInt(centerX+dx, 0, width-1)
			cy := clampInt(centerY+dy, 0, height-1)

			distance := abs(dx) + abs(dy)
			desiredHeight := targetHeight
			if distance >= radius-1 {
				desiredHeight = max(targetHeight-1, 1)
			}
			desiredHeight = clampInt(desiredHeight, 1, depth-2)

			currentTop := vxlMap.FindTopBlock(cx, cy)
			if currentTop > desiredHeight {
				for z := desiredHeight + 1; z <= currentTop; z++ {
					vxlMap.SetAir(cx, cy, z)
				}
			} else if currentTop < desiredHeight {
				color := fallbackTerrainColor
				if currentTop >= 0 {
					if c := vxlMap.Get(cx, cy, currentTop); c != 0 {
						color = c
					}
				}
				start := max(currentTop+1, 0)
				for z := start; z <= desiredHeight; z++ {
					vxlMap.Set(cx, cy, z, color)
				}
			}
		}
	}
}

func splitMapSpec(spec string) (string, string) {
	parts := strings.SplitN(spec, ":", 2)
	base := strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		return base, strings.TrimSpace(parts[1])
	}
	return base, ""
}

func (s *Server) resolveClassicgenSeed(seedHint string) (uint32, string) {
	var seed uint32
	if seedHint != "" {
		if v, err := strconv.ParseUint(seedHint, 0, 32); err == nil {
			seed = uint32(v)
		} else {
			seed = crc32.ChecksumIEEE([]byte(seedHint))
		}
	}
	if seed == 0 {
		seed = uint32(time.Now().UnixNano())
	}
	displayName := fmt.Sprintf("classicgen (seed: %d)", seed)
	return seed, displayName
}

func (s *Server) run() {
	ticker := time.NewTicker(s.tickRate)
	defer ticker.Stop()

	worldUpdateTicker := time.NewTicker(time.Second / 10)
	defer worldUpdateTicker.Stop()

	for s.running {
		select {
		case <-s.ctx.Done():
			s.logger.Info("server context cancelled, exiting run loop")
			return

		case <-ticker.C:
			s.update()

		case <-worldUpdateTicker.C:
			s.sendWorldUpdate()
		}

		s.handleNetworkEvents()
	}
}

func (s *Server) update() {
	dt := float32(s.tickRate.Seconds())
	gameTime := float32(time.Since(s.startTime).Seconds())

	s.gameState.Players.ForEach(func(p *player.Player) {
		if p.IsAlive() {
			fallDamage := physics.MovePlayer(p, s.gameState.Map, dt, gameTime)
			if fallDamage > 0 {
				if s.damagePlayer(p.ID, uint8(fallDamage), p.GetPosition(), uint8(protocol.KillTypeFall)) {
					s.handleEnvironmentKill(p, protocol.KillTypeFall)
				}
			}

			s.checkWaterDamage(p)
			s.checkBoundaryDamage(p)
			s.checkIntelPickup(p)
			s.checkIntelCapture(p)
			s.checkRestock(p)

			s.gameMode.OnPlayerUpdate(p)
		}

		if p.UpdateReload() {
			s.sendWeaponReload(p)
			s.sendPlayerProperties(p)
		}

		if p.GetState() == player.PlayerStateDead && time.Now().After(p.GetRespawnTime()) {
			s.respawnPlayer(p.ID)
		}
	})

	s.updateGrenades(dt)

	if s.gameState.IsTimeLimitReached() {
		s.handleTimeLimitReached()
	}

	if luaMode, ok := s.gameMode.(*gamemode.LuaGameMode); ok {
		if err := luaMode.UpdateTimers(); err != nil {
			s.logger.Error("failed to update gamemode timers", "error", err)
		}
	}
}

func (s *Server) handleNetworkEvents() {
	for i := 0; i < 100; i++ {
		event, err := s.network.Service(0)
		if err != nil {
			s.logger.Error("network service error", "error", err)
			continue
		}

		if event.Type == network.EventTypeNone {
			break
		}

		switch event.Type {
		case network.EventTypeConnect:
			s.handleConnect(event.Peer)

		case network.EventTypeDisconnect:
			s.handleDisconnect(event.Peer)

		case network.EventTypeReceive:
			s.handlePacket(event.Peer, event.Data)
		}
	}
}

func (s *Server) handleConnect(peer enet.Peer) {
	ip := peer.GetAddress().String()

	if banned, ban := s.banManager.IsBanned(ip); banned {
		s.logger.Info("banned player attempted to connect", "ip", ip, "reason", ban.Reason)
		peer.DisconnectNow(uint32(protocol.DisconnectReasonBanned))
		return
	}

	playerID, ok := s.gameState.Players.FindFreeID(s.config.Server.MaxPlayers)
	if !ok {
		s.logger.Warn("server full, rejecting connection")
		peer.DisconnectNow(uint32(protocol.DisconnectReasonServerFull))
		return
	}

	p := player.New(playerID, peer)
	s.gameState.Players.Add(p)

	p.Lock()
	p.State = player.PlayerStateLoading
	p.Unlock()

	s.callbacks.OnConnect(playerID)

	s.logger.Info("starting initial packet send", "player", p.ID)
	if err := s.sendInitialPackets(p); err != nil {
		s.logger.Error("failed to send initial packets", "id", p.ID, "error", err)
	}

	for _, ms := range s.masterServers {
		ms.UpdatePlayerCount(uint8(s.gameState.Players.Count()))
	}

	s.updatePingServerInfo()

	s.logger.Info("player connected", "id", playerID, "address", ip)
}

func (s *Server) handleDisconnect(peer enet.Peer) {
	p, ok := s.gameState.Players.GetByPeer(peer)
	if !ok {
		return
	}

	s.logger.Info("player disconnected", "id", p.ID, "name", p.GetName())

	p.RLock()
	hasIntel := p.HasIntel
	team := p.Team
	p.RUnlock()

	if hasIntel {
		pos := p.GetPosition()
		groundPos := s.getGroundIntelDropPosition(pos)
		s.gameState.DropIntel(1-team, groundPos)
		s.broadcastIntelDrop(1-team, groundPos)
	}

	s.broadcastPlayerLeft(p.ID)

	s.voteManager.HandlePlayerDisconnect(p.ID)

	s.callbacks.OnDisconnect(p.ID)

	p.Lock()
	p.State = player.PlayerStateDisconnected
	p.Unlock()

	s.gameState.Players.Remove(p.ID)

	for _, ms := range s.masterServers {
		ms.UpdatePlayerCount(uint8(s.gameState.Players.Count()))
	}

	s.updatePingServerInfo()
}

func (s *Server) checkRateLimit(p *player.Player, packetType protocol.PacketType) bool {
	if !s.config.RateLimit.Enabled {
		return true
	}

	now := time.Now()

	p.Lock()
	defer p.Unlock()

	if now.Sub(p.LastRateLimitReset) >= time.Second {
		p.PacketCounts = make(map[protocol.PacketType]int)
		p.TotalPacketCount = 0
		p.LastRateLimitReset = now
	}

	p.TotalPacketCount++
	p.PacketCounts[packetType]++

	if p.TotalPacketCount > s.config.RateLimit.BurstSize {
		p.RateLimitViolations++
		s.logger.Warn("rate limit exceeded (burst)",
			"player", p.Name,
			"id", p.ID,
			"packets", p.TotalPacketCount,
			"violations", p.RateLimitViolations)

		if p.RateLimitViolations >= 5 {
			s.logger.Warn("disconnecting player for excessive rate limit violations",
				"player", p.Name,
				"id", p.ID)
			peer := p.Peer
			p.Unlock()
			peer.DisconnectLater(0)
			return false
		}
		return false
	}

	var perTypeLimit int
	switch packetType {
	case protocol.PacketTypePositionData:
		perTypeLimit = s.config.RateLimit.PositionPacketsPerSec
	case protocol.PacketTypeOrientationData:
		perTypeLimit = s.config.RateLimit.OrientPacketsPerSec
	case protocol.PacketTypeBlockAction, protocol.PacketTypeBlockLine:
		perTypeLimit = s.config.RateLimit.BlockPacketsPerSec
	default:
		perTypeLimit = s.config.RateLimit.PacketsPerSecond
	}

	if perTypeLimit > 0 && p.PacketCounts[packetType] > perTypeLimit {
		p.RateLimitViolations++
		s.logger.Warn("rate limit exceeded (per-type)",
			"player", p.Name,
			"id", p.ID,
			"type", packetType,
			"count", p.PacketCounts[packetType],
			"limit", perTypeLimit,
			"violations", p.RateLimitViolations)

		if p.RateLimitViolations >= 5 {
			s.logger.Warn("disconnecting player for excessive rate limit violations",
				"player", p.Name,
				"id", p.ID)
			peer := p.Peer
			p.Unlock()
			peer.DisconnectLater(0)
			return false
		}
		return false
	}

	return true
}

func (s *Server) handlePacket(peer enet.Peer, data []byte) {
	if len(data) < 1 {
		return
	}

	p, ok := s.gameState.Players.GetByPeer(peer)
	if !ok {
		return
	}

	packetType := protocol.PacketType(data[0])

	if !s.checkRateLimit(p, packetType) {
		return
	}

	switch packetType {
	case protocol.PacketTypePositionData:
		s.handlePositionData(p, data)

	case protocol.PacketTypeOrientationData:
		s.handleOrientationData(p, data)

	case protocol.PacketTypeInputData:
		s.handleInputData(p, data)

	case protocol.PacketTypeWeaponInput:
		s.handleWeaponInput(p, data)

	case protocol.PacketTypeHit:
		s.handleHit(p, data)

	case protocol.PacketTypeSetTool:
		s.handleSetTool(p, data)

	case protocol.PacketTypeSetColor:
		s.handleSetColor(p, data)

	case protocol.PacketTypeExistingPlayer:
		s.handleExistingPlayer(p, data)

	case protocol.PacketTypeBlockAction:
		s.handleBlockAction(p, data)

	case protocol.PacketTypeBlockLine:
		s.handleBlockLine(p, data)

	case protocol.PacketTypeGrenade:
		s.handleGrenade(p, data)

	case protocol.PacketTypeChatMessage:
		s.handleChatMessage(p, data)

	case protocol.PacketTypeWeaponReload:
		s.handleWeaponReload(p, data)

	case protocol.PacketTypeChangeTeam:
		s.handleChangeTeam(p, data)

	case protocol.PacketTypeChangeWeapon:
		s.handleChangeWeapon(p, data)

	case protocol.PacketTypeHandShakeReturn:
		s.logger.Info("received handshake return", "player", p.ID)
		s.handleHandshakeReturn(p, data)

	case protocol.PacketTypeVersionResponse:
		s.logger.Info("received version response", "player", p.ID)
		s.handleVersionResponse(p, data)

	case protocol.PacketTypeExtensionInfo:
		s.logger.Info("received extension info", "player", p.ID)
		s.handleExtensionInfo(p, data)

	default:
		s.logger.Warn("received unhandled packet", "type", packetType, "len", len(data))
	}
}

func (s *Server) handlePositionData(p *player.Player, data []byte) {
	var packet protocol.PacketPositionData
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	if !protocol.IsValidPosition(packet.X, packet.Y, packet.Z) {
		return
	}

	p.Lock()
	p.Position = protocol.Vector3f{X: packet.X, Y: packet.Y, Z: packet.Z}
	p.EyePos = protocol.Vector3f{X: packet.X, Y: packet.Y, Z: packet.Z}
	p.Velocity = protocol.Vector3f{X: 0, Y: 0, Z: 0}
	p.Unlock()
}

func (s *Server) handleOrientationData(p *player.Player, data []byte) {
	var packet protocol.PacketOrientationData
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	if !protocol.IsValidOrientation(packet.X, packet.Y, packet.Z) {
		return
	}

	p.SetOrientation(protocol.Vector3f{X: packet.X, Y: packet.Y, Z: packet.Z})
}

func (s *Server) handleInputData(p *player.Player, data []byte) {
	var packet protocol.PacketInputData
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	p.Lock()
	p.KeyStates = packet.KeyStates
	p.MoveForward = packet.KeyStates&protocol.KeyStateForward != 0
	p.MoveBackwards = packet.KeyStates&protocol.KeyStateBackward != 0
	p.MoveLeft = packet.KeyStates&protocol.KeyStateLeft != 0
	p.MoveRight = packet.KeyStates&protocol.KeyStateRight != 0
	p.Jumping = packet.KeyStates&protocol.KeyStateJump != 0
	p.Crouching = packet.KeyStates&protocol.KeyStateCrouch != 0
	p.Sneaking = packet.KeyStates&protocol.KeyStateSneak != 0
	p.Sprinting = packet.KeyStates&protocol.KeyStateSprint != 0
	p.Unlock()

	s.broadcastPacketExcept(&packet, p.ID, false)
}

func (s *Server) handleWeaponInput(p *player.Player, data []byte) {
	var packet protocol.PacketWeaponInput
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	p.Lock()
	p.PrimaryFire = packet.WeaponInput&protocol.WeaponInputPrimary != 0
	p.SecondaryFire = packet.WeaponInput&protocol.WeaponInputSecondary != 0
	p.Unlock()

	s.broadcastPacketExcept(&packet, p.ID, false)

	if p.PrimaryFire && p.CanShoot() {
		s.handleShot(p)
	}
}

func (s *Server) handleShot(p *player.Player) {
	if !p.Shoot() {
		return
	}

	s.callbacks.OnWeaponFire(p)

	pos := p.GetPosition()
	ori := p.GetOrientation()

	eyePos := protocol.Vector3f{
		X: pos.X,
		Y: pos.Y,
		Z: pos.Z - 0.3,
	}

	weapon := p.GetWeapon()
	pellets := protocol.GetPelletCount(weapon)
	spread := float32(0.01)
	if weapon == protocol.WeaponTypeShotgun {
		spread = 0.05
	}

	for i := 0; i < pellets; i++ {
		direction := ori

		if spread > 0 && pellets > 1 {
			spreadX := (float32(i%3) - 1.0) * spread
			spreadY := (float32(i/3) - 1.0) * spread
			direction.X += spreadX
			direction.Y += spreadY

			length := float32(math.Sqrt(float64(direction.X*direction.X + direction.Y*direction.Y + direction.Z*direction.Z)))
			if length > 0 {
				direction.X /= length
				direction.Y /= length
				direction.Z /= length
			}
		}

		s.processShot(p, eyePos, direction)
	}
}

func (s *Server) isValidTarget(shooter, target *player.Player) bool {
	if shooter == nil || target == nil {
		return false
	}
	return target.ID != shooter.ID && target.IsAlive() && target.GetTeam() != shooter.Team
}

func (s *Server) calculateDistance(v protocol.Vector3f) float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

func (s *Server) determineHitType(closestPoint, targetPos protocol.Vector3f) protocol.HitType {
	headZ := targetPos.Z - 1.0
	torsoZ := targetPos.Z - 0.5
	legsZ := targetPos.Z + 0.5

	if closestPoint.Z < headZ {
		return protocol.HitTypeHead
	} else if closestPoint.Z < torsoZ {
		return protocol.HitTypeTorso
	} else if closestPoint.Z < legsZ {
		return protocol.HitTypeArms
	}
	return protocol.HitTypeLegs
}

func (s *Server) checkPlayerHit(eyePos, direction protocol.Vector3f, target *player.Player, maxDistance float32) (bool, float32, protocol.HitType) {
	targetPos := target.GetPosition()

	toTarget := protocol.Vector3f{
		X: targetPos.X - eyePos.X,
		Y: targetPos.Y - eyePos.Y,
		Z: targetPos.Z - eyePos.Z,
	}

	distance := s.calculateDistance(toTarget)
	if distance > maxDistance {
		return false, 0, 0
	}

	dot := direction.X*toTarget.X + direction.Y*toTarget.Y + direction.Z*toTarget.Z
	if dot < 0 {
		return false, 0, 0
	}

	closestPoint := protocol.Vector3f{
		X: eyePos.X + direction.X*dot,
		Y: eyePos.Y + direction.Y*dot,
		Z: eyePos.Z + direction.Z*dot,
	}

	dx := closestPoint.X - targetPos.X
	dy := closestPoint.Y - targetPos.Y
	dz := closestPoint.Z - targetPos.Z
	distanceToLine := s.calculateDistance(protocol.Vector3f{X: dx, Y: dy, Z: dz})

	if distanceToLine > 0.4 {
		return false, 0, 0
	}

	hitType := s.determineHitType(closestPoint, targetPos)
	return true, distance, hitType
}

func (s *Server) findClosestPlayerHit(shooter *player.Player, eyePos, direction protocol.Vector3f, maxRange float32) (*player.Player, protocol.HitType, float32) {
	var closestPlayer *player.Player
	closestDistance := maxRange
	var hitType protocol.HitType

	s.gameState.Players.ForEach(func(target *player.Player) {
		if !s.isValidTarget(shooter, target) {
			return
		}

		hit, distance, ht := s.checkPlayerHit(eyePos, direction, target, closestDistance)
		if hit {
			closestPlayer = target
			closestDistance = distance
			hitType = ht
		}
	})

	return closestPlayer, hitType, closestDistance
}

func (s *Server) handlePlayerKill(shooter, victim *player.Player, killType protocol.KillType) {
	if shooter != nil {
		shooter.Lock()
		shooter.Kills++
		shooter.Unlock()
	}

	s.gameMode.OnPlayerKill(shooter, victim, killType)
	s.callbacks.OnPlayerKill(shooter, victim, killType)

	killerID := victim.ID
	if shooter != nil {
		killerID = shooter.ID
	}
	s.broadcastKillAction(victim.ID, killerID, killType)

	s.checkWinConditionAndRotate()
}

func (s *Server) checkWinConditionAndRotate() {
	won, winningTeam := s.gameMode.CheckWinCondition()
	if won {
		s.gameState.ResetScores()
		s.broadcastChat(fmt.Sprintf("%s team wins!", s.getTeamName(winningTeam)), protocol.ChatTypeSystem)

		if s.gameMode.ShouldRotateMap() {
			time.Sleep(5 * time.Second)
			s.rotateMap()
		}
	}
}

func (s *Server) broadcastBlockDestruction(shooterID uint8, hitBlock protocol.Vector3i) {
	blockPacket := protocol.PacketBlockAction{
		PacketID: uint8(protocol.PacketTypeBlockAction),
		PlayerID: shooterID,
		Action:   protocol.BlockActionTypeSpadeGunDestroy,
		X:        hitBlock.X,
		Y:        hitBlock.Y,
		Z:        hitBlock.Z,
	}
	s.broadcastPacket(&blockPacket, true)
}

func (s *Server) processShot(shooter *player.Player, eyePos, direction protocol.Vector3f) {
	maxRange := float32(128.0)

	closestPlayer, hitType, _ := s.findClosestPlayerHit(shooter, eyePos, direction, maxRange)
	hit, hitPos, hitBlock, _ := physics.RaycastVXL(s.gameState.Map, eyePos, direction, maxRange)

	if closestPlayer != nil {
		targetPos := closestPlayer.GetPosition()
		playerDistance := s.calculateDistance(protocol.Vector3f{
			X: targetPos.X - eyePos.X,
			Y: targetPos.Y - eyePos.Y,
			Z: targetPos.Z - eyePos.Z,
		})

		terrainDistance := s.calculateDistance(protocol.Vector3f{
			X: hitPos.X - eyePos.X,
			Y: hitPos.Y - eyePos.Y,
			Z: hitPos.Z - eyePos.Z,
		})

		if !hit || playerDistance < terrainDistance {
			damage := physics.CalculateDamage(shooter.Weapon, hitType, playerDistance*playerDistance)
			if s.damagePlayer(closestPlayer.ID, damage, eyePos, 1) {
				killType := protocol.KillTypeWeapon
				if hitType == protocol.HitTypeHead {
					killType = protocol.KillTypeHeadshot
				}
				s.handlePlayerKill(shooter, closestPlayer, killType)
			}
		}
	} else if hit {
		s.broadcastBlockDestruction(shooter.ID, hitBlock)
	}
}

func (s *Server) validateHitTarget(attacker, target *player.Player) bool {
	if !target.IsAlive() || !attacker.IsAlive() {
		return false
	}

	if target.GetTeam() == attacker.GetTeam() && target.ID != attacker.ID {
		return false
	}

	return true
}

func (s *Server) validateWeaponState(p *player.Player, hitType protocol.HitType) bool {
	p.RLock()
	tool := p.Tool
	reloading := p.Reloading
	magazineAmmo := p.MagazineAmmo
	name := p.Name
	p.RUnlock()

	if hitType == protocol.HitTypeMelee {
		if tool != protocol.ItemTypeSpade {
			s.logger.Warn("melee hit with wrong tool", "player", name, "tool", tool)
			return false
		}
	} else {
		if tool != protocol.ItemTypeGun {
			s.logger.Warn("weapon hit with wrong tool", "player", name, "tool", tool)
			return false
		}

		if reloading {
			s.logger.Warn("hit while reloading", "player", name)
			return false
		}

		if magazineAmmo == 0 {
			s.logger.Warn("hit with no ammo", "player", name)
			return false
		}
	}

	return true
}

func (s *Server) validateWeaponRange(p *player.Player, weapon protocol.WeaponType, pos, targetPos protocol.Vector3f, distance float32) bool {
	maxWeaponRange := float32(128.0)
	if weapon == protocol.WeaponTypeShotgun {
		maxWeaponRange = 64.0
	}

	if distance > maxWeaponRange {
		s.logger.Warn("weapon range exceeded",
			"player", p.GetName(),
			"distance", distance,
			"max", maxWeaponRange,
			"weapon", weapon)
		return false
	}

	eyePos := protocol.Vector3f{
		X: pos.X,
		Y: pos.Y,
		Z: pos.Z + 1.0,
	}

	targetEyePos := protocol.Vector3f{
		X: targetPos.X,
		Y: targetPos.Y,
		Z: targetPos.Z + 1.0,
	}

	direction := protocol.Vector3f{
		X: targetEyePos.X - eyePos.X,
		Y: targetEyePos.Y - eyePos.Y,
		Z: targetEyePos.Z - eyePos.Z,
	}
	length := s.calculateDistance(direction)
	if length > 0 {
		direction.X /= length
		direction.Y /= length
		direction.Z /= length
	}

	hit, _, _, _ := physics.RaycastVXL(s.gameState.Map, eyePos, direction, distance)
	if hit {
		s.logger.Debug("hit blocked by terrain",
			"player", p.GetName(),
			"target", p.GetName())
		return false
	}

	return true
}

func (s *Server) handleHit(p *player.Player, data []byte) {
	var packet protocol.PacketHit
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	target, ok := s.gameState.Players.Get(packet.PlayerID)
	if !ok {
		return
	}

	if !s.validateHitTarget(p, target) {
		return
	}

	if !s.validateWeaponState(p, packet.HitType) {
		return
	}

	pos := p.GetPosition()
	targetPos := target.GetPosition()
	distance := s.calculateDistance(protocol.Vector3f{
		X: targetPos.X - pos.X,
		Y: targetPos.Y - pos.Y,
		Z: targetPos.Z - pos.Z,
	})

	p.RLock()
	weapon := p.Weapon
	p.RUnlock()

	if packet.HitType != protocol.HitTypeMelee {
		if !s.validateWeaponRange(p, weapon, pos, targetPos, distance) {
			return
		}
	}

	damage := physics.CalculateDamage(weapon, packet.HitType, distance)

	target, ok = s.gameState.Players.Get(packet.PlayerID)
	if !ok || target == nil {
		return
	}

	if s.damagePlayer(target.ID, damage, pos, 1) {
		killType := protocol.KillTypeWeapon
		if packet.HitType == protocol.HitTypeHead {
			killType = protocol.KillTypeHeadshot
		}
		s.handlePlayerKill(p, target, killType)
	}
}

func (s *Server) handleSetTool(p *player.Player, data []byte) {
	var packet protocol.PacketSetTool
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	p.Lock()
	wasReloading := p.Reloading && p.Tool == protocol.ItemTypeGun
	p.Tool = packet.Tool
	p.Unlock()

	if wasReloading && packet.Tool != protocol.ItemTypeGun {
		p.Lock()
		p.Reloading = false
		p.Unlock()

		var reloadPacket protocol.PacketWeaponReload
		reloadPacket.PacketID = uint8(protocol.PacketTypeWeaponReload)
		reloadPacket.PlayerID = p.ID
		p.RLock()
		reloadPacket.MagazineAmmo = p.MagazineAmmo
		reloadPacket.ReserveAmmo = p.ReserveAmmo
		p.RUnlock()
		s.sendPacket(p, &reloadPacket, true)
	}

	s.broadcastPacketExcept(&packet, p.ID, true)
}

func (s *Server) handleSetColor(p *player.Player, data []byte) {
	var packet protocol.PacketSetColor
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	p.Lock()
	p.Color = packet.Color
	p.Unlock()

	s.broadcastPacketExcept(&packet, p.ID, true)
}

func (s *Server) handleExistingPlayer(p *player.Player, data []byte) {
	s.logger.Debug("received ExistingPlayer packet", "player", p.ID, "state", p.GetState())

	if len(data) < 13 {
		s.logger.Error("ExistingPlayer packet too small", "player", p.ID, "len", len(data))
		return
	}

	playerState := p.GetState()
	if playerState != player.PlayerStateWaitingForExistingPlayer && p.GetTeam() != spectatorTeamID {
		s.logger.Warn("received ExistingPlayer in wrong state", "player", p.ID, "state", p.GetState())
		return
	}

	playerID := data[1]
	teamRaw := data[2]
	team, ok := toInternalTeamID(teamRaw)
	if !ok {
		s.logger.Warn("received ExistingPlayer with invalid team", "player", p.ID, "team", teamRaw)
		return
	}
	weapon := protocol.WeaponType(data[3])
	item := protocol.ItemType(data[4])
	kills := binary.LittleEndian.Uint32(data[5:9])
	color := protocol.Color3b{
		B: data[9],
		G: data[10],
		R: data[11],
	}

	nameBytes := data[12:]
	name, err := protocol.CP437ToString(nameBytes)
	if err != nil {
		name = "Unknown"
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = "Deuce"
	}

	_ = playerID
	_ = kills

	if banned, ban := s.banManager.IsBannedByName(name); banned {
		s.logger.Info("banned player attempted to join", "name", name, "reason", ban.Reason)
		p.Peer.DisconnectNow(uint32(protocol.DisconnectReasonBanned))
		return
	}

	p.SetTeam(team)
	p.SetWeapon(weapon)

	p.Lock()
	p.Name = name
	p.Tool = item
	p.Color = color
	p.State = player.PlayerStateReady
	p.HasIntel = false
	p.Unlock()

	s.logger.Info("player joined", "player", p.ID, "name", name, "team", team)
	s.finalizePlayerJoin(p)
}

func (s *Server) handleBlockAction(p *player.Player, data []byte) {
	var packet protocol.PacketBlockAction
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	x, y, z := int(packet.X), int(packet.Y), int(packet.Z)

	if !s.gameState.Map.IsInside(x, y, z) {
		return
	}

	if packet.Action == protocol.BlockActionTypeBuild {
		if z >= s.gameState.Map.Depth()-2 {
			return
		}

		if !s.gameState.Map.HasNeighbors(x, y, z) {
			return
		}

		now := time.Now()
		p.Lock()
		if now.Sub(p.LastBlockPlaceTime) >= 100*time.Millisecond {
			p.BlockPlaceQuota = 4
			p.LastBlockPlaceTime = now
		}

		if p.BlockPlaceQuota <= 0 {
			p.Unlock()
			s.logger.Warn("block place rate limit exceeded", "player", p.GetName())
			return
		}

		p.BlockPlaceQuota--
		p.Unlock()

		if p.Blocks > 0 {
			p.Lock()
			p.Blocks--
			colorRGB := p.Color
			p.Unlock()

			color := uint32(colorRGB.R)<<16 | uint32(colorRGB.G)<<8 | uint32(colorRGB.B)
			s.gameState.Map.Set(x, y, z, color)

			s.broadcastPacket(&packet, true)
		}
	} else {
		now := time.Now()
		p.Lock()
		if now.Sub(p.LastBlockDestroyTime) >= 100*time.Millisecond {
			p.BlockDestroyQuota = 8
			p.LastBlockDestroyTime = now
		}

		if p.BlockDestroyQuota <= 0 {
			p.Unlock()
			s.logger.Warn("block destroy rate limit exceeded", "player", p.GetName())
			return
		}

		p.BlockDestroyQuota--
		p.Unlock()

		if s.gameState.Map.IsSolid(x, y, z) {
			if packet.Action == protocol.BlockActionTypeSpadeGunDestroy {
				p.Lock()
				if p.Blocks < 50 {
					p.Blocks++
				}
				p.Unlock()
			} else if packet.Action == protocol.BlockActionTypeSpadeSecondaryDestroy {
				blocksDestroyed := 0
				if s.gameState.Map.IsSolid(x, y, z) {
					blocksDestroyed++
				}
				if s.gameState.Map.IsSolid(x, y, z-1) {
					blocksDestroyed++
				}
				if s.gameState.Map.IsSolid(x, y, z+1) {
					blocksDestroyed++
				}
				p.Lock()
				for i := 0; i < blocksDestroyed && p.Blocks < 50; i++ {
					p.Blocks++
				}
				p.Unlock()
			}
		}

		s.gameState.Map.SetAir(x, y, z)
		s.broadcastPacket(&packet, true)
	}
}

func (s *Server) handleBlockLine(p *player.Player, data []byte) {
	var packet protocol.PacketBlockLine
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	x1, y1, z1 := int(packet.StartX), int(packet.StartY), int(packet.StartZ)
	x2, y2, z2 := int(packet.EndX), int(packet.EndY), int(packet.EndZ)

	if !s.gameState.Map.IsInside(x1, y1, z1) || !s.gameState.Map.IsInside(x2, y2, z2) {
		return
	}

	if z1 >= s.gameState.Map.Depth()-2 || z2 >= s.gameState.Map.Depth()-2 {
		return
	}

	if !s.gameState.Map.HasNeighbors(x1, y1, z1) || !s.gameState.Map.HasNeighbors(x2, y2, z2) {
		return
	}

	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	dz := abs(z2 - z1)
	maxLen := max(max(dx, dy), dz)

	if maxLen > 64 {
		s.logger.Warn("block line too long", "player", p.GetName(), "length", maxLen)
		return
	}

	blocksNeeded := maxLen + 1
	if blocksNeeded > 255 {
		return
	}

	p.Lock()
	if p.Blocks < uint8(blocksNeeded) {
		p.Unlock()
		return
	}
	p.Blocks -= uint8(blocksNeeded)
	colorRGB := p.Color
	p.Unlock()

	color := uint32(colorRGB.R)<<16 | uint32(colorRGB.G)<<8 | uint32(colorRGB.B)

	if maxLen == 0 {
		s.gameState.Map.Set(x1, y1, z1, color)
	} else {
		steps := maxLen
		for i := 0; i <= steps; i++ {
			x := x1 + (x2-x1)*i/steps
			y := y1 + (y2-y1)*i/steps
			z := z1 + (z2-z1)*i/steps
			s.gameState.Map.Set(x, y, z, color)
		}
	}

	s.broadcastPacket(&packet, true)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func (s *Server) handleGrenade(p *player.Player, data []byte) {
	var packet protocol.PacketGrenade
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	if p.Grenades > 0 {
		p.Lock()
		p.Grenades--
		p.Unlock()

		s.callbacks.OnGrenadeToss(p)

		grenade := &gamestate.Grenade{
			Position:    protocol.Vector3f{X: packet.X, Y: packet.Y, Z: packet.Z},
			Velocity:    protocol.Vector3f{X: packet.VX, Y: packet.VY, Z: packet.VZ},
			FuseLength:  packet.FuseLength,
			TimeCreated: float64(time.Now().UnixNano()) / 1e9,
			PlayerID:    p.ID,
		}
		s.gameState.AddGrenade(grenade)

		s.broadcastPacketExcept(&packet, p.ID, true)
	}
}

func (s *Server) handleChatMessage(p *player.Player, data []byte) {
	var packet protocol.PacketChatMessage
	if err := packet.Read(data); err != nil {
		return
	}

	trimmed := bytes.TrimRight(packet.Message, "\x00")
	message := string(trimmed)
	message = strings.TrimSpace(message)

	if message == "" {
		return
	}

	s.logger.Info("chat message", "player", p.GetName(), "message", message)

	if p.Muted {
		s.sendChatToPlayer(p, "You are muted and cannot send messages.")
		return
	}

	if s.handleCommand(p, message) {
		return
	}

	if packet.Type == protocol.ChatTypeTeam {
		playerTeam := p.GetTeam()
		s.gameState.Players.ForEach(func(target *player.Player) {
			if target.GetTeam() == playerTeam && target.GetState() == player.PlayerStateReady {
				target.RLock()
				peer := target.Peer
				target.RUnlock()
				data, err := marshalPacket(&packet)
				if err == nil {
					s.network.SendPacket(peer, data, true)
				}
			}
		})
	} else {
		s.broadcastPacket(&packet, true)
	}
}

func (s *Server) handleWeaponReload(p *player.Player, data []byte) {
	if p.StartReload() {
		var packet protocol.PacketWeaponReload
		packet.PacketID = uint8(protocol.PacketTypeWeaponReload)
		packet.PlayerID = p.ID
		packet.MagazineAmmo = p.MagazineAmmo
		packet.ReserveAmmo = p.ReserveAmmo

		s.broadcastPacketExcept(&packet, p.ID, true)
	}
}

func (s *Server) handleChangeTeam(p *player.Player, data []byte) {
	var packet protocol.PacketChangeTeam
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	team, ok := toInternalTeamID(packet.TeamID)
	if !ok {
		s.logger.Warn("received invalid team change request", "player", p.ID, "team", packet.TeamID)
		return
	}

	s.changePlayerTeam(p, team)
}

func (s *Server) handleChangeWeapon(p *player.Player, data []byte) {
	var packet protocol.PacketChangeWeapon
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet); err != nil {
		return
	}

	p.SetWeapon(packet.WeaponID)
	s.respawnPlayer(p.ID)
	s.broadcastShortPlayerData(p)
}

func (s *Server) handleHandshakeReturn(p *player.Player, data []byte) {
	var packet protocol.PacketHandShakeReturn
	if err := packet.Read(data); err != nil {
		s.logger.Warn("invalid handshake return packet", "error", err)
		return
	}

	p.Lock()
	expected := p.HandshakeChallenge
	if expected == 0 || packet.Challenge != expected {
		p.Unlock()
		s.logger.Warn("handshake challenge mismatch", "id", p.ID, "name", p.Name)
		return
	}
	p.HandshakeComplete = true
	p.Unlock()

	s.logger.Debug("handshake challenge verified", "id", p.ID, "name", p.Name)

	if !p.VersionInfoReceived {
		s.sendVersionRequest(p)
	} else {
		s.finalizePlayerJoin(p)
	}
}

func (s *Server) handleVersionResponse(p *player.Player, data []byte) {
	var packet protocol.PacketVersionResponse
	if err := packet.Read(data); err != nil {
		s.logger.Warn("invalid version response packet", "error", err)
		return
	}

	p.Lock()
	p.VersionInfoReceived = true
	p.ClientIdentifier = packet.ClientIdentifier
	p.VersionMajor = packet.VersionMajor
	p.VersionMinor = packet.VersionMinor
	p.VersionRevision = packet.VersionRevision
	p.Version = protocol.Vector3f{
		X: float32(packet.VersionMajor),
		Y: float32(packet.VersionMinor),
		Z: float32(packet.VersionRevision),
	}
	if packet.OSInfo != "" {
		p.OSInfo = packet.OSInfo
	}
	p.Unlock()

	identifier := string([]byte{packet.ClientIdentifier})
	s.logger.Debug("client version reported",
		"id", p.ID,
		"name", p.Name,
		"client", identifier,
		"major", packet.VersionMajor,
		"minor", packet.VersionMinor,
		"revision", packet.VersionRevision,
		"os", packet.OSInfo)

	supportsExtensions := false
	if packet.ClientIdentifier == 'o' {
		if packet.VersionMajor > 0 || (packet.VersionMajor == 0 && packet.VersionMinor > 1) ||
			(packet.VersionMajor == 0 && packet.VersionMinor == 1 && packet.VersionRevision > 3) {
			supportsExtensions = true
		}
	} else if packet.ClientIdentifier == 'B' {
		supportsExtensions = true
	}

	if supportsExtensions {
		s.sendExtensionInfo(p)
	} else {
		s.logger.Debug("client does not support extensions", "player", p.ID, "client", identifier)
	}
}

func (s *Server) handleExtensionInfo(p *player.Player, data []byte) {
	var packet protocol.PacketExtensionInfo
	if err := packet.Read(data); err != nil {
		s.logger.Warn("invalid extension info packet", "error", err)
		return
	}

	for _, entry := range packet.Entries {
		p.AddExtension(entry.ExtensionID, entry.ExtensionVersion)
	}

	s.logger.Info("client extensions registered",
		"player", p.Name,
		"count", len(packet.Entries))
}

func (s *Server) sendExtensionInfo(p *player.Player) {
	packet := protocol.PacketExtensionInfo{
		PacketID: uint8(protocol.PacketTypeExtensionInfo),
		Entries: []protocol.ExtensionEntry{
			{
				ExtensionID:      protocol.ExtensionIDPlayerProperties,
				ExtensionVersion: 1,
			},
			{
				ExtensionID:      protocol.ExtensionID256Players,
				ExtensionVersion: 1,
			},
			{
				ExtensionID:      protocol.ExtensionIDMessageTypes,
				ExtensionVersion: 1,
			},
			{
				ExtensionID:      protocol.ExtensionIDKickReason,
				ExtensionVersion: 1,
			},
		},
	}
	packet.Length = uint8(len(packet.Entries))

	s.sendPacket(p, &packet, true)
	s.logger.Debug("sent extension info", "player", p.ID, "extensions", len(packet.Entries))
}

func (s *Server) sendMapData(p *player.Player) error {
	s.logger.Debug("writing map data", "player", p.ID)
	mapData, err := s.gameState.Map.Write()
	if err != nil {
		return fmt.Errorf("failed to write map: %w", err)
	}
	s.logger.Debug("map data written", "player", p.ID, "size", len(mapData))

	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(mapData); err != nil {
		return fmt.Errorf("failed to compress map data: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("failed to finalize map compression: %w", err)
	}
	s.logger.Debug("map data compressed", "player", p.ID, "compressed_size", compressed.Len())

	data := compressed.Bytes()

	startPacket := protocol.PacketMapStart{
		PacketID: uint8(protocol.PacketTypeMapStart),
		MapSize:  uint32(len(data)),
	}
	s.logger.Info("sending map start", "player", p.ID, "size", len(data))
	s.sendPacket(p, &startPacket, true)

	chunkSize := 8192
	numChunks := (len(data) + chunkSize - 1) / chunkSize
	s.logger.Debug("sending map chunks", "player", p.ID, "num_chunks", numChunks)

	timeout := time.After(60 * time.Second)

	for i := 0; i < len(data); i += chunkSize {
		select {
		case <-timeout:
			return fmt.Errorf("map send timeout for player %d", p.ID)
		default:
		}

		if p.GetState() == player.PlayerStateDisconnected {
			return fmt.Errorf("player %d disconnected during map send", p.ID)
		}

		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunkPacket := protocol.PacketMapChunk{
			PacketID: uint8(protocol.PacketTypeMapChunk),
			Data:     data[i:end],
		}

		s.sendPacket(p, &chunkPacket, true)

		if i%10 == 0 {
			time.Sleep(time.Millisecond)
		}
	}
	s.logger.Debug("finished sending map chunks", "player", p.ID)

	return nil
}

func (s *Server) sendStateDataPacket(p *player.Player) {
	s.logger.Info("preparing state data", "player", p.ID)
	stateData := s.gameState.GetStateData(p.ID)
	s.sendPacket(p, &stateData, true)
	s.logger.Debug("state data sent", "player", p.ID)
}

func (s *Server) sendIntelPositions(p *player.Player) {
	if s.gameState == nil || !s.intelEnabled() {
		return
	}

	for team := uint8(0); team < 2; team++ {
		position, _ := s.gameState.GetIntelState(team)
		packet := protocol.PacketMoveObject{
			PacketID: uint8(protocol.PacketTypeMoveObject),
			ObjectID: team,
			Team:     team,
			X:        position.X,
			Y:        position.Y,
			Z:        position.Z,
		}
		s.sendPacket(p, &packet, true)
	}
}

func (s *Server) sendInitialPackets(p *player.Player) error {
	s.logger.Info("sending map/state", "player", p.ID)
	if err := s.sendMapData(p); err != nil {
		return err
	}
	s.logger.Debug("map data sent successfully", "player", p.ID)

	s.sendStateDataPacket(p)
	s.sendIntelPositions(p)
	s.logger.Debug("calling sendExistingPlayersData", "player", p.ID)
	s.sendExistingPlayersData(p)

	s.sendVersionRequest(p)

	p.Lock()
	p.State = player.PlayerStateWaitingForExistingPlayer
	p.Unlock()
	s.logger.Info("player waiting for ExistingPlayer packet", "player", p.ID)

	return nil
}

func (s *Server) sendMapStart(p *player.Player) {
	s.logger.Info("sending map for reconnect/map change", "player", p.ID)
	if err := s.sendMapData(p); err != nil {
		s.logger.Error("failed to send map data", "error", err)
		return
	}

	s.sendStateDataPacket(p)
	s.sendIntelPositions(p)

	p.Lock()
	p.State = player.PlayerStateReady
	p.Unlock()

	s.finalizePlayerJoin(p)
}

func (s *Server) sendExistingPlayersData(p *player.Player) {
	s.gameState.Players.ForEach(func(other *player.Player) {
		if other.ID == p.ID || other.GetState() != player.PlayerStateReady {
			return
		}

		other.RLock()
		packet := protocol.PacketExistingPlayer{
			PacketID: uint8(protocol.PacketTypeExistingPlayer),
			PlayerID: other.ID,
			Team:     toNetworkTeamID(other.Team),
			Weapon:   other.Weapon,
			Item:     other.Tool,
			Kills:    other.Kills,
			Color:    other.Color,
		}
		copy(packet.Name[:], other.Name)

		other.RUnlock()

		s.sendPacket(p, &packet, true)
	})
}

func (s *Server) finalizePlayerJoin(p *player.Player) {
	p.Lock()
	p.HasIntel = false
	p.Unlock()

	for _, msg := range s.config.Server.WelcomeMessages {
		s.sendChatToPlayer(p, msg)
	}

	s.callbacks.OnPlayerJoin(p)

	if p.GetTeam() == spectatorTeamID {
		s.sendSpectatorConfirmation(p)
		return
	}

	s.broadcastNewPlayer(p)
	s.respawnPlayer(p.ID)
}

func (s *Server) broadcastNewPlayer(p *player.Player) {
	p.RLock()
	packet := protocol.PacketCreatePlayer{
		PacketID: uint8(protocol.PacketTypeCreatePlayer),
		PlayerID: p.ID,
		Weapon:   p.Weapon,
		Team:     toNetworkTeamID(p.Team),
		X:        p.Position.X,
		Y:        p.Position.Y,
		Z:        p.Position.Z,
	}
	copy(packet.Name[:], p.Name)
	p.RUnlock()

	s.broadcastPacketExcept(&packet, p.ID, true)
	s.broadcastShortPlayerData(p)
}

func (s *Server) sendSpectatorConfirmation(p *player.Player) {
	p.RLock()
	packet := protocol.PacketCreatePlayer{
		PacketID: uint8(protocol.PacketTypeCreatePlayer),
		PlayerID: p.ID,
		Weapon:   p.Weapon,
		Team:     spectatorClientTeamID,
		X:        p.Position.X,
		Y:        p.Position.Y,
		Z:        p.Position.Z,
	}
	copy(packet.Name[:], p.Name)
	p.RUnlock()

	s.sendPacket(p, &packet, true)
	s.sendPlayerProperties(p)
}

func (s *Server) respawnPlayer(playerID uint8) {
	p, ok := s.gameState.Players.Get(playerID)
	if !ok {
		return
	}
	if p.GetTeam() > 1 {
		return
	}

	spawnPos := s.gameState.GetSpawnPosition(p.Team)
	p.Respawn(spawnPos)

	s.gameMode.OnPlayerSpawn(p)
	s.callbacks.OnPlayerSpawn(p)

	p.RLock()
	packet := protocol.PacketCreatePlayer{
		PacketID: uint8(protocol.PacketTypeCreatePlayer),
		PlayerID: p.ID,
		Weapon:   p.Weapon,
		Team:     toNetworkTeamID(p.Team),
		X:        spawnPos.X,
		Y:        spawnPos.Y,
		Z:        spawnPos.Z,
	}
	copy(packet.Name[:], p.Name)

	reloadPacket := protocol.PacketWeaponReload{
		PacketID:     uint8(protocol.PacketTypeWeaponReload),
		PlayerID:     p.ID,
		MagazineAmmo: p.MagazineAmmo,
		ReserveAmmo:  p.ReserveAmmo,
	}
	p.RUnlock()

	s.broadcastPacket(&packet, true)
	s.sendPacket(p, &reloadPacket, true)
	s.broadcastShortPlayerData(p)
	s.sendPlayerProperties(p)
}

func (s *Server) damagePlayer(playerID uint8, damage uint8, source protocol.Vector3f, damageType uint8) bool {
	p, ok := s.gameState.Players.Get(playerID)
	if !ok {
		return false
	}

	p.Damage(damage, source, damageType)
	s.callbacks.OnPlayerDamage(p, damage, source)

	killed := false
	p.RLock()
	if p.HP == 0 {
		killed = true
	}
	p.RUnlock()

	hpPacket := protocol.PacketSetHP{
		PacketID: uint8(protocol.PacketTypeSetHP),
		HP:       p.HP,
		Type:     damageType,
		SourceX:  source.X,
		SourceY:  source.Y,
		SourceZ:  source.Z,
	}
	s.sendPacket(p, &hpPacket, true)
	s.sendPlayerProperties(p)

	if killed {
		if p.HasIntel && p.Team <= 1 {
			oppositeTeam := uint8(1 - p.Team)
			pos := p.GetPosition()
			groundPos := s.getGroundIntelDropPosition(pos)
			s.gameState.DropIntel(oppositeTeam, groundPos)
			s.broadcastIntelDrop(oppositeTeam, groundPos)
			p.Lock()
			p.HasIntel = false
			p.Unlock()
			s.logger.Info("intel dropped on death", "player", p.Name, "team", p.Team)
		}

		p.Lock()
		p.State = player.PlayerStateDead
		p.RespawnTime = time.Now().Add(time.Duration(s.config.Server.RespawnTime) * time.Second)
		p.Unlock()
	}

	return killed
}

func (s *Server) broadcastKillAction(victimID, killerID uint8, killType protocol.KillType) {
	packet := protocol.PacketKillAction{
		PacketID:    uint8(protocol.PacketTypeKillAction),
		PlayerID:    victimID,
		KillerID:    killerID,
		KillType:    killType,
		RespawnTime: uint8(s.config.Server.RespawnTime),
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) KillPlayer(victimID uint8, killerID uint8, killType protocol.KillType) {
	victim, exists := s.gameState.Players.Get(victimID)
	if !exists || victim == nil {
		return
	}

	victim.Lock()
	victim.Alive = false
	victim.HP = 0
	victim.State = player.PlayerStateDead
	victim.RespawnTime = time.Now().Add(time.Duration(s.config.Server.RespawnTime) * time.Second)
	victim.Unlock()

	s.callbacks.OnPlayerKill(nil, victim, killType)
	s.broadcastKillAction(victimID, killerID, killType)
}

func (s *Server) handleEnvironmentKill(victim *player.Player, killType protocol.KillType) {
	if victim == nil {
		return
	}

	s.callbacks.OnPlayerKill(nil, victim, killType)
	s.broadcastKillAction(victim.ID, victim.ID, killType)
}

func (s *Server) broadcastPlayerLeft(playerID uint8) {
	packet := protocol.PacketPlayerLeft{
		PacketID: uint8(protocol.PacketTypePlayerLeft),
		PlayerID: playerID,
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) broadcastPlayerLeftExcept(playerID, exceptID uint8) {
	packet := protocol.PacketPlayerLeft{
		PacketID: uint8(protocol.PacketTypePlayerLeft),
		PlayerID: playerID,
	}
	s.broadcastPacketExcept(&packet, exceptID, true)
}

func (s *Server) broadcastChangeTeam(p *player.Player) {
	packet := protocol.PacketChangeTeam{
		PacketID: uint8(protocol.PacketTypeChangeTeam),
		PlayerID: p.ID,
		TeamID:   toNetworkTeamID(p.GetTeam()),
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) changePlayerTeam(p *player.Player, team uint8) {
	if p == nil {
		return
	}

	if team != spectatorTeamID && team > 1 {
		return
	}

	currentTeam := p.GetTeam()
	if currentTeam == team {
		return
	}

	var droppedIntel bool
	var dropPos protocol.Vector3f

	p.Lock()
	if p.HasIntel && currentTeam <= 1 {
		droppedIntel = true
		dropPos = p.Position
		p.HasIntel = false
	}
	p.Team = team
	if team == spectatorTeamID {
		p.Alive = false
	}
	p.State = player.PlayerStateReady
	p.Unlock()

	if droppedIntel {
		opposite := uint8(1 - currentTeam)
		s.gameState.DropIntel(opposite, dropPos)
		s.broadcastIntelDrop(opposite, dropPos)
	}

	if team <= 1 {
		s.broadcastChangeTeam(p)
		s.respawnPlayer(p.ID)
		return
	}

	s.broadcastPlayerLeftExcept(p.ID, p.ID)
	s.broadcastShortPlayerData(p)
	s.sendSpectatorConfirmation(p)
}

func (s *Server) broadcastIntelDrop(team uint8, position protocol.Vector3f) {
	if !s.intelEnabled() {
		return
	}
	packet := protocol.PacketIntelDrop{
		PacketID: uint8(protocol.PacketTypeIntelDrop),
		PlayerID: team,
		Position: position,
	}
	s.broadcastPacket(&packet, true)
	s.broadcastMoveObject(team, position)
}

func (s *Server) BroadcastTerritoryCapture(playerID, entityID, winning, state uint8) {
	packet := protocol.PacketTerritoryCapture{
		PacketID: uint8(protocol.PacketTypeTerritoryCapture),
		PlayerID: playerID,
		EntityID: entityID,
		Winning:  winning,
		State:    state,
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) BroadcastProgressBar(entityID, capturingTeam uint8, rate int8, progress float32) {
	packet := protocol.PacketProgressBar{
		PacketID:      uint8(protocol.PacketTypeProgressBar),
		EntityID:      entityID,
		CapturingTeam: capturingTeam,
		Rate:          rate,
		Progress:      progress,
	}
	s.broadcastPacket(&packet, false)
}

func (s *Server) sendWeaponReload(p *player.Player) {
	p.RLock()
	packet := protocol.PacketWeaponReload{
		PacketID:     uint8(protocol.PacketTypeWeaponReload),
		PlayerID:     p.ID,
		MagazineAmmo: p.MagazineAmmo,
		ReserveAmmo:  p.ReserveAmmo,
	}
	p.RUnlock()
	s.sendPacket(p, &packet, true)
}

func (s *Server) sendWorldUpdate() {
	worldUpdate := protocol.PacketWorldUpdate{
		PacketID: uint8(protocol.PacketTypeWorldUpdate),
	}

	s.gameState.Players.ForEach(func(p *player.Player) {
		if p.GetState() == player.PlayerStateReady && p.GetTeam() <= 1 {
			pos := p.GetPosition()
			ori := p.GetOrientation()

			worldUpdate.Players[p.ID] = protocol.PlayerPositionData{
				X:  pos.X,
				Y:  pos.Y,
				Z:  pos.Z,
				OX: ori.X,
				OY: ori.Y,
				OZ: ori.Z,
			}
		}
	})

	s.gameState.Players.ForEach(func(p *player.Player) {
		if p.GetState() != player.PlayerStateReady {
			return
		}
		s.sendPacket(p, &worldUpdate, false)
	})
}

func (s *Server) updateGrenades(dt float32) {
	currentTime := float64(time.Now().UnixNano()) / 1e9
	var explodedGrenades []*gamestate.Grenade

	s.gameState.UpdateGrenades(func(grenades []*gamestate.Grenade) []int {
		var toRemove []int

		for i, grenade := range grenades {
			elapsed := float32(currentTime - grenade.TimeCreated)
			if elapsed >= grenade.FuseLength {
				explodedGrenades = append(explodedGrenades, grenade)
				toRemove = append(toRemove, i)
				continue
			}

			physics.MoveGrenade(s.gameState.Map, &grenade.Position, &grenade.Velocity, dt)
		}

		return toRemove
	})

	for _, grenade := range explodedGrenades {
		s.explodeGrenade(grenade)
	}
}

func (s *Server) explodeGrenade(grenade *gamestate.Grenade) {
	x, y, z := int(grenade.Position.X), int(grenade.Position.Y), int(grenade.Position.Z)

	if grenade.Position.Z >= 62.0 {
		return
	}

	var destroyedBlocks []protocol.Vector3i

	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				bx, by, bz := x+dx, y+dy, z+dz
				if bz >= 62 {
					continue
				}
				if s.gameState.Map.IsInside(bx, by, bz) && s.gameState.Map.IsSolid(bx, by, bz) {
					destroyedBlocks = append(destroyedBlocks, protocol.Vector3i{
						X: int32(bx),
						Y: int32(by),
						Z: int32(bz),
					})
				}
			}
		}
	}

	s.gameState.Players.ForEach(func(p *player.Player) {
		if !p.IsAlive() {
			return
		}

		pos := p.GetPosition()
		dx := pos.X - grenade.Position.X
		dy := pos.Y - grenade.Position.Y
		dz := pos.Z - grenade.Position.Z

		if abs(int(dx)) >= 16 || abs(int(dy)) >= 16 || abs(int(dz)) >= 16 {
			return
		}

		distanceSquared := float32(dx*dx + dy*dy + dz*dz)
		if distanceSquared < 1e-6 {
			if s.damagePlayer(p.ID, 100, grenade.Position, 1) {
				if killer, ok := s.gameState.Players.Get(grenade.PlayerID); ok {
					s.handlePlayerKill(killer, p, protocol.KillTypeGrenade)
				} else {
					s.handleEnvironmentKill(p, protocol.KillTypeGrenade)
				}
			}
			return
		}

		if !physics.CanSee(s.gameState.Map, grenade.Position, pos) {
			return
		}

		damage := float32(4096.0) / distanceSquared
		if damage > 100.0 {
			damage = 100.0
		}

		if s.damagePlayer(p.ID, uint8(damage), grenade.Position, 1) {
			if killer, ok := s.gameState.Players.Get(grenade.PlayerID); ok {
				s.handlePlayerKill(killer, p, protocol.KillTypeGrenade)
			} else {
				s.handleEnvironmentKill(p, protocol.KillTypeGrenade)
			}
		}
	})

	for _, block := range destroyedBlocks {
		bx := int(block.X)
		by := int(block.Y)
		bz := int(block.Z)

		s.gameState.Map.SetAir(bx, by, bz)

		blockPacket := protocol.PacketBlockAction{
			PacketID: uint8(protocol.PacketTypeBlockAction),
			PlayerID: grenade.PlayerID,
			Action:   protocol.BlockActionTypeGrenadeDestroy,
			X:        block.X,
			Y:        block.Y,
			Z:        block.Z,
		}
		s.broadcastPacket(&blockPacket, true)
	}
}

func marshalPacket(packet interface{}) ([]byte, error) {
	var buf bytes.Buffer

	if writer, ok := packet.(interface{ Write(io.Writer) error }); ok {
		if err := writer.Write(&buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	if err := binary.Write(&buf, binary.LittleEndian, packet); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *Server) sendPacket(p *player.Player, packet interface{}, reliable bool) {
	data, err := marshalPacket(packet)
	if err != nil {
		s.logger.Error("failed to encode packet", "error", err)
		return
	}

	if len(data) > 0 {
		level := slog.LevelDebug
		if data[0] == uint8(protocol.PacketTypeHandShakeInit) || data[0] == uint8(protocol.PacketTypeMapStart) || data[0] == uint8(protocol.PacketTypeStateData) {
			level = slog.LevelInfo
		}
		s.logger.LogAttrs(context.Background(), level, "sending packet",
			slog.Int("player", int(p.ID)),
			slog.Int("type", int(data[0])),
			slog.Int("len", len(data)),
			slog.Bool("reliable", reliable))
	}

	if err := s.network.SendPacket(p.Peer, data, reliable); err != nil {
		s.logger.Error("failed to send packet", "error", err)
	}
}

func (s *Server) broadcastPacket(packet interface{}, reliable bool) {
	data, err := marshalPacket(packet)
	if err != nil {
		s.logger.Error("failed to encode packet", "error", err)
		return
	}

	if err := s.network.Broadcast(data, reliable); err != nil {
		s.logger.Error("failed to broadcast packet", "error", err)
	}
}

func (s *Server) broadcastPacketExcept(packet interface{}, exceptID uint8, reliable bool) {
	data, err := marshalPacket(packet)
	if err != nil {
		s.logger.Error("failed to encode packet", "error", err)
		return
	}

	s.gameState.Players.ForEach(func(p *player.Player) {
		if p.ID != exceptID && p.GetState() == player.PlayerStateReady {
			p.RLock()
			peer := p.Peer
			p.RUnlock()
			if err := s.network.SendPacket(peer, data, reliable); err != nil {
				s.logger.Error("failed to send packet", "player", p.ID, "error", err)
			}
		}
	})
}

func (s *Server) intelEnabled() bool {
	gm, _ := config.ParseGamemode(s.config.Server.Gamemode)
	if gm == config.GamemodeTDM && s.config.Server.RemoveIntel {
		return false
	}
	return true
}

func (s *Server) envHazardsEnabled() bool {
	gm, _ := config.ParseGamemode(s.config.Server.Gamemode)
	return !(gm == config.GamemodeTDM && s.config.Server.RemoveIntel)
}

func (s *Server) sendVersionRequest(p *player.Player) {
	packet := protocol.PacketVersionRequest{
		PacketID: uint8(protocol.PacketTypeVersionRequest),
	}
	s.sendPacket(p, &packet, true)
}

func (s *Server) broadcastShortPlayerData(p *player.Player) {
	if p == nil {
		return
	}
	packet := protocol.PacketShortPlayerData{
		PacketID: uint8(protocol.PacketTypeShortPlayerData),
		PlayerID: p.ID,
		Team:     p.Team,
		Weapon:   p.Weapon,
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) broadcastMoveObject(team uint8, position protocol.Vector3f) {
	if !s.running || !s.intelEnabled() {
		return
	}
	packet := protocol.PacketMoveObject{
		PacketID: uint8(protocol.PacketTypeMoveObject),
		ObjectID: team,
		Team:     team,
		X:        position.X,
		Y:        position.Y,
		Z:        position.Z,
	}
	s.broadcastPacket(&packet, true)
}

// sends a position/orientation packet for a player without updating server state
// if player id is 255, broadcasts to all players
func (s *Server) SendPlayerPositionPacketTo(playerID uint8, pos, ori protocol.Vector3f, toPlayerID uint8) {
	if !s.running {
		return
	}

	packet := protocol.PacketPositionData{
		X: pos.X,
		Y: pos.Y,
		Z: pos.Z,
	}

	data, err := marshalPacket(&packet)
	if err != nil {
		s.logger.Error("failed to encode position packet", "error", err)
		return
	}

	if toPlayerID == 255 {
		// broadcast to all players
		if err := s.network.Broadcast(data, false); err != nil {
			s.logger.Error("failed to broadcast position packet", "error", err)
		}
	} else {
		// send to specific player
		targetPlayer, ok := s.gameState.Players.Get(toPlayerID)
		if ok && targetPlayer.GetState() == player.PlayerStateReady {
			if err := s.network.SendPacket(targetPlayer.Peer, data, false); err != nil {
				s.logger.Error("failed to send position packet", "player", toPlayerID, "error", err)
			}
		}
	}
}

func (s *Server) SendIntelPositionPacketOnly(objectID uint8, team uint8, position protocol.Vector3f) {
	if !s.running {
		return
	}

	packet := protocol.PacketMoveObject{
		PacketID: uint8(protocol.PacketTypeMoveObject),
		ObjectID: objectID,
		Team:     team,
		X:        position.X,
		Y:        position.Y,
		Z:        position.Z,
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) syncIntelPositions() {
	if !s.running || !s.intelEnabled() {
		return
	}
	for team := uint8(0); team < 2; team++ {
		position, _ := s.gameState.GetIntelState(team)
		s.broadcastMoveObject(team, position)
	}
}

func (s *Server) GetMapBlock(x, y, z int) uint32 {
	if s.gameState == nil || s.gameState.Map == nil {
		return 0
	}
	return s.gameState.Map.Get(x, y, z)
}

func (s *Server) SetMapBlock(x, y, z int, color uint32) {
	if s.gameState == nil || s.gameState.Map == nil {
		return
	}
	s.gameState.Map.Set(x, y, z, color)

	packet := protocol.PacketBlockAction{
		PacketID: uint8(protocol.PacketTypeBlockAction),
		PlayerID: 32,
		Action:   protocol.BlockActionTypeBuild,
		X:        int32(x),
		Y:        int32(y),
		Z:        int32(z),
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) IsMapSolid(x, y, z int) bool {
	if s.gameState == nil || s.gameState.Map == nil {
		return false
	}
	return s.gameState.Map.IsSolid(x, y, z)
}

func (s *Server) FindTopBlock(x, y int) int {
	if s.gameState == nil || s.gameState.Map == nil {
		return 0
	}
	return s.gameState.Map.FindTopBlock(x, y)
}

func (s *Server) checkWaterDamage(p *player.Player) {
	if !s.envHazardsEnabled() {
		return
	}
	var waterDamage int
	var waterLevel float32 = 63.0

	if s.gameState.MapConfig.Extensions.WaterDamage != nil {
		waterDamage = *s.gameState.MapConfig.Extensions.WaterDamage
	} else if s.gameState.MapConfig.Water.Enabled {
		waterDamage = s.gameState.MapConfig.Water.Damage
		waterLevel = s.gameState.MapConfig.Water.Level
	} else {
		return
	}

	if waterDamage <= 0 {
		return
	}

	pos := p.GetPosition()
	if pos.Z >= waterLevel {
		now := time.Now()
		if now.Sub(p.LastWaterDamage) >= time.Second {
			if s.damagePlayer(p.ID, uint8(waterDamage), pos, 0) {
				s.handleEnvironmentKill(p, protocol.KillTypeFall)
			}
			p.LastWaterDamage = now
		}
	}
}

func (s *Server) checkBoundaryDamage(p *player.Player) {
	if !s.envHazardsEnabled() || s.gameState.MapConfig.Extensions.BoundaryDamage == nil {
		return
	}

	boundary := s.gameState.MapConfig.Extensions.BoundaryDamage
	pos := p.GetPosition()

	outOfBounds := false
	if int(pos.X) <= boundary.Left || int(pos.X) >= boundary.Right {
		outOfBounds = true
	}
	if int(pos.Y) <= boundary.Top || int(pos.Y) >= boundary.Bottom {
		outOfBounds = true
	}

	if outOfBounds {
		now := time.Now()
		if now.Sub(p.LastBoundaryDamage) >= time.Second {
			if s.damagePlayer(p.ID, uint8(boundary.Damage), pos, 0) {
				s.handleEnvironmentKill(p, protocol.KillTypeFall)
			}
			p.LastBoundaryDamage = now
		}
	}
}

func (s *Server) checkIntelPickup(p *player.Player) {
	if !s.intelEnabled() {
		return
	}
	if p.HasIntel {
		return
	}

	if p.Team > 1 {
		return
	}

	gm, _ := config.ParseGamemode(s.config.Server.Gamemode)

	var intelTeam uint8
	if gm == config.GamemodeBabel {
		// in babel, check the center flag (team 0)
		intelTeam = 0
	} else {
		// in ctf/tdm check opposite team's flag
		intelTeam = uint8(1 - p.Team)
	}

	intelPos, held := s.gameState.GetIntelState(intelTeam)

	if held {
		return
	}

	pos := p.GetPosition()
	dx := pos.X - intelPos.X
	dy := pos.Y - intelPos.Y
	dz := pos.Z - intelPos.Z
	distSquared := dx*dx + dy*dy + dz*dz

	if distSquared <= 1.5*1.5 {
		if s.gameMode.OnIntelPickup(p, intelTeam) {
			if s.gameState.PickupIntel(p.ID, p.Team) {
				packet := protocol.PacketIntelPickup{
					PacketID: uint8(protocol.PacketTypeIntelPickup),
					PlayerID: p.ID,
				}
				s.broadcastPacket(&packet, true)
				s.logger.Info("intel picked up", "player", p.Name, "team", p.Team)
			}
		}
	}
}

func (s *Server) isNearBase(pos protocol.Vector3f, team uint8) bool {
	base := s.gameState.GetBase(team)
	dx := pos.X - base.X
	dy := pos.Y - base.Y
	dz := pos.Z - base.Z
	distSquared := dx*dx + dy*dy + dz*dz
	return distSquared <= 3.0*3.0
}

func (s *Server) getGroundIntelDropPosition(pos protocol.Vector3f) protocol.Vector3f {
	x := int(pos.X)
	y := int(pos.Y)

	if x < 0 || y < 0 || x >= s.gameState.Map.Width() || y >= s.gameState.Map.Height() {
		x = max(0, min(x, s.gameState.Map.Width()-1))
		y = max(0, min(y, s.gameState.Map.Height()-1))
	}

	groundZ := s.gameState.Map.FindGroundLevel(x, y)

	return protocol.Vector3f{
		X: float32(x) + 0.5,
		Y: float32(y) + 0.5,
		Z: float32(groundZ),
	}
}

func (s *Server) checkBabelIntelCapture(p *player.Player, pos protocol.Vector3f) bool {
	if s.isNearBase(pos, 0) {
		if s.gameMode.OnIntelCapture(p, 0) {
			if s.gameState.CaptureIntel(p.ID, p.Team) {
				s.handleCaptureSuccess(p)
				return true
			}
		}
	}

	if s.isNearBase(pos, 1) {
		if s.gameMode.OnIntelCapture(p, 1) {
			if s.gameState.CaptureIntel(p.ID, p.Team) {
				s.handleCaptureSuccess(p)
				return true
			}
		}
	}

	return false
}

func (s *Server) checkCTFIntelCapture(p *player.Player, pos protocol.Vector3f) {
	if !s.isNearBase(pos, p.Team) {
		return
	}

	if !s.gameState.IsIntelAtBase(p.Team) {
		return
	}

	if s.gameMode.OnIntelCapture(p, p.Team) {
		if s.gameState.CaptureIntel(p.ID, p.Team) {
			s.handleCaptureSuccess(p)
		}
	}
}

func (s *Server) checkIntelCapture(p *player.Player) {
	if !s.intelEnabled() {
		return
	}
	if !p.HasIntel {
		return
	}

	if p.Team > 1 {
		return
	}

	gm, _ := config.ParseGamemode(s.config.Server.Gamemode)
	pos := p.GetPosition()

	if gm == config.GamemodeBabel {
		s.checkBabelIntelCapture(p, pos)
	} else {
		s.checkCTFIntelCapture(p, pos)
	}
}

func (s *Server) checkRestock(p *player.Player) {
	const restockCooldown = 15 * time.Second
	const restockRadius = 3.0

	if p.Team > 1 {
		return
	}

	if time.Since(p.LastRestockTime) < restockCooldown {
		return
	}

	pos := p.GetPosition()
	base := s.gameState.Base[p.Team]

	dx := math.Abs(float64(pos.X - base.X))
	dy := math.Abs(float64(pos.Y - base.Y))
	dz := math.Abs(float64(pos.Z - base.Z))

	if dx < restockRadius && dy < restockRadius && dz < restockRadius {
		if p.NeedsRestock() {
			p.Restock()
			p.LastRestockTime = time.Now()
			s.callbacks.OnRestock(p)
			s.sendRestock(p)
			s.sendPlayerProperties(p)
		}
	}
}

func (s *Server) sendRestock(p *player.Player) {
	packet := protocol.PacketRestock{
		PacketID: uint8(protocol.PacketTypeRestock),
		PlayerID: p.ID,
	}
	s.broadcastPacket(&packet, true)
}

func (s *Server) sendPlayerProperties(p *player.Player) {
	if !p.SupportsExtension(protocol.ExtensionIDPlayerProperties) {
		return
	}

	p.RLock()
	packet := protocol.PacketPlayerProperties{
		PacketID:     uint8(protocol.PacketTypePlayerProperties),
		SubPacketID:  0,
		PlayerID:     p.ID,
		HP:           p.HP,
		Blocks:       p.Blocks,
		Grenades:     p.Grenades,
		MagazineAmmo: p.MagazineAmmo,
		ReserveAmmo:  p.ReserveAmmo,
		Score:        p.Kills,
	}
	p.RUnlock()

	s.sendPacket(p, &packet, true)
}

func (s *Server) handleCaptureSuccess(p *player.Player) {
	won, winningTeam := s.gameMode.CheckWinCondition()
	winning := uint8(0)
	if won {
		winning = 1
	}

	packet := protocol.PacketIntelCapture{
		PacketID: uint8(protocol.PacketTypeIntelCapture),
		PlayerID: p.ID,
		Winning:  winning,
	}
	s.broadcastPacket(&packet, true)

	s.syncIntelPositions()

	gm, _ := config.ParseGamemode(s.config.Server.Gamemode)
	if gm == config.GamemodeBabel || gm == config.GamemodeTC {
		tc := protocol.PacketTerritoryCapture{
			PacketID: uint8(protocol.PacketTypeTerritoryCapture),
			PlayerID: p.ID,
			EntityID: 0,
			Winning:  p.Team,
			State:    p.Team,
		}
		s.broadcastPacket(&tc, true)

		progress := protocol.PacketProgressBar{
			PacketID:      uint8(protocol.PacketTypeProgressBar),
			EntityID:      0,
			CapturingTeam: p.Team,
			Rate:          0,
			Progress:      1.0,
		}
		s.broadcastPacket(&progress, true)
	}

	score := s.gameState.GetTeamScore(p.Team)
	s.logger.Info("intel captured", "player", p.Name, "team", p.Team, "score", score, "winning", winning)

	if won {
		s.gameState.ResetScores()
		s.broadcastChat(fmt.Sprintf("%s team wins!", s.getTeamName(winningTeam)), protocol.ChatTypeSystem)

		if s.gameMode.ShouldRotateMap() {
			time.Sleep(5 * time.Second)
			s.rotateMap()
		}
	}
}

func (s *Server) handleTimeLimitReached() {
	team1Score := s.gameState.GetTeamScore(0)
	team2Score := s.gameState.GetTeamScore(1)

	var message string

	if team1Score > team2Score {
		message = fmt.Sprintf("Time limit reached! %s team wins!", s.getTeamName(0))
	} else if team2Score > team1Score {
		message = fmt.Sprintf("Time limit reached! %s team wins!", s.getTeamName(1))
	} else {
		message = "Time limit reached! It's a draw!"
	}

	s.logger.Info("time limit reached", "team1", team1Score, "team2", team2Score)
	s.broadcastChat(message, protocol.ChatTypeSystem)

	s.gameState.ResetScores()
	s.gameState.ResetIntel()

	if s.gameMode.ShouldRotateMap() {
		time.Sleep(5 * time.Second)
		s.rotateMap()
	}
}

func (s *Server) getTeamName(team uint8) string {
	if team == 0 {
		return s.config.Teams.Team1.Name
	}
	return s.config.Teams.Team2.Name
}

func (s *Server) changeMap(mapName string) error {
	s.logger.Info("changing map", "spec", mapName)

	if err := s.loadMap(mapName); err != nil {
		return fmt.Errorf("failed to load map: %w", err)
	}

	displayName := s.GetCurrentMapName()
	reportName := s.getReportedMapName()

	s.gameState.Players.ForEach(func(p *player.Player) {
		if p.GetState() == player.PlayerStateReady {
			p.Lock()
			p.State = player.PlayerStateLoading
			p.HasIntel = false
			p.Unlock()

			s.sendMapStart(p)
		}
	})

	for _, ms := range s.masterServers {
		ms.UpdateMap(reportName)
	}

	s.updatePingServerInfo()

	s.logger.Info("map changed", "spec", mapName, "display", displayName)

	if s.running {
		s.syncIntelPositions()
	}

	return nil
}

func (s *Server) rotateMap() {
	s.currentMap = (s.currentMap + 1) % len(s.config.Server.Maps)
	mapName := s.config.Server.Maps[s.currentMap]

	if err := s.changeMap(mapName); err != nil {
		s.logger.Error("failed to rotate map", "error", err)
	} else {
		s.broadcastChat(fmt.Sprintf("Map changed to %s", s.GetCurrentMapName()), protocol.ChatTypeSystem)
	}
}

func (s *Server) startPeriodicAnnouncements() {
	if len(s.config.Server.PeriodicMessages) == 0 {
		return
	}

	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	messageIndex := 0

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("periodic announcements stopped")
			return

		case <-ticker.C:
			if !s.running {
				return
			}

			message := s.config.Server.PeriodicMessages[messageIndex]
			s.broadcastChat(message, protocol.ChatTypeSystem)

			messageIndex = (messageIndex + 1) % len(s.config.Server.PeriodicMessages)
		}
	}
}

func (s *Server) StartVotekick(instigator *player.Player, victimID uint8, reason string) error {
	victim, ok := s.gameState.Players.Get(victimID)
	if !ok {
		return fmt.Errorf("player not found")
	}

	config := vote.VotekickConfig{
		Percentage:  35,
		BanDuration: 30 * time.Minute,
		PublicVotes: true,
		OnSuccess: func(p *player.Player, reason string, duration time.Duration) {
			ip := p.Peer.GetAddress().String()
			err := s.banManager.AddBan(ip, p.Name, reason, instigator.Name, duration)
			if err != nil {
				s.logger.Error("failed to ban player", "error", err)
			}

			time.AfterFunc(100*time.Millisecond, func() {
				s.network.DisconnectPeer(p.Peer, false)
			})
		},
		OnCancel: func(msg string) {
			s.broadcastChat(msg, protocol.ChatTypeSystem)
		},
		OnTimeout: func() {
			s.broadcastChat("Votekick timed out", protocol.ChatTypeSystem)
		},
		OnUpdate: func(msg string) {
			s.broadcastChat(msg, protocol.ChatTypeSystem)
		},
		GetPlayerCount: func() int {
			count := 0
			s.gameState.Players.ForEach(func(p *player.Player) {
				if p.GetState() == player.PlayerStateReady {
					count++
				}
			})
			return count
		},
	}

	votekick := vote.NewVotekick(instigator, victim, reason, config)
	return s.voteManager.StartVote(votekick)
}

func (s *Server) StartVotemap(instigator *player.Player) error {
	if !s.config.Voting.VotemapEnabled {
		return fmt.Errorf("votemap is disabled on this server")
	}

	config := vote.VotemapConfig{
		Percentage:  max(1, s.config.Voting.VotemapPercentage),
		AllowExtend: s.config.Voting.VotemapAllowExtend,
		OnSuccess: func(mapName string) {
			if mapName == "extend" {
				s.broadcastChat("Map extended by 15 minutes", protocol.ChatTypeSystem)
			} else {
				time.AfterFunc(5*time.Second, func() {
					err := s.changeMap(mapName)
					if err != nil {
						s.logger.Error("failed to change map", "error", err)
						s.broadcastChat("Failed to change map", protocol.ChatTypeSystem)
					}
				})
			}
		},
		OnCancel: func(msg string) {
			s.broadcastChat(msg, protocol.ChatTypeSystem)
		},
		OnTimeout: func() {
			s.broadcastChat("Map vote timed out", protocol.ChatTypeSystem)
		},
		OnUpdate: func(msg string) {
			s.broadcastChat(msg, protocol.ChatTypeSystem)
		},
		GetPlayerCount: func() int {
			count := 0
			s.gameState.Players.ForEach(func(p *player.Player) {
				if p.GetState() == player.PlayerStateReady {
					count++
				}
			})
			return count
		},
		GetMapRotation: func() []string {
			return s.config.Server.Maps
		},
		GetCurrentMap: func() string {
			return s.GetCurrentMapName()
		},
	}

	votemap := vote.NewVotemap(instigator, config)
	return s.voteManager.StartVote(votemap)
}

func (s *Server) CastVote(p *player.Player, choice interface{}) error {
	return s.voteManager.CastVote(p, choice)
}

func (s *Server) CancelVote(p *player.Player) error {
	return s.voteManager.CancelVote(p)
}

func (s *Server) GetActiveVote() vote.Vote {
	return s.voteManager.GetActiveVote()
}

func (s *Server) HasActiveVote() bool {
	return s.voteManager.HasActiveVote()
}

func (s *Server) updatePingServerInfo() {
	if s.pingHandler == nil {
		return
	}

	playerCount := 0
	s.gameState.Players.ForEach(func(p *player.Player) {
		if p.GetTeam() < 2 {
			playerCount++
		}
	})

	gm, _ := config.ParseGamemode(s.config.Server.Gamemode)

	s.pingHandler.UpdateServerInfo(&ping.ServerInfo{
		Name:           s.config.Server.Name,
		PlayersCurrent: playerCount,
		PlayersMax:     s.config.Server.MaxPlayers,
		Map:            s.getReportedMapName(),
		GameMode:       gm.String(),
		GameVersion:    "0.75",
	})
}
