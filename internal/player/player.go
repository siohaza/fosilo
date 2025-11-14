package player

import (
	"sync"
	"time"

	"github.com/siohaza/fosilo/internal/protocol"

	"github.com/codecat/go-enet"
)

func GetClock() uint64 {
	return uint64(time.Now().UnixNano())
}

type Player struct {
	ID            uint8
	Peer          enet.Peer
	Name          string
	Team          uint8
	Weapon        protocol.WeaponType
	Tool          protocol.ItemType
	Color         protocol.Color3b
	Position      protocol.Vector3f
	EyePos        protocol.Vector3f
	Orientation   protocol.Vector3f
	Velocity      protocol.Vector3f
	HP            uint8
	Blocks        uint8
	Grenades      uint8
	MagazineAmmo  uint8
	ReserveAmmo   uint8
	Kills         uint32
	Deaths        uint32
	Alive         bool
	Crouching     bool
	Airborne      bool
	Wade          bool
	Sprinting     bool
	PrimaryFire   bool
	SecondaryFire bool
	KeyStates     protocol.KeyState
	State         PlayerState

	MoveForward        bool
	MoveBackwards      bool
	MoveLeft           bool
	MoveRight          bool
	Jumping            bool
	Sneaking           bool
	LastClimb          float32
	RespawnTime        time.Time
	LastShotTime       time.Time
	Reloading          bool
	ReloadTime         time.Time
	LastWaterDamage    time.Time
	LastBoundaryDamage time.Time
	HasIntel           bool

	LastBlockPlaceTime   time.Time
	LastBlockDestroyTime time.Time
	BlockPlaceQuota      int
	BlockDestroyQuota    int
	LastPositionUpdate   time.Time
	LastPosition         protocol.Vector3f
	LastRestockTime      time.Time

	NextBulletFireClock      uint64
	NextBlockPlacementClock  uint64
	NextBlock1DestroyClock   uint64
	NextBlock3DestroyClock   uint64
	ReloadClock              uint64
	LastUpdatedPositionClock uint64

	Permissions         uint64
	Muted               bool
	Invisible           bool
	Client              byte
	Version             protocol.Vector3f
	OSInfo              string
	HandshakeChallenge  uint32
	HandshakeComplete   bool
	VersionInfoReceived bool
	ClientIdentifier    byte
	VersionMajor        int8
	VersionMinor        int8
	VersionRevision     int8
	SupportedExtensions map[protocol.ExtensionID]uint8

	PacketCounts        map[protocol.PacketType]int
	PacketCountWindow   time.Time
	TotalPacketCount    int
	LastRateLimitReset  time.Time
	RateLimitViolations int

	mu sync.RWMutex
}

type PlayerState int

const (
	PlayerStateDisconnected PlayerState = iota
	PlayerStateConnecting
	PlayerStateLoading
	PlayerStateWaitingForExistingPlayer
	PlayerStateReady
	PlayerStateDead
)

func New(id uint8, peer enet.Peer) *Player {
	return &Player{
		ID:                  id,
		Peer:                peer,
		State:               PlayerStateConnecting,
		HP:                  protocol.InitialHP,
		Blocks:              protocol.InitialBlocks,
		Grenades:            protocol.InitialGrenades,
		Alive:               false,
		SupportedExtensions: make(map[protocol.ExtensionID]uint8),
		PacketCounts:        make(map[protocol.PacketType]int),
		LastRateLimitReset:  time.Now(),
	}
}

func (p *Player) Lock() {
	p.mu.Lock()
}

func (p *Player) Unlock() {
	p.mu.Unlock()
}

func (p *Player) RLock() {
	p.mu.RLock()
}

func (p *Player) RUnlock() {
	p.mu.RUnlock()
}

func (p *Player) SetTeam(team uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Team = team
}

func (p *Player) SetWeapon(weapon protocol.WeaponType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Weapon = weapon
	p.MagazineAmmo = protocol.GetDefaultMagazineAmmo(weapon)
	p.ReserveAmmo = protocol.GetDefaultReserveAmmo(weapon)
}

func (p *Player) SetPosition(pos protocol.Vector3f) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Position = pos
}

func (p *Player) SetOrientation(ori protocol.Vector3f) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Orientation = ori
}

func (p *Player) GetPosition() protocol.Vector3f {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Position
}

func (p *Player) GetOrientation() protocol.Vector3f {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Orientation
}

func (p *Player) Damage(amount uint8, source protocol.Vector3f, damageType uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Alive || p.HP == 0 {
		return
	}

	if amount >= p.HP {
		p.HP = 0
		p.Alive = false
		p.Deaths++
	} else {
		p.HP -= amount
	}
}

func (p *Player) Kill() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.HP = 0
	p.Alive = false
	p.Deaths++
}

func (p *Player) Respawn(position protocol.Vector3f) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.HP = protocol.InitialHP
	p.Blocks = protocol.InitialBlocks
	p.Grenades = protocol.InitialGrenades
	p.MagazineAmmo = protocol.GetDefaultMagazineAmmo(p.Weapon)
	p.ReserveAmmo = protocol.GetDefaultReserveAmmo(p.Weapon)
	p.Position = position
	p.EyePos = position
	p.Alive = true
	p.State = PlayerStateReady
	p.Velocity = protocol.Vector3f{X: 0, Y: 0, Z: 0}
	p.Airborne = false
	p.Crouching = false
	p.Wade = false
	p.Jumping = false
	p.LastClimb = 0
	p.Tool = protocol.ItemTypeGun
	p.Reloading = false
	p.HasIntel = false
}

func (p *Player) Restock() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Blocks < protocol.MaxBlocks {
		p.Blocks = protocol.MaxBlocks
	}
	if p.Grenades < protocol.MaxGrenades {
		p.Grenades = protocol.MaxGrenades
	}
	if p.HP < protocol.MaxHP {
		p.HP = protocol.MaxHP
	}

	p.MagazineAmmo = protocol.GetDefaultMagazineAmmo(p.Weapon)
	p.ReserveAmmo = protocol.GetDefaultReserveAmmo(p.Weapon)
	p.Reloading = false
}

func (p *Player) NeedsRestock() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.HP < protocol.MaxHP ||
		p.Blocks < protocol.MaxBlocks ||
		p.Grenades < protocol.MaxGrenades {
		return true
	}

	maxMag := protocol.GetDefaultMagazineAmmo(p.Weapon)
	maxReserve := protocol.GetDefaultReserveAmmo(p.Weapon)

	return p.MagazineAmmo < maxMag || p.ReserveAmmo < maxReserve
}

func (p *Player) CanShoot() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.Alive || p.Tool != protocol.ItemTypeGun || p.Reloading {
		return false
	}

	currentClock := GetClock()
	return currentClock >= p.NextBulletFireClock && p.MagazineAmmo > 0
}

func (p *Player) Shoot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Alive || p.Tool != protocol.ItemTypeGun || p.Reloading || p.MagazineAmmo == 0 {
		return false
	}

	currentClock := GetClock()
	if currentClock < p.NextBulletFireClock {
		return false
	}

	p.MagazineAmmo--
	p.LastShotTime = time.Now()
	fireDelayNanos := uint64(protocol.GetFireDelay(p.Weapon)) * 1000000
	p.NextBulletFireClock = currentClock + fireDelayNanos
	return true
}

func (p *Player) StartReload() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Alive || p.Reloading || p.ReserveAmmo == 0 {
		return false
	}

	maxAmmo := protocol.GetDefaultMagazineAmmo(p.Weapon)
	if p.MagazineAmmo >= maxAmmo {
		return false
	}

	p.Reloading = true
	p.ReloadTime = time.Now().Add(2500 * time.Millisecond)
	currentClock := GetClock()
	p.ReloadClock = currentClock + 2500*1000000
	return true
}

func (p *Player) FinishReload() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Reloading {
		return
	}

	maxAmmo := protocol.GetDefaultMagazineAmmo(p.Weapon)
	needed := maxAmmo - p.MagazineAmmo

	if needed > p.ReserveAmmo {
		needed = p.ReserveAmmo
	}

	p.MagazineAmmo += needed
	p.ReserveAmmo -= needed
	p.Reloading = false
}

func (p *Player) UpdateReload() bool {
	p.mu.RLock()
	if !p.Reloading {
		p.mu.RUnlock()
		return false
	}

	currentClock := GetClock()
	if currentClock < p.ReloadClock {
		p.mu.RUnlock()
		return false
	}
	p.mu.RUnlock()

	p.FinishReload()
	return true
}

type Manager struct {
	players map[uint8]*Player
	mu      sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		players: make(map[uint8]*Player),
	}
}

func (m *Manager) Add(player *Player) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.players[player.ID] = player
}

func (m *Manager) Remove(id uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.players, id)
}

func (m *Manager) Get(id uint8) (*Player, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	player, ok := m.players[id]
	return player, ok
}

func (m *Manager) GetByPeer(peer enet.Peer) (*Player, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, player := range m.players {
		if player.Peer == peer {
			return player, true
		}
	}
	return nil, false
}

func (m *Manager) GetAll() []*Player {
	m.mu.RLock()
	defer m.mu.RUnlock()

	players := make([]*Player, 0, len(m.players))
	for _, player := range m.players {
		players = append(players, player)
	}
	return players
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.players)
}

func (m *Manager) FindFreeID(maxPlayers int) (uint8, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id := uint8(0); id < uint8(maxPlayers); id++ {
		if _, exists := m.players[id]; !exists {
			return id, true
		}
	}
	return 0, false
}

func (m *Manager) ForEach(fn func(*Player)) {
	m.mu.RLock()
	players := make([]*Player, 0, len(m.players))
	for _, player := range m.players {
		players = append(players, player)
	}
	m.mu.RUnlock()

	for _, player := range players {
		if player.GetState() == PlayerStateDisconnected {
			continue
		}
		fn(player)
	}
}

func (m *Manager) Contains(id uint8) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.players[id]
	return exists
}

func (p *Player) IsAlive() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Alive
}

func (p *Player) SupportsExtension(extID protocol.ExtensionID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, supported := p.SupportedExtensions[extID]
	return supported
}

func (p *Player) AddExtension(extID protocol.ExtensionID, version uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.SupportedExtensions[extID] = version
}

func (p *Player) GetState() PlayerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.State
}

func (p *Player) GetName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Name
}

func (p *Player) GetRespawnTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.RespawnTime
}

func (p *Player) GetHP() uint8 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.HP
}

func (p *Player) GetTeam() uint8 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Team
}

func (p *Player) GetWeapon() protocol.WeaponType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Weapon
}
