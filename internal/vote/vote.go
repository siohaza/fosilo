package vote

import (
	"fmt"
	"sync"
	"time"

	"github.com/siohaza/fosilo/internal/player"
)

type VoteType int

const (
	VoteTypeKick VoteType = iota
	VoteTypeMap
)

type Vote interface {
	Type() VoteType
	Instigator() *player.Player
	Start() error
	CastVote(p *player.Player, choice interface{}) error
	Cancel() error
	Update() bool
	IsActive() bool
	GetStatus() string
	Timeout()
}

type Manager struct {
	activeVote Vote
	cooldowns  map[uint8]time.Time
	mu         sync.RWMutex
	stopChan   chan struct{}
}

func NewManager() *Manager {
	return &Manager{
		cooldowns: make(map[uint8]time.Time),
		stopChan:  make(chan struct{}),
	}
}

func (m *Manager) HasActiveVote() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeVote != nil && m.activeVote.IsActive()
}

func (m *Manager) GetActiveVote() Vote {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeVote
}

func (m *Manager) StartVote(v Vote) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeVote != nil && m.activeVote.IsActive() {
		return fmt.Errorf("vote already in progress")
	}

	instigator := v.Instigator()
	if instigator == nil {
		return fmt.Errorf("vote must have an instigator")
	}

	if cooldown, exists := m.cooldowns[instigator.ID]; exists {
		if time.Now().Before(cooldown) {
			remaining := time.Until(cooldown)
			return fmt.Errorf("please wait %d seconds before starting another vote", int(remaining.Seconds()))
		}
	}

	if err := v.Start(); err != nil {
		return err
	}

	m.activeVote = v
	m.cooldowns[instigator.ID] = time.Now().Add(120 * time.Second)

	go m.runVoteLoop(v)

	return nil
}

func (m *Manager) runVoteLoop(v Vote) {
	timeout := time.NewTimer(120 * time.Second)
	updateTicker := time.NewTicker(30 * time.Second)
	defer timeout.Stop()
	defer updateTicker.Stop()

	for {
		select {
		case <-timeout.C:
			m.mu.Lock()
			if m.activeVote == v {
				v.Timeout()
				m.activeVote = nil
			}
			m.mu.Unlock()
			return

		case <-updateTicker.C:
			m.mu.Lock()
			if m.activeVote == v && v.IsActive() {
				v.Update()
			}
			m.mu.Unlock()

		case <-m.stopChan:
			return
		}
	}
}

func (m *Manager) CastVote(p *player.Player, choice interface{}) error {
	m.mu.RLock()
	v := m.activeVote
	m.mu.RUnlock()

	if v == nil || !v.IsActive() {
		return fmt.Errorf("no active vote")
	}

	return v.CastVote(p, choice)
}

func (m *Manager) CancelVote(p *player.Player) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeVote == nil || !m.activeVote.IsActive() {
		return fmt.Errorf("no active vote to cancel")
	}

	instigator := m.activeVote.Instigator()
	if instigator.ID != p.ID && p.Permissions&uint64(1<<4) == 0 {
		return fmt.Errorf("only the instigator or admins can cancel votes")
	}

	if err := m.activeVote.Cancel(); err != nil {
		return err
	}

	m.activeVote = nil
	return nil
}

func (m *Manager) HandlePlayerDisconnect(playerID uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeVote != nil && m.activeVote.IsActive() {
		instigator := m.activeVote.Instigator()
		if instigator != nil && instigator.ID == playerID {
			m.activeVote.Cancel()
			m.activeVote = nil
		}
	}
}

func (m *Manager) Stop() {
	close(m.stopChan)
}
