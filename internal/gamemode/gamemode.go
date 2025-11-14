package gamemode

import (
	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
)

type GameMode interface {
	Name() string
	OnPlayerSpawn(p *player.Player)
	OnPlayerKill(killer, victim *player.Player, killType protocol.KillType)
	OnPlayerUpdate(p *player.Player)
	OnIntelPickup(p *player.Player, team uint8) bool
	OnIntelCapture(p *player.Player, team uint8) bool
	CheckWinCondition() (won bool, winningTeam uint8)
	ShouldRotateMap() bool
}

type BaseGameMode struct {
	name string
}

func (b *BaseGameMode) Name() string {
	return b.name
}

func (b *BaseGameMode) OnPlayerSpawn(p *player.Player) {}

func (b *BaseGameMode) OnPlayerKill(killer, victim *player.Player, killType protocol.KillType) {}

func (b *BaseGameMode) OnPlayerUpdate(p *player.Player) {}

func (b *BaseGameMode) OnIntelPickup(p *player.Player, team uint8) bool {
	return true
}

func (b *BaseGameMode) OnIntelCapture(p *player.Player, team uint8) bool {
	return true
}

func (b *BaseGameMode) CheckWinCondition() (bool, uint8) {
	return false, 0
}

func (b *BaseGameMode) ShouldRotateMap() bool {
	return false
}
