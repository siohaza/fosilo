package lua

import (
	"fmt"
	"math"
	"time"

	"github.com/siohaza/fosilo/internal/bans"
	"github.com/siohaza/fosilo/internal/gamestate"
	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
	"github.com/siohaza/fosilo/internal/vote"

	"github.com/Shopify/go-lua"
)

type ServerInterface interface {
	KickPlayer(playerID uint8, reason string)
	KillPlayer(victimID uint8, killerID uint8, killType protocol.KillType)
	SendChatToAll(message string)
	SendChatToPlayer(p *player.Player, message string)
	SendChatWithType(message string, chatType protocol.ChatType)
	StartVotekick(instigator *player.Player, victimID uint8, reason string) error
	StartVotemap(instigator *player.Player) error
	CastVote(p *player.Player, choice interface{}) error
	CancelVote(p *player.Player) error
	HasActiveVote() bool
	GetActiveVote() vote.Vote
	ReloadCommands() error
	ReloadGamemode() error
	GetConfigPassword(role string) string
	GetCurrentMapName() string
	GetServerName() string
	GetUptime() time.Duration
	RespawnPlayer(playerID uint8)
	SetPlayerTeam(playerID uint8, team uint8)
	RestockPlayer(playerID uint8)
	BroadcastShortPlayerData(p *player.Player)
	DisconnectPlayerWithReason(p *player.Player, reason uint32)
	SendPlayerLeftPacket(playerID uint8)
	SaveMap(filename string) (string, error)
	BroadcastTerritoryCapture(playerID, entityID, winning, state uint8)
	BroadcastProgressBar(entityID, capturingTeam uint8, rate int8, progress float32)
}

type GameAPI struct {
	gameState      *gamestate.GameState
	banManager     *bans.Manager
	server         ServerInterface
	commandManager *CommandManager
	gamemodeVM     *VM
}

func NewGameAPI(gs *gamestate.GameState) *GameAPI {
	return &GameAPI{
		gameState: gs,
	}
}

func (api *GameAPI) SetBanManager(bm *bans.Manager) {
	api.banManager = bm
}

func (api *GameAPI) SetServer(srv ServerInterface) {
	api.server = srv
}

func (api *GameAPI) SetCommandManager(cm *CommandManager) {
	api.commandManager = cm
}

func (api *GameAPI) SetGamemodeVM(vm *VM) {
	api.gamemodeVM = vm
}

func (api *GameAPI) RegisterFunctions(vm *VM) {
	state := vm.State()

	state.Register("find_top_block", api.findTopBlock)
	state.Register("get_block", api.getBlock)
	state.Register("is_solid", api.isSolid)
	state.Register("set_block", api.setBlock)
	state.Register("destroy_block", api.destroyBlock)
	state.Register("get_player", api.getPlayer)
	state.Register("get_player_by_id", api.getPlayerByID)
	state.Register("get_player_count", api.getPlayerCount)
	state.Register("get_team_score", api.getTeamScore)
	state.Register("set_team_score", api.setTeamScore)
	state.Register("set_intel_position", api.setIntelPosition)
	state.Register("set_base_position", api.setBasePosition)
	state.Register("get_intel_position", api.getIntelPosition)
	state.Register("get_base_position", api.getBasePosition)
	state.Register("send_chat", api.sendChat)
	state.Register("kill_player", api.killPlayer)
	state.Register("set_player_position", api.setPlayerPosition)
	state.Register("get_player_position", api.getPlayerPosition)
	state.Register("get_player_team", api.getPlayerTeam)
	state.Register("get_player_name", api.getPlayerName)
	state.Register("is_player_alive", api.isPlayerAlive)
	state.Register("get_map_width", api.getMapWidth)
	state.Register("get_map_height", api.getMapHeight)
	state.Register("get_map_depth", api.getMapDepth)
	state.Register("ban_player", api.banPlayer)
	state.Register("unban_ip", api.unbanIP)
	state.Register("is_banned", api.isBanned)
	state.Register("kick_player_cmd", api.kickPlayerCmd)
	state.Register("disconnect_player", api.disconnectPlayer)
	state.Register("broadcast_chat", api.broadcastChat)
	state.Register("get_player_ip", api.getPlayerIP)
	state.Register("start_votekick", api.startVotekick)
	state.Register("start_votemap", api.startVotemap)
	state.Register("cast_vote", api.castVote)
	state.Register("cancel_vote", api.cancelVote)
	state.Register("has_active_vote", api.hasActiveVote)
	state.Register("get_vote_choices", api.getVoteChoices)
	state.Register("get_vote_type", api.getVoteType)
	state.Register("get_player_by_name", api.getPlayerByName)
	state.Register("reload_commands", api.reloadCommands)
	state.Register("reload_gamemode", api.reloadGamemode)
	state.Register("send_big_message", api.sendBigMessage)
	state.Register("send_info_message", api.sendInfoMessage)
	state.Register("send_warning_message", api.sendWarningMessage)
	state.Register("send_error_message", api.sendErrorMessage)
	state.Register("get_available_commands", api.getAvailableCommands)
	state.Register("has_permission", api.hasPermission)
	state.Register("set_player_permission", api.setPlayerPermission)
	state.Register("get_config_password", api.getConfigPassword)
	state.Register("get_map_name", api.getMapName)
	state.Register("save_map", api.saveMap)

	state.Register("set_player_hp", api.setPlayerHP)
	state.Register("set_player_team", api.setPlayerTeam)
	state.Register("heal_player", api.healPlayer)
	state.Register("set_player_weapon", api.setPlayerWeapon)
	state.Register("set_player_ammo", api.setPlayerAmmo)
	state.Register("set_player_grenades", api.setPlayerGrenades)
	state.Register("set_player_blocks", api.setPlayerBlocks)
	state.Register("respawn_player", api.respawnPlayer)
	state.Register("get_player_weapon", api.getPlayerWeapon)

	state.Register("schedule_callback", api.scheduleCallback)
	state.Register("cancel_callback", api.cancelCallback)
	state.Register("get_server_name", api.getServerName)
	state.Register("get_server_time", api.getServerTime)

	state.Register("get_spawn_location", api.getSpawnLocation)
	state.Register("is_valid_position", api.isValidPosition)

	state.Register("get_player_ping", api.getPlayerPing)
	state.Register("get_player_ammo", api.getPlayerAmmo)
	state.Register("get_player_grenades", api.getPlayerGrenades)
	state.Register("get_player_blocks", api.getPlayerBlocks)
	state.Register("get_player_color", api.getPlayerColor)
	state.Register("get_player_tool", api.getPlayerTool)
	state.Register("get_player_state", api.getPlayerState)
	state.Register("set_player_orientation", api.setPlayerOrientation)
	state.Register("get_player_orientation", api.getPlayerOrientation)

	state.Register("distance_3d", api.distance3D)
	state.Register("distance_2d", api.distance2D)
	state.Register("rgb_to_color", api.rgbToColor)
	state.Register("color_to_rgb", api.colorToRgb)
	state.Register("for_each_player", api.forEachPlayer)
	state.Register("clamp", api.clamp)
	state.Register("lerp", api.lerp)

	state.Register("get_timer_info", api.getTimerInfo)
	state.Register("pause_timer", api.pauseTimer)
	state.Register("resume_timer", api.resumeTimer)
	state.Register("create_explosion", api.createExplosion)

	state.Register("send_territory_capture", api.sendTerritoryCapture)
	state.Register("send_progress_bar", api.sendProgressBar)
	state.Register("get_config_value", api.getConfigValue)
}

func (api *GameAPI) findTopBlock(state *lua.State) int {
	x, _ := state.ToInteger(1)
	y, _ := state.ToInteger(2)

	if api.gameState.Map == nil {
		state.PushInteger(-1)
		return 1
	}

	z := api.gameState.Map.FindTopBlock(x, y)
	state.PushInteger(z)
	return 1
}

func (api *GameAPI) getBlock(state *lua.State) int {
	x, _ := state.ToInteger(1)
	y, _ := state.ToInteger(2)
	z, _ := state.ToInteger(3)

	if api.gameState.Map == nil {
		state.PushInteger(0)
		return 1
	}

	color := api.gameState.Map.Get(x, y, z)
	state.PushInteger(int(color))
	return 1
}

func (api *GameAPI) isSolid(state *lua.State) int {
	x, _ := state.ToInteger(1)
	y, _ := state.ToInteger(2)
	z, _ := state.ToInteger(3)

	if api.gameState.Map == nil {
		state.PushBoolean(false)
		return 1
	}

	solid := api.gameState.Map.IsSolid(x, y, z)
	state.PushBoolean(solid)
	return 1
}

func (api *GameAPI) setBlock(state *lua.State) int {
	x, _ := state.ToInteger(1)
	y, _ := state.ToInteger(2)
	z, _ := state.ToInteger(3)
	color, _ := state.ToInteger(4)

	if api.gameState.Map != nil {
		api.gameState.Map.Set(x, y, z, uint32(color))
	}
	return 0
}

func (api *GameAPI) destroyBlock(state *lua.State) int {
	x, _ := state.ToInteger(1)
	y, _ := state.ToInteger(2)
	z, _ := state.ToInteger(3)

	if api.gameState.Map != nil {
		api.gameState.Map.SetAir(x, y, z)
	}
	return 0
}

func (api *GameAPI) getPlayerCount(state *lua.State) int {
	count := 0
	api.gameState.Players.ForEach(func(p *player.Player) {
		count++
	})
	state.PushInteger(count)
	return 1
}

func (api *GameAPI) getPlayer(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	pushPlayerTable(state, p)
	return 1
}

func (api *GameAPI) getPlayerByID(state *lua.State) int {
	return api.getPlayer(state)
}

func pushPlayerTable(state *lua.State, p *player.Player) {
	if p == nil {
		state.PushNil()
		return
	}

	p.RLock()
	defer p.RUnlock()

	state.NewTable()
	state.PushInteger(int(p.ID))
	state.SetField(-2, "id")
	state.PushString(p.Name)
	state.SetField(-2, "name")
	state.PushInteger(int(p.Team))
	state.SetField(-2, "team")
	state.PushBoolean(p.Alive)
	state.SetField(-2, "alive")
	state.PushInteger(int(p.HP))
	state.SetField(-2, "hp")
	state.PushInteger(int(p.Kills))
	state.SetField(-2, "kills")
	state.PushInteger(int(p.Deaths))
	state.SetField(-2, "deaths")
	state.PushBoolean(p.HasIntel)
	state.SetField(-2, "has_intel")
	state.PushInteger(int(p.Permissions))
	state.SetField(-2, "permissions")

	state.PushString(string(p.ClientIdentifier))
	state.SetField(-2, "client_identifier")
	state.PushInteger(int(p.VersionMajor))
	state.SetField(-2, "version_major")
	state.PushInteger(int(p.VersionMinor))
	state.SetField(-2, "version_minor")
	state.PushInteger(int(p.VersionRevision))
	state.SetField(-2, "version_revision")
	state.PushString(p.OSInfo)
	state.SetField(-2, "os_info")

	state.NewTable()
	state.PushNumber(float64(p.Position.X))
	state.RawSetInt(-2, 1)
	state.PushNumber(float64(p.Position.Y))
	state.RawSetInt(-2, 2)
	state.PushNumber(float64(p.Position.Z))
	state.RawSetInt(-2, 3)
	state.SetField(-2, "position")
}

func (api *GameAPI) getTeamScore(state *lua.State) int {
	team, _ := state.ToInteger(1)

	var score uint8
	if team == 0 {
		score = api.gameState.Team1Score
	} else if team == 1 {
		score = api.gameState.Team2Score
	}

	state.PushInteger(int(score))
	return 1
}

func (api *GameAPI) setTeamScore(state *lua.State) int {
	team, _ := state.ToInteger(1)
	score, _ := state.ToInteger(2)

	if team == 0 {
		api.gameState.Team1Score = uint8(score)
	} else if team == 1 {
		api.gameState.Team2Score = uint8(score)
	}

	return 0
}

func (api *GameAPI) setIntelPosition(state *lua.State) int {
	xNum, _ := state.ToNumber(1)
	x := float32(xNum)
	yNum, _ := state.ToNumber(2)
	y := float32(yNum)
	zNum, _ := state.ToNumber(3)
	z := float32(zNum)
	team, _ := state.ToInteger(4)

	if team == 0 || team == 1 {
		api.gameState.Intel[team].Position = protocol.Vector3f{X: x, Y: y, Z: z}
		api.gameState.IntelSpawnPos[team] = protocol.Vector3f{X: x, Y: y, Z: z}
	}

	return 0
}

func (api *GameAPI) setBasePosition(state *lua.State) int {
	xNum, _ := state.ToNumber(1)
	x := float32(xNum)
	yNum, _ := state.ToNumber(2)
	y := float32(yNum)
	zNum, _ := state.ToNumber(3)
	z := float32(zNum)
	team, _ := state.ToInteger(4)

	if team == 0 || team == 1 {
		api.gameState.Base[team] = protocol.Vector3f{X: x, Y: y, Z: z}
	}

	return 0
}

func (api *GameAPI) getIntelPosition(state *lua.State) int {
	team, _ := state.ToInteger(1)

	var pos protocol.Vector3f
	if team == 0 || team == 1 {
		pos = api.gameState.Intel[team].Position
	}

	state.PushNumber(float64(pos.X))
	state.PushNumber(float64(pos.Y))
	state.PushNumber(float64(pos.Z))
	return 3
}

func (api *GameAPI) getBasePosition(state *lua.State) int {
	team, _ := state.ToInteger(1)

	var pos protocol.Vector3f
	if team == 0 || team == 1 {
		pos = api.gameState.Base[team]
	}

	state.PushNumber(float64(pos.X))
	state.PushNumber(float64(pos.Y))
	state.PushNumber(float64(pos.Z))
	return 3
}

func (api *GameAPI) sendChat(state *lua.State) int {
	playerID, _ := state.ToInteger(1)
	message, _ := state.ToString(2)

	if api.server == nil {
		return 0
	}

	p, _ := api.gameState.Players.Get(uint8(playerID))
	if p != nil {
		api.server.SendChatToPlayer(p, message)
	}

	return 0
}

func (api *GameAPI) killPlayer(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.Alive = false
		p.HP = 0
		p.Unlock()

		if api.server != nil {
			api.server.KillPlayer(uint8(id), uint8(id), protocol.KillTypeTeamChange)
		}
	}

	return 0
}

func (api *GameAPI) setPlayerPosition(state *lua.State) int {
	id, _ := state.ToInteger(1)
	xNum, _ := state.ToNumber(2)
	x := float32(xNum)
	yNum, _ := state.ToNumber(3)
	y := float32(yNum)
	zNum, _ := state.ToNumber(4)
	z := float32(zNum)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.Position = protocol.Vector3f{X: x, Y: y, Z: z}
		p.Unlock()
	}

	return 0
}

func (api *GameAPI) getPlayerPosition(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushNumber(0)
		state.PushNumber(0)
		state.PushNumber(0)
		return 3
	}

	p.Lock()
	pos := p.Position
	p.Unlock()

	state.PushNumber(float64(pos.X))
	state.PushNumber(float64(pos.Y))
	state.PushNumber(float64(pos.Z))
	return 3
}

func (api *GameAPI) getPlayerTeam(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(-1)
		return 1
	}

	p.Lock()
	team := p.Team
	p.Unlock()

	state.PushInteger(int(team))
	return 1
}

func (api *GameAPI) getPlayerName(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushString("")
		return 1
	}

	p.Lock()
	name := p.Name
	p.Unlock()

	state.PushString(name)
	return 1
}

func (api *GameAPI) isPlayerAlive(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushBoolean(false)
		return 1
	}

	p.Lock()
	alive := p.Alive
	p.Unlock()

	state.PushBoolean(alive)
	return 1
}

func (api *GameAPI) getMapWidth(state *lua.State) int {
	if api.gameState.Map == nil {
		state.PushInteger(0)
		return 1
	}
	state.PushInteger(api.gameState.Map.Width())
	return 1
}

func (api *GameAPI) getMapHeight(state *lua.State) int {
	if api.gameState.Map == nil {
		state.PushInteger(0)
		return 1
	}
	state.PushInteger(api.gameState.Map.Height())
	return 1
}

func (api *GameAPI) getMapDepth(state *lua.State) int {
	if api.gameState.Map == nil {
		state.PushInteger(0)
		return 1
	}
	state.PushInteger(api.gameState.Map.Depth())
	return 1
}

func PushPlayer(state *lua.State, p *player.Player) {
	pushPlayerTable(state, p)
}

func CheckPlayer(state *lua.State, idx int, gs *gamestate.GameState) (*player.Player, error) {
	if !state.IsTable(idx) {
		return nil, fmt.Errorf("expected table at index %d", idx)
	}

	state.Field(idx, "id")
	if !state.IsNumber(-1) {
		state.Pop(1)
		return nil, fmt.Errorf("player.id is not a number")
	}
	id, _ := state.ToInteger(-1)
	state.Pop(1)

	p, _ := gs.Players.Get(uint8(id))
	if p == nil {
		return nil, fmt.Errorf("player with id %d not found", id)
	}

	return p, nil
}

func (api *GameAPI) banPlayer(state *lua.State) int {
	ip, _ := state.ToString(1)
	name, _ := state.ToString(2)
	reason, _ := state.ToString(3)
	bannedBy, _ := state.ToString(4)
	durationHours, _ := state.ToNumber(5)

	if api.banManager == nil {
		state.PushBoolean(false)
		state.PushString("ban manager not available")
		return 2
	}

	var duration time.Duration
	if durationHours == 0 {
		duration = 0
	} else {
		duration = time.Duration(durationHours * float64(time.Hour))
	}

	err := api.banManager.AddBan(ip, name, reason, bannedBy, duration)
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) unbanIP(state *lua.State) int {
	ip, _ := state.ToString(1)

	if api.banManager == nil {
		state.PushBoolean(false)
		state.PushString("ban manager not available")
		return 2
	}

	err := api.banManager.RemoveBan(ip)
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) isBanned(state *lua.State) int {
	ip, _ := state.ToString(1)

	if api.banManager == nil {
		state.PushBoolean(false)
		return 1
	}

	banned, _ := api.banManager.IsBanned(ip)
	state.PushBoolean(banned)
	return 1
}

func (api *GameAPI) kickPlayerCmd(state *lua.State) int {
	id, _ := state.ToInteger(1)
	reason, _ := state.ToString(2)

	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	api.server.KickPlayer(uint8(id), reason)

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) disconnectPlayer(state *lua.State) int {
	id, _ := state.ToInteger(1)
	reason, _ := state.ToInteger(2)

	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	p, ok := api.gameState.Players.Get(uint8(id))
	if !ok {
		state.PushBoolean(false)
		state.PushString("player not found")
		return 2
	}

	api.server.DisconnectPlayerWithReason(p, uint32(reason))

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) broadcastChat(state *lua.State) int {
	message, _ := state.ToString(1)

	if api.server == nil {
		return 0
	}

	api.server.SendChatToAll(message)
	return 0
}

func (api *GameAPI) getPlayerIP(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushString("")
		return 1
	}

	ip := p.Peer.GetAddress().String()
	state.PushString(ip)
	return 1
}

func (api *GameAPI) startVotekick(state *lua.State) int {
	instigatorID, _ := state.ToInteger(1)
	victimID, _ := state.ToInteger(2)
	reason, _ := state.ToString(3)

	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	instigator, _ := api.gameState.Players.Get(uint8(instigatorID))
	if instigator == nil {
		state.PushBoolean(false)
		state.PushString("instigator not found")
		return 2
	}

	err := api.server.StartVotekick(instigator, uint8(victimID), reason)
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) startVotemap(state *lua.State) int {
	instigatorID, _ := state.ToInteger(1)

	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	serverConfig := api.gameState.Config
	if serverConfig != nil && !serverConfig.Voting.VotemapEnabled {
		state.PushBoolean(false)
		state.PushString("votemap is disabled")
		return 2
	}

	instigator, _ := api.gameState.Players.Get(uint8(instigatorID))
	if instigator == nil {
		state.PushBoolean(false)
		state.PushString("instigator not found")
		return 2
	}

	err := api.server.StartVotemap(instigator)
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) castVote(state *lua.State) int {
	playerID, _ := state.ToInteger(1)

	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	p, _ := api.gameState.Players.Get(uint8(playerID))
	if p == nil {
		state.PushBoolean(false)
		state.PushString("player not found")
		return 2
	}

	var voteChoice interface{}
	if state.TypeOf(2) == lua.TypeBoolean {
		voteChoice = state.ToBoolean(2)
	} else if state.TypeOf(2) == lua.TypeString {
		voteChoice, _ = state.ToString(2)
	} else if state.TypeOf(2) == lua.TypeNumber {
		choiceNum, _ := state.ToInteger(2)
		activeVote := api.server.GetActiveVote()
		if activeVote != nil {
			if votemap, ok := activeVote.(*vote.Votemap); ok {
				choices := votemap.GetMapChoices()
				if choiceNum >= 1 && choiceNum <= len(choices) {
					voteChoice = choices[choiceNum-1]
				} else {
					state.PushBoolean(false)
					state.PushString(fmt.Sprintf("invalid choice number: must be 1-%d", len(choices)))
					return 2
				}
			} else {
				state.PushBoolean(false)
				state.PushString("numeric choice is only valid for map votes")
				return 2
			}
		} else {
			state.PushBoolean(false)
			state.PushString("no active vote")
			return 2
		}
	} else {
		state.PushBoolean(false)
		state.PushString("invalid choice type")
		return 2
	}

	err := api.server.CastVote(p, voteChoice)
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) cancelVote(state *lua.State) int {
	playerID, _ := state.ToInteger(1)

	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	p, _ := api.gameState.Players.Get(uint8(playerID))
	if p == nil {
		state.PushBoolean(false)
		state.PushString("player not found")
		return 2
	}

	err := api.server.CancelVote(p)
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) hasActiveVote(state *lua.State) int {
	if api.server == nil {
		state.PushBoolean(false)
		return 1
	}

	hasVote := api.server.HasActiveVote()
	state.PushBoolean(hasVote)
	return 1
}

func (api *GameAPI) getVoteChoices(state *lua.State) int {
	if api.server == nil {
		state.PushNil()
		return 1
	}

	activeVote := api.server.GetActiveVote()
	if activeVote == nil {
		state.PushNil()
		return 1
	}

	if votemap, ok := activeVote.(*vote.Votemap); ok {
		choices := votemap.GetMapChoices()
		state.CreateTable(len(choices), 0)
		for i, choice := range choices {
			state.PushInteger(i + 1)
			state.PushString(choice)
			state.SetTable(-3)
		}
		return 1
	}

	state.PushNil()
	return 1
}

func (api *GameAPI) getVoteType(state *lua.State) int {
	if api.server == nil {
		state.PushNil()
		return 1
	}

	activeVote := api.server.GetActiveVote()
	if activeVote == nil {
		state.PushNil()
		return 1
	}

	voteType := activeVote.Type()
	switch voteType {
	case vote.VoteTypeKick:
		state.PushString("kick")
	case vote.VoteTypeMap:
		state.PushString("map")
	default:
		state.PushNil()
	}
	return 1
}

func (api *GameAPI) getPlayerByName(state *lua.State) int {
	name, _ := state.ToString(1)

	var found *player.Player
	api.gameState.Players.ForEach(func(p *player.Player) {
		if p.Name == name {
			found = p
		}
	})

	pushPlayerTable(state, found)
	return 1
}

func (api *GameAPI) reloadCommands(state *lua.State) int {
	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	err := api.server.ReloadCommands()
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) reloadGamemode(state *lua.State) int {
	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	err := api.server.ReloadGamemode()
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) sendBigMessage(state *lua.State) int {
	message, _ := state.ToString(1)

	if api.server == nil {
		return 0
	}

	api.server.SendChatWithType(message, protocol.ChatTypeBig)
	return 0
}

func (api *GameAPI) sendInfoMessage(state *lua.State) int {
	message, _ := state.ToString(1)

	if api.server == nil {
		return 0
	}

	api.server.SendChatWithType(message, protocol.ChatTypeInfo)
	return 0
}

func (api *GameAPI) sendWarningMessage(state *lua.State) int {
	message, _ := state.ToString(1)

	if api.server == nil {
		return 0
	}

	api.server.SendChatWithType(message, protocol.ChatTypeWarning)
	return 0
}

func (api *GameAPI) sendErrorMessage(state *lua.State) int {
	message, _ := state.ToString(1)

	if api.server == nil {
		return 0
	}

	api.server.SendChatWithType(message, protocol.ChatTypeError)
	return 0
}

func (api *GameAPI) getAvailableCommands(state *lua.State) int {
	if api.commandManager == nil {
		state.NewTable()
		return 1
	}

	playerID, _ := state.ToInteger(1)
	p, exists := api.gameState.Players.Get(uint8(playerID))
	if !exists {
		state.NewTable()
		return 1
	}

	commands := api.commandManager.List(p)

	state.NewTable()
	for i, cmd := range commands {
		state.NewTable()

		state.PushString(cmd.Name)
		state.SetField(-2, "name")

		state.PushString(cmd.Description)
		state.SetField(-2, "description")

		state.PushString(cmd.Usage)
		state.SetField(-2, "usage")

		if len(cmd.Aliases) > 0 {
			state.NewTable()
			for j, alias := range cmd.Aliases {
				if alias != "" {
					state.PushString(alias)
					state.RawSetInt(-2, j+1)
				}
			}
			state.SetField(-2, "aliases")
		}

		state.RawSetInt(-2, i+1)
	}

	return 1
}

func (api *GameAPI) hasPermission(state *lua.State) int {
	playerID, _ := state.ToInteger(1)
	permissionName, _ := state.ToString(2)

	p, exists := api.gameState.Players.Get(uint8(playerID))
	if !exists {
		state.PushBoolean(false)
		return 1
	}

	requiredLevel := parsePermissionLevel(permissionName)
	p.RLock()
	perms := p.Permissions
	p.RUnlock()

	playerLevel := getPermissionLevel(perms)
	state.PushBoolean(playerLevel >= requiredLevel)
	return 1
}

func parsePermissionLevel(perm string) int {
	switch perm {
	case "trusted":
		return 1
	case "guard":
		return 2
	case "moderator", "mod":
		return 3
	case "admin":
		return 4
	case "manager":
		return 5
	default:
		return 0
	}
}

func getPermissionLevel(perms uint64) int {
	if perms&(1<<5) != 0 {
		return 5
	}
	if perms&(1<<4) != 0 {
		return 4
	}
	if perms&(1<<3) != 0 {
		return 3
	}
	if perms&(1<<2) != 0 {
		return 2
	}
	if perms&(1<<1) != 0 {
		return 1
	}
	return 0
}

func (api *GameAPI) setPlayerPermission(state *lua.State) int {
	playerID, _ := state.ToInteger(1)
	permissionName, _ := state.ToString(2)

	p, exists := api.gameState.Players.Get(uint8(playerID))
	if !exists {
		state.PushBoolean(false)
		state.PushString("player not found")
		return 2
	}

	var permBit uint64
	switch permissionName {
	case "trusted":
		permBit = 1 << 1
	case "guard":
		permBit = 1 << 2
	case "moderator", "mod":
		permBit = 1 << 3
	case "admin":
		permBit = 1 << 4
	case "manager":
		permBit = 1 << 5
	default:
		state.PushBoolean(false)
		state.PushString("invalid permission level")
		return 2
	}

	p.Lock()
	p.Permissions = permBit
	p.Unlock()

	state.PushBoolean(true)
	state.PushString("")
	return 2
}

func (api *GameAPI) getConfigPassword(state *lua.State) int {
	role, _ := state.ToString(1)

	if api.server == nil {
		state.PushString("")
		return 1
	}

	password := api.server.GetConfigPassword(role)
	state.PushString(password)
	return 1
}

func (api *GameAPI) getMapName(state *lua.State) int {
	if api.server == nil {
		state.PushString("")
		return 1
	}

	mapName := api.server.GetCurrentMapName()
	state.PushString(mapName)
	return 1
}

func (api *GameAPI) setPlayerHP(state *lua.State) int {
	id, _ := state.ToInteger(1)
	hp, _ := state.ToInteger(2)

	if api.server == nil {
		return 0
	}

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		return 0
	}

	p.Lock()
	p.HP = uint8(hp)
	if p.HP <= 0 {
		p.Alive = false
	}
	p.Unlock()

	return 0
}

func (api *GameAPI) setPlayerTeam(state *lua.State) int {
	id, _ := state.ToInteger(1)
	team, _ := state.ToInteger(2)

	if api.server == nil {
		return 0
	}

	api.server.SetPlayerTeam(uint8(id), uint8(team))

	return 0
}

func (api *GameAPI) healPlayer(state *lua.State) int {
	if api.server == nil {
		return 0
	}

	id, _ := state.ToInteger(1)
	api.server.RestockPlayer(uint8(id))
	return 0
}

func (api *GameAPI) setPlayerWeapon(state *lua.State) int {
	id, _ := state.ToInteger(1)
	weapon, _ := state.ToInteger(2)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.Weapon = protocol.WeaponType(weapon)
		p.Unlock()
	}

	return 0
}

func (api *GameAPI) setPlayerAmmo(state *lua.State) int {
	id, _ := state.ToInteger(1)
	primary, _ := state.ToInteger(2)
	secondary, _ := state.ToInteger(3)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.MagazineAmmo = uint8(primary)
		p.ReserveAmmo = uint8(secondary)
		p.Unlock()
	}

	return 0
}

func (api *GameAPI) setPlayerGrenades(state *lua.State) int {
	id, _ := state.ToInteger(1)
	grenades, _ := state.ToInteger(2)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.Grenades = uint8(grenades)
		p.Unlock()
	}

	return 0
}

func (api *GameAPI) setPlayerBlocks(state *lua.State) int {
	id, _ := state.ToInteger(1)
	blocks, _ := state.ToInteger(2)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.Blocks = uint8(blocks)
		p.Unlock()
	}

	return 0
}

func (api *GameAPI) respawnPlayer(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.Alive = true
		p.HP = 100
		p.Unlock()
	}

	return 0
}

func (api *GameAPI) getPlayerWeapon(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(-1)
		return 1
	}

	p.RLock()
	weapon := p.Weapon
	p.RUnlock()

	state.PushInteger(int(weapon))
	return 1
}

func (api *GameAPI) scheduleCallback(state *lua.State) int {
	seconds, _ := state.ToNumber(1)
	callback, _ := state.ToString(2)
	repeat := false
	if state.Top() >= 3 && state.IsBoolean(3) {
		repeat = state.ToBoolean(3)
	}

	if api.gamemodeVM == nil {
		state.PushInteger(-1)
		return 1
	}

	interval := time.Duration(seconds * float64(time.Second))
	timerID := api.gamemodeVM.RegisterTimer(callback, interval, repeat)
	state.PushInteger(timerID)
	return 1
}

func (api *GameAPI) cancelCallback(state *lua.State) int {
	id, _ := state.ToInteger(1)

	if api.gamemodeVM != nil {
		api.gamemodeVM.CancelTimer(id)
	}

	return 0
}

func (api *GameAPI) getServerName(state *lua.State) int {
	if api.server == nil {
		state.PushString("")
		return 1
	}

	name := api.server.GetServerName()
	state.PushString(name)
	return 1
}

func (api *GameAPI) getServerTime(state *lua.State) int {
	if api.server == nil {
		state.PushNumber(0)
		return 1
	}

	uptime := api.server.GetUptime().Seconds()
	state.PushNumber(uptime)
	return 1
}

func (api *GameAPI) getSpawnLocation(state *lua.State) int {
	team, _ := state.ToInteger(1)

	var pos protocol.Vector3f
	if team == 0 || team == 1 {
		pos = api.gameState.Base[team]
	}

	state.PushNumber(float64(pos.X))
	state.PushNumber(float64(pos.Y))
	state.PushNumber(float64(pos.Z))
	return 3
}

func (api *GameAPI) isValidPosition(state *lua.State) int {
	x, _ := state.ToInteger(1)
	y, _ := state.ToInteger(2)
	z, _ := state.ToInteger(3)

	if api.gameState.Map == nil {
		state.PushBoolean(false)
		return 1
	}

	width := api.gameState.Map.Width()
	height := api.gameState.Map.Height()
	depth := api.gameState.Map.Depth()

	valid := x >= 0 && x < width && y >= 0 && y < height && z >= 0 && z < depth
	state.PushBoolean(valid)
	return 1
}

func (api *GameAPI) getPlayerPing(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(-1)
		return 1
	}

	if p.Peer == nil {
		state.PushInteger(-1)
		return 1
	}

	rtt := p.Peer.GetRoundTripTime()
	state.PushInteger(int(rtt))
	return 1
}

func (api *GameAPI) getPlayerAmmo(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(0)
		state.PushInteger(0)
		return 2
	}

	p.RLock()
	magazine := p.MagazineAmmo
	reserve := p.ReserveAmmo
	p.RUnlock()

	state.PushInteger(int(magazine))
	state.PushInteger(int(reserve))
	return 2
}

func (api *GameAPI) getPlayerGrenades(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(0)
		return 1
	}

	p.RLock()
	grenades := p.Grenades
	p.RUnlock()

	state.PushInteger(int(grenades))
	return 1
}

func (api *GameAPI) getPlayerBlocks(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(0)
		return 1
	}

	p.RLock()
	blocks := p.Blocks
	p.RUnlock()

	state.PushInteger(int(blocks))
	return 1
}

func (api *GameAPI) getPlayerColor(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(0)
		state.PushInteger(0)
		state.PushInteger(0)
		return 3
	}

	p.RLock()
	color := p.Color
	p.RUnlock()

	state.PushInteger(int(color.R))
	state.PushInteger(int(color.G))
	state.PushInteger(int(color.B))
	return 3
}

func (api *GameAPI) getPlayerTool(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushInteger(-1)
		return 1
	}

	p.RLock()
	tool := p.Tool
	p.RUnlock()

	state.PushInteger(int(tool))
	return 1
}

func (api *GameAPI) getPlayerState(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushNil()
		return 1
	}

	p.RLock()
	crouching := p.Crouching
	sprinting := p.Sprinting
	airborne := p.Airborne
	p.RUnlock()

	state.NewTable()
	state.PushBoolean(crouching)
	state.SetField(-2, "crouching")
	state.PushBoolean(sprinting)
	state.SetField(-2, "sprinting")
	state.PushBoolean(airborne)
	state.SetField(-2, "airborne")

	return 1
}

func (api *GameAPI) setPlayerOrientation(state *lua.State) int {
	id, _ := state.ToInteger(1)
	xNum, _ := state.ToNumber(2)
	x := float32(xNum)
	yNum, _ := state.ToNumber(3)
	y := float32(yNum)
	zNum, _ := state.ToNumber(4)
	z := float32(zNum)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p != nil {
		p.Lock()
		p.Orientation = protocol.Vector3f{X: x, Y: y, Z: z}
		p.Unlock()
	}

	return 0
}

func (api *GameAPI) getPlayerOrientation(state *lua.State) int {
	id, _ := state.ToInteger(1)

	p, _ := api.gameState.Players.Get(uint8(id))
	if p == nil {
		state.PushNumber(0)
		state.PushNumber(0)
		state.PushNumber(0)
		return 3
	}

	p.RLock()
	ori := p.Orientation
	p.RUnlock()

	state.PushNumber(float64(ori.X))
	state.PushNumber(float64(ori.Y))
	state.PushNumber(float64(ori.Z))
	return 3
}

func (api *GameAPI) distance3D(state *lua.State) int {
	x1, _ := state.ToNumber(1)
	y1, _ := state.ToNumber(2)
	z1, _ := state.ToNumber(3)
	x2, _ := state.ToNumber(4)
	y2, _ := state.ToNumber(5)
	z2, _ := state.ToNumber(6)

	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1

	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	state.PushNumber(distance)
	return 1
}

func (api *GameAPI) distance2D(state *lua.State) int {
	x1, _ := state.ToNumber(1)
	y1, _ := state.ToNumber(2)
	x2, _ := state.ToNumber(3)
	y2, _ := state.ToNumber(4)

	dx := x2 - x1
	dy := y2 - y1

	distance := math.Sqrt(dx*dx + dy*dy)
	state.PushNumber(distance)
	return 1
}

func (api *GameAPI) rgbToColor(state *lua.State) int {
	r, _ := state.ToInteger(1)
	g, _ := state.ToInteger(2)
	b, _ := state.ToInteger(3)

	color := uint32(b) | (uint32(g) << 8) | (uint32(r) << 16)
	state.PushInteger(int(color))
	return 1
}

func (api *GameAPI) colorToRgb(state *lua.State) int {
	color, _ := state.ToInteger(1)

	r := (color >> 16) & 0xFF
	g := (color >> 8) & 0xFF
	b := color & 0xFF

	state.PushInteger(r)
	state.PushInteger(g)
	state.PushInteger(b)
	return 3
}

func (api *GameAPI) forEachPlayer(state *lua.State) int {
	if !state.IsFunction(1) {
		return 0
	}

	var players []*player.Player
	api.gameState.Players.ForEach(func(p *player.Player) {
		players = append(players, p)
	})

	for _, p := range players {
		state.PushValue(1)
		pushPlayerTable(state, p)
		if err := state.ProtectedCall(1, 0, 0); err != nil {
			return 0
		}
	}

	return 0
}

func (api *GameAPI) clamp(state *lua.State) int {
	value, _ := state.ToNumber(1)
	min, _ := state.ToNumber(2)
	max, _ := state.ToNumber(3)

	if value < min {
		value = min
	} else if value > max {
		value = max
	}

	state.PushNumber(value)
	return 1
}

func (api *GameAPI) lerp(state *lua.State) int {
	a, _ := state.ToNumber(1)
	b, _ := state.ToNumber(2)
	t, _ := state.ToNumber(3)

	result := a + (b-a)*t
	state.PushNumber(result)
	return 1
}

func (api *GameAPI) getTimerInfo(state *lua.State) int {
	id, _ := state.ToInteger(1)

	if api.gamemodeVM == nil {
		state.PushNil()
		return 1
	}

	api.gamemodeVM.timerLock.Lock()
	timer, exists := api.gamemodeVM.timers[id]
	if !exists {
		api.gamemodeVM.timerLock.Unlock()
		state.PushNil()
		return 1
	}

	remaining := time.Until(timer.NextRun).Seconds()
	interval := timer.Interval.Seconds()
	repeat := timer.Repeat
	api.gamemodeVM.timerLock.Unlock()

	state.NewTable()
	state.PushNumber(remaining)
	state.SetField(-2, "remaining")
	state.PushNumber(interval)
	state.SetField(-2, "interval")
	state.PushBoolean(repeat)
	state.SetField(-2, "repeat")

	return 1
}

func (api *GameAPI) pauseTimer(state *lua.State) int {
	state.PushBoolean(false)
	state.PushString("pause_timer not yet implemented - requires vm timer structure changes")
	return 2
}

func (api *GameAPI) resumeTimer(state *lua.State) int {
	state.PushBoolean(false)
	state.PushString("resume_timer not yet implemented - requires vm timer structure changes")
	return 2
}

func (api *GameAPI) createExplosion(state *lua.State) int {
	state.PushBoolean(false)
	state.PushString("create_explosion not yet implemented - requires server packet sending")
	return 2
}

func (api *GameAPI) saveMap(state *lua.State) int {
	filename, _ := state.ToString(1)

	if api.server == nil {
		state.PushBoolean(false)
		state.PushString("server not available")
		return 2
	}

	savedPath, err := api.server.SaveMap(filename)
	if err != nil {
		state.PushBoolean(false)
		state.PushString(err.Error())
		return 2
	}

	state.PushBoolean(true)
	state.PushString(savedPath)
	return 2
}

func (api *GameAPI) sendTerritoryCapture(state *lua.State) int {
	playerID, _ := state.ToInteger(1)
	entityID, _ := state.ToInteger(2)
	winning, _ := state.ToInteger(3)
	territoryState, _ := state.ToInteger(4)

	if api.server == nil {
		return 0
	}

	api.server.BroadcastTerritoryCapture(uint8(playerID), uint8(entityID), uint8(winning), uint8(territoryState))
	return 0
}

func (api *GameAPI) sendProgressBar(state *lua.State) int {
	entityID, _ := state.ToInteger(1)
	capturingTeam, _ := state.ToInteger(2)
	rateNum, _ := state.ToNumber(3)
	rate := int8(rateNum)
	progressNum, _ := state.ToNumber(4)
	progress := float32(progressNum)

	if api.server == nil {
		return 0
	}

	api.server.BroadcastProgressBar(uint8(entityID), uint8(capturingTeam), rate, progress)
	return 0
}

func (api *GameAPI) getConfigValue(state *lua.State) int {
	key, _ := state.ToString(1)

	if api.gameState == nil || api.gameState.Config == nil {
		state.PushNil()
		return 1
	}

	cfg := api.gameState.Config.Server

	switch key {
	case "capture_limit":
		state.PushInteger(cfg.CaptureLimit)
	case "capture_time_bonus":
		state.PushNumber(cfg.CaptureTimeBonus)
	case "flag_return_time":
		state.PushNumber(cfg.FlagReturnTime)
	case "kill_limit":
		state.PushInteger(cfg.KillLimit)
	case "intel_points":
		state.PushInteger(cfg.IntelPoints)
	case "remove_intel":
		state.PushBoolean(cfg.RemoveIntel)
	case "headshot_multiplier":
		state.PushNumber(cfg.HeadshotMultiplier)
	case "enable_killstreaks":
		state.PushBoolean(cfg.EnableKillstreaks)
	case "babel_reverse":
		state.PushBoolean(cfg.BabelReverse)
	case "babel_capture_limit":
		state.PushInteger(cfg.BabelCaptureLimit)
	case "regenerate_tower":
		state.PushBoolean(cfg.RegenerateTower)
	case "regeneration_rate":
		state.PushNumber(cfg.RegenerationRate)
	case "arena_score_limit":
		state.PushInteger(cfg.ArenaScoreLimit)
	case "timeout_is_draw":
		state.PushBoolean(cfg.ArenaTimeoutIsDraw)
	case "sudden_death_enabled":
		state.PushBoolean(cfg.ArenaSuddenDeath)
	case "tc_max_score":
		state.PushInteger(cfg.TCMaxScore)
	case "tc_capture_distance":
		state.PushNumber(cfg.TCCaptureDistance)
	case "tc_capture_rate":
		state.PushNumber(cfg.TCCaptureRate)
	default:
		state.PushNil()
	}

	return 1
}
