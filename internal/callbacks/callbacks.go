package callbacks

import (
	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
)

type Callbacks interface {
	OnConnect(playerID uint8)
	OnDisconnect(playerID uint8)
	OnPlayerJoin(p *player.Player)
	OnPlayerKill(killer *player.Player, victim *player.Player, killType protocol.KillType)
	OnPlayerSpawn(p *player.Player)
	OnPlayerDamage(victim *player.Player, damage uint8, source protocol.Vector3f)
	OnChatMessage(p *player.Player, message string) bool
	OnBlockPlace(p *player.Player, x, y, z int) bool
	OnBlockDestroy(p *player.Player, x, y, z int) bool
	OnIntelPickup(p *player.Player, team uint8) bool
	OnIntelCapture(p *player.Player, team uint8) bool
	OnIntelDrop(p *player.Player, team uint8)
	OnWeaponFire(p *player.Player)
	OnGrenadeToss(p *player.Player)
	OnRestock(p *player.Player)
}

type DefaultCallbacks struct{}

func (d *DefaultCallbacks) OnConnect(playerID uint8)      {}
func (d *DefaultCallbacks) OnDisconnect(playerID uint8)   {}
func (d *DefaultCallbacks) OnPlayerJoin(p *player.Player) {}
func (d *DefaultCallbacks) OnPlayerKill(killer *player.Player, victim *player.Player, killType protocol.KillType) {
}
func (d *DefaultCallbacks) OnPlayerSpawn(p *player.Player) {}
func (d *DefaultCallbacks) OnPlayerDamage(victim *player.Player, damage uint8, source protocol.Vector3f) {
}
func (d *DefaultCallbacks) OnChatMessage(p *player.Player, message string) bool { return true }
func (d *DefaultCallbacks) OnBlockPlace(p *player.Player, x, y, z int) bool     { return true }
func (d *DefaultCallbacks) OnBlockDestroy(p *player.Player, x, y, z int) bool   { return true }
func (d *DefaultCallbacks) OnIntelPickup(p *player.Player, team uint8) bool     { return true }
func (d *DefaultCallbacks) OnIntelCapture(p *player.Player, team uint8) bool    { return true }
func (d *DefaultCallbacks) OnIntelDrop(p *player.Player, team uint8)            {}
func (d *DefaultCallbacks) OnWeaponFire(p *player.Player)                       {}
func (d *DefaultCallbacks) OnGrenadeToss(p *player.Player)                      {}
func (d *DefaultCallbacks) OnRestock(p *player.Player)                          {}

type CallbackChain struct {
	callbacks []Callbacks
}

func NewCallbackChain() *CallbackChain {
	return &CallbackChain{
		callbacks: make([]Callbacks, 0),
	}
}

func (c *CallbackChain) Register(cb Callbacks) {
	c.callbacks = append(c.callbacks, cb)
}

func (c *CallbackChain) OnConnect(playerID uint8) {
	for _, cb := range c.callbacks {
		cb.OnConnect(playerID)
	}
}

func (c *CallbackChain) OnDisconnect(playerID uint8) {
	for _, cb := range c.callbacks {
		cb.OnDisconnect(playerID)
	}
}

func (c *CallbackChain) OnPlayerJoin(p *player.Player) {
	for _, cb := range c.callbacks {
		cb.OnPlayerJoin(p)
	}
}

func (c *CallbackChain) OnPlayerKill(killer *player.Player, victim *player.Player, killType protocol.KillType) {
	for _, cb := range c.callbacks {
		cb.OnPlayerKill(killer, victim, killType)
	}
}

func (c *CallbackChain) OnPlayerSpawn(p *player.Player) {
	for _, cb := range c.callbacks {
		cb.OnPlayerSpawn(p)
	}
}

func (c *CallbackChain) OnPlayerDamage(victim *player.Player, damage uint8, source protocol.Vector3f) {
	for _, cb := range c.callbacks {
		cb.OnPlayerDamage(victim, damage, source)
	}
}

func (c *CallbackChain) OnChatMessage(p *player.Player, message string) bool {
	for _, cb := range c.callbacks {
		if !cb.OnChatMessage(p, message) {
			return false
		}
	}
	return true
}

func (c *CallbackChain) OnBlockPlace(p *player.Player, x, y, z int) bool {
	for _, cb := range c.callbacks {
		if !cb.OnBlockPlace(p, x, y, z) {
			return false
		}
	}
	return true
}

func (c *CallbackChain) OnBlockDestroy(p *player.Player, x, y, z int) bool {
	for _, cb := range c.callbacks {
		if !cb.OnBlockDestroy(p, x, y, z) {
			return false
		}
	}
	return true
}

func (c *CallbackChain) OnIntelPickup(p *player.Player, team uint8) bool {
	for _, cb := range c.callbacks {
		if !cb.OnIntelPickup(p, team) {
			return false
		}
	}
	return true
}

func (c *CallbackChain) OnIntelCapture(p *player.Player, team uint8) bool {
	for _, cb := range c.callbacks {
		if !cb.OnIntelCapture(p, team) {
			return false
		}
	}
	return true
}

func (c *CallbackChain) OnIntelDrop(p *player.Player, team uint8) {
	for _, cb := range c.callbacks {
		cb.OnIntelDrop(p, team)
	}
}

func (c *CallbackChain) OnWeaponFire(p *player.Player) {
	for _, cb := range c.callbacks {
		cb.OnWeaponFire(p)
	}
}

func (c *CallbackChain) OnGrenadeToss(p *player.Player) {
	for _, cb := range c.callbacks {
		cb.OnGrenadeToss(p)
	}
}

func (c *CallbackChain) OnRestock(p *player.Player) {
	for _, cb := range c.callbacks {
		cb.OnRestock(p)
	}
}
