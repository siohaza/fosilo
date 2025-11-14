package gamemode

import (
	"fmt"
	"log/slog"

	"github.com/siohaza/fosilo/internal/gamestate"
	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
	"github.com/siohaza/fosilo/pkg/lua"
)

type LuaGameMode struct {
	vm     *lua.VM
	api    *lua.GameAPI
	name   string
	logger *slog.Logger
}

func NewLuaGameMode(scriptPath string, gs *gamestate.GameState, api *lua.GameAPI, logger *slog.Logger) (*LuaGameMode, error) {
	vm := lua.NewVM()

	if api != nil {
		api.RegisterFunctions(vm)
	}

	if err := vm.LoadFile(scriptPath); err != nil {
		vm.Close()
		return nil, fmt.Errorf("failed to load gamemode script: %w", err)
	}

	name, err := vm.GetGlobalString("name")
	if err != nil {
		name = "lua_gamemode"
	}

	gm := &LuaGameMode{
		vm:     vm,
		api:    api,
		name:   name,
		logger: logger,
	}

	if api != nil {
		api.SetGamemodeVM(vm)
	}

	if vm.HasFunction("on_init") {
		if err := vm.CallFunction("on_init"); err != nil {
			vm.Close()
			return nil, fmt.Errorf("failed to call on_init: %w", err)
		}
	}

	return gm, nil
}

func (gm *LuaGameMode) UpdateTimers() error {
	if gm.vm != nil {
		return gm.vm.UpdateTimers()
	}
	return nil
}

func (gm *LuaGameMode) Name() string {
	return gm.name
}

func (gm *LuaGameMode) OnPlayerSpawn(p *player.Player) {
	if !gm.vm.HasFunction("on_player_spawn") {
		return
	}

	gm.vm.State().Global("on_player_spawn")
	lua.PushPlayer(gm.vm.State(), p)
	if err := gm.vm.State().ProtectedCall(1, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_player_spawn error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnPlayerKill(killer, victim *player.Player, killType protocol.KillType) {
	if !gm.vm.HasFunction("on_player_kill") {
		return
	}

	gm.vm.State().Global("on_player_kill")
	lua.PushPlayer(gm.vm.State(), killer)
	lua.PushPlayer(gm.vm.State(), victim)
	gm.vm.State().PushInteger(int(killType))
	if err := gm.vm.State().ProtectedCall(3, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_player_kill error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnPlayerUpdate(p *player.Player) {
	if !gm.vm.HasFunction("on_player_update") {
		return
	}

	gm.vm.State().Global("on_player_update")
	lua.PushPlayer(gm.vm.State(), p)
	if err := gm.vm.State().ProtectedCall(1, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_player_update error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnIntelPickup(p *player.Player, team uint8) bool {
	if gm.vm.HasFunction("on_intel_pickup") {
		results, err := gm.vm.CallFunctionWithReturn("on_intel_pickup", 1, int(p.ID), int(team))
		if err != nil {
			if gm.logger != nil {
				gm.logger.Error("lua gamemode on_intel_pickup error", "error", err)
			}
			return true
		}
		if len(results) > 0 {
			if allow, ok := results[0].(bool); ok {
				return allow
			}
		}
	}
	return true
}

func (gm *LuaGameMode) OnIntelCapture(p *player.Player, team uint8) bool {
	if gm.vm.HasFunction("on_intel_capture") {
		results, err := gm.vm.CallFunctionWithReturn("on_intel_capture", 1, int(p.ID), int(team))
		if err != nil {
			if gm.logger != nil {
				gm.logger.Error("lua gamemode on_intel_capture error", "error", err)
			}
			return true
		}
		if len(results) > 0 {
			if allow, ok := results[0].(bool); ok {
				return allow
			}
		}
	}
	return true
}

func (gm *LuaGameMode) CheckWinCondition() (bool, uint8) {
	if gm.vm.HasFunction("check_win_condition") {
		results, err := gm.vm.CallFunctionWithReturn("check_win_condition", 2)
		if err != nil {
			if gm.logger != nil {
				gm.logger.Error("lua gamemode check_win_condition error", "error", err)
			}
			return false, 0
		}
		if len(results) >= 2 {
			won := false
			winningTeam := uint8(0)

			if w, ok := results[0].(bool); ok {
				won = w
			}
			if t, ok := results[1].(float64); ok {
				winningTeam = uint8(t)
			}

			return won, winningTeam
		}
	}
	return false, 0
}

func (gm *LuaGameMode) ShouldRotateMap() bool {
	if gm.vm.HasFunction("should_rotate_map") {
		results, err := gm.vm.CallFunctionWithReturn("should_rotate_map", 1)
		if err != nil {
			if gm.logger != nil {
				gm.logger.Error("lua gamemode should_rotate_map error", "error", err)
			}
			return false
		}
		if len(results) > 0 {
			if rotate, ok := results[0].(bool); ok {
				return rotate
			}
		}
	}
	return false
}

func (gm *LuaGameMode) Close() {
	if gm.vm != nil {
		gm.vm.Close()
	}
}

func (gm *LuaGameMode) OnConnect(playerID uint8) {
	if !gm.vm.HasFunction("on_connect") {
		return
	}

	if err := gm.vm.CallFunction("on_connect", int(playerID)); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_connect error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnDisconnect(playerID uint8) {
	if !gm.vm.HasFunction("on_disconnect") {
		return
	}

	if err := gm.vm.CallFunction("on_disconnect", int(playerID)); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_disconnect error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnPlayerJoin(p *player.Player) {
	if !gm.vm.HasFunction("on_player_join") {
		return
	}

	gm.vm.State().Global("on_player_join")
	lua.PushPlayer(gm.vm.State(), p)
	if err := gm.vm.State().ProtectedCall(1, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_player_join error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnPlayerDamage(victim *player.Player, damage uint8, source protocol.Vector3f) {
	if !gm.vm.HasFunction("on_player_damage") {
		return
	}

	state := gm.vm.State()
	state.Global("on_player_damage")
	lua.PushPlayer(state, victim)
	state.PushInteger(int(damage))
	state.PushNumber(float64(source.X))
	state.PushNumber(float64(source.Y))
	state.PushNumber(float64(source.Z))
	if err := state.ProtectedCall(5, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_player_damage error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnChatMessage(p *player.Player, message string) bool {
	if gm.vm.HasFunction("on_chat_message") {
		state := gm.vm.State()
		state.Global("on_chat_message")
		lua.PushPlayer(state, p)
		state.PushString(message)

		if err := state.ProtectedCall(2, 1, 0); err != nil {
			if gm.logger != nil {
				gm.logger.Error("lua gamemode on_chat_message error", "error", err)
			}
			return true
		}

		if state.IsBoolean(-1) {
			allow := state.ToBoolean(-1)
			state.Pop(1)
			return allow
		}
		state.Pop(1)
	}
	return true
}

func (gm *LuaGameMode) OnBlockPlace(p *player.Player, x, y, z int) bool {
	if gm.vm.HasFunction("on_block_place") {
		state := gm.vm.State()
		state.Global("on_block_place")
		lua.PushPlayer(state, p)
		state.PushInteger(x)
		state.PushInteger(y)
		state.PushInteger(z)

		if err := state.ProtectedCall(4, 1, 0); err != nil {
			if gm.logger != nil {
				gm.logger.Error("lua gamemode on_block_place error", "error", err)
			}
			return true
		}

		if state.IsBoolean(-1) {
			allow := state.ToBoolean(-1)
			state.Pop(1)
			return allow
		}
		state.Pop(1)
	}
	return true
}

func (gm *LuaGameMode) OnBlockDestroy(p *player.Player, x, y, z int) bool {
	if gm.vm.HasFunction("on_block_destroy") {
		state := gm.vm.State()
		state.Global("on_block_destroy")
		lua.PushPlayer(state, p)
		state.PushInteger(x)
		state.PushInteger(y)
		state.PushInteger(z)

		if err := state.ProtectedCall(4, 1, 0); err != nil {
			if gm.logger != nil {
				gm.logger.Error("lua gamemode on_block_destroy error", "error", err)
			}
			return true
		}

		if state.IsBoolean(-1) {
			allow := state.ToBoolean(-1)
			state.Pop(1)
			return allow
		}
		state.Pop(1)
	}
	return true
}

func (gm *LuaGameMode) OnIntelDrop(p *player.Player, team uint8) {
	if !gm.vm.HasFunction("on_intel_drop") {
		return
	}

	_, err := gm.vm.CallFunctionWithReturn("on_intel_drop", 0, int(p.ID), int(team))
	if err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_intel_drop error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnWeaponFire(p *player.Player) {
	if !gm.vm.HasFunction("on_weapon_fire") {
		return
	}

	gm.vm.State().Global("on_weapon_fire")
	lua.PushPlayer(gm.vm.State(), p)
	if err := gm.vm.State().ProtectedCall(1, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_weapon_fire error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnGrenadeToss(p *player.Player) {
	if !gm.vm.HasFunction("on_grenade_toss") {
		return
	}

	gm.vm.State().Global("on_grenade_toss")
	lua.PushPlayer(gm.vm.State(), p)
	if err := gm.vm.State().ProtectedCall(1, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_grenade_toss error", "error", err)
		}
	}
}

func (gm *LuaGameMode) OnRestock(p *player.Player) {
	if !gm.vm.HasFunction("on_restock") {
		return
	}

	gm.vm.State().Global("on_restock")
	lua.PushPlayer(gm.vm.State(), p)
	if err := gm.vm.State().ProtectedCall(1, 0, 0); err != nil {
		if gm.logger != nil {
			gm.logger.Error("lua gamemode on_restock error", "error", err)
		}
	}
}
