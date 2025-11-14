package bans

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type BanType string

const (
	BanTypeIP       BanType = "ip"
	BanTypeUsername BanType = "username"
)

type Ban struct {
	Type      BanType   `json:"type"`
	IP        string    `json:"ip,omitempty"`
	Name      string    `json:"name"`
	Reason    string    `json:"reason"`
	BannedBy  string    `json:"banned_by"`
	BannedAt  time.Time `json:"banned_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Permanent bool      `json:"permanent"`
}

type Manager struct {
	ipBans       map[string]*Ban
	usernameBans map[string]*Ban
	filePath     string
	mu           sync.RWMutex
}

func NewManager(filePath string) *Manager {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Warning: failed to create bans directory: %v\n", err)
	}

	return &Manager{
		ipBans:       make(map[string]*Ban),
		usernameBans: make(map[string]*Ban),
		filePath:     filePath,
	}
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read bans file: %w", err)
	}

	var bans []*Ban
	if err := json.Unmarshal(data, &bans); err != nil {
		return fmt.Errorf("failed to parse bans file: %w", err)
	}

	m.ipBans = make(map[string]*Ban)
	m.usernameBans = make(map[string]*Ban)
	for _, ban := range bans {
		if !ban.Permanent && time.Now().After(ban.ExpiresAt) {
			continue
		}

		if ban.Type == "" {
			ban.Type = BanTypeIP
		}

		switch ban.Type {
		case BanTypeIP:
			if ban.IP != "" {
				m.ipBans[ban.IP] = ban
			}
		case BanTypeUsername:
			if ban.Name != "" {
				m.usernameBans[ban.Name] = ban
			}
		}
	}

	return nil
}

func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bans := make([]*Ban, 0, len(m.ipBans)+len(m.usernameBans))
	for _, ban := range m.ipBans {
		bans = append(bans, ban)
	}
	for _, ban := range m.usernameBans {
		bans = append(bans, ban)
	}

	data, err := json.MarshalIndent(bans, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bans: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write bans file: %w", err)
	}

	return nil
}

func (m *Manager) IsBanned(ip string) (bool, *Ban) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ban, exists := m.ipBans[ip]
	if !exists {
		return false, nil
	}

	if !ban.Permanent && time.Now().After(ban.ExpiresAt) {
		return false, nil
	}

	return true, ban
}

func (m *Manager) IsBannedByName(name string) (bool, *Ban) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ban, exists := m.usernameBans[name]
	if !exists {
		return false, nil
	}

	if !ban.Permanent && time.Now().After(ban.ExpiresAt) {
		return false, nil
	}

	return true, ban
}

func (m *Manager) AddBan(ip, name, reason, bannedBy string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ban := &Ban{
		Type:      BanTypeIP,
		IP:        ip,
		Name:      name,
		Reason:    reason,
		BannedBy:  bannedBy,
		BannedAt:  time.Now(),
		Permanent: duration == 0,
	}

	if duration > 0 {
		ban.ExpiresAt = time.Now().Add(duration)
	}

	m.ipBans[ip] = ban

	return m.saveUnlocked()
}

func (m *Manager) AddBanByName(name, reason, bannedBy string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ban := &Ban{
		Type:      BanTypeUsername,
		Name:      name,
		Reason:    reason,
		BannedBy:  bannedBy,
		BannedAt:  time.Now(),
		Permanent: duration == 0,
	}

	if duration > 0 {
		ban.ExpiresAt = time.Now().Add(duration)
	}

	m.usernameBans[name] = ban

	return m.saveUnlocked()
}

func (m *Manager) RemoveBan(ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.ipBans, ip)

	return m.saveUnlocked()
}

func (m *Manager) RemoveBanByName(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.usernameBans, name)

	return m.saveUnlocked()
}

func (m *Manager) saveUnlocked() error {
	bans := make([]*Ban, 0, len(m.ipBans)+len(m.usernameBans))
	for _, ban := range m.ipBans {
		bans = append(bans, ban)
	}
	for _, ban := range m.usernameBans {
		bans = append(bans, ban)
	}

	data, err := json.MarshalIndent(bans, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bans: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write bans file: %w", err)
	}

	return nil
}

func (m *Manager) GetAll() []*Ban {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bans := make([]*Ban, 0, len(m.ipBans)+len(m.usernameBans))
	for _, ban := range m.ipBans {
		if !ban.Permanent && time.Now().After(ban.ExpiresAt) {
			continue
		}
		bans = append(bans, ban)
	}
	for _, ban := range m.usernameBans {
		if !ban.Permanent && time.Now().After(ban.ExpiresAt) {
			continue
		}
		bans = append(bans, ban)
	}

	return bans
}

func (m *Manager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for ip, ban := range m.ipBans {
		if !ban.Permanent && now.After(ban.ExpiresAt) {
			delete(m.ipBans, ip)
		}
	}
	for name, ban := range m.usernameBans {
		if !ban.Permanent && now.After(ban.ExpiresAt) {
			delete(m.usernameBans, name)
		}
	}

	return m.saveUnlocked()
}
