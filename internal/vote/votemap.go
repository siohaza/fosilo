package vote

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/siohaza/fosilo/internal/player"
)

type Votemap struct {
	instigator     *player.Player
	mapChoices     []string
	votes          map[uint8]string
	startTime      time.Time
	active         bool
	percentage     int
	allowExtend    bool
	currentMap     string
	mu             sync.RWMutex
	onSuccess      func(string)
	onCancel       func(string)
	onTimeout      func()
	onUpdate       func(string)
	getPlayerCount func() int
	getMapRotation func() []string
	getCurrentMap  func() string
}

type VotemapConfig struct {
	Percentage     int
	AllowExtend    bool
	OnSuccess      func(string)
	OnCancel       func(string)
	OnTimeout      func()
	OnUpdate       func(string)
	GetPlayerCount func() int
	GetMapRotation func() []string
	GetCurrentMap  func() string
}

func NewVotemap(instigator *player.Player, config VotemapConfig) *Votemap {
	v := &Votemap{
		instigator:     instigator,
		votes:          make(map[uint8]string),
		startTime:      time.Now(),
		active:         false,
		percentage:     config.Percentage,
		allowExtend:    config.AllowExtend,
		onSuccess:      config.OnSuccess,
		onCancel:       config.OnCancel,
		onTimeout:      config.OnTimeout,
		onUpdate:       config.OnUpdate,
		getPlayerCount: config.GetPlayerCount,
		getMapRotation: config.GetMapRotation,
		getCurrentMap:  config.GetCurrentMap,
	}

	v.currentMap = config.GetCurrentMap()
	v.mapChoices = v.selectMaps()

	return v
}

func (v *Votemap) Type() VoteType {
	return VoteTypeMap
}

func (v *Votemap) Instigator() *player.Player {
	return v.instigator
}

func (v *Votemap) Start() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	required := v.getRequiredVotes()
	if required == 0 {
		return fmt.Errorf("not enough players to start a vote")
	}

	v.active = true

	if v.onUpdate != nil {
		v.onUpdate(fmt.Sprintf("%s started a map vote", v.instigator.Name))
		v.onUpdate("Available maps:")
		for i, mapName := range v.mapChoices {
			v.onUpdate(fmt.Sprintf("  %d. %s", i+1, mapName))
		}
		v.onUpdate("Vote with /vote <number> or /vote <mapname>")
	}

	return nil
}

func (v *Votemap) CastVote(p *player.Player, choice interface{}) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.active {
		return fmt.Errorf("vote is not active")
	}

	mapChoice, ok := choice.(string)
	if !ok {
		return fmt.Errorf("invalid vote choice")
	}

	validChoice := false
	for _, m := range v.mapChoices {
		if m == mapChoice {
			validChoice = true
			break
		}
	}

	if !validChoice {
		return fmt.Errorf("invalid map choice: %s", mapChoice)
	}

	v.votes[p.ID] = mapChoice

	if v.onUpdate != nil {
		v.onUpdate(fmt.Sprintf("%s voted for %s", p.Name, mapChoice))
	}

	if v.checkMajority(mapChoice) {
		v.succeed(mapChoice)
	}

	return nil
}

func (v *Votemap) Cancel() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.active {
		return fmt.Errorf("vote is not active")
	}

	v.active = false

	if v.onCancel != nil {
		v.onCancel("Map vote cancelled")
	}

	return nil
}

func (v *Votemap) Update() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.active {
		return false
	}

	if v.onUpdate != nil {
		counts := make(map[string]int)
		for _, mapName := range v.votes {
			counts[mapName]++
		}

		elapsed := time.Since(v.startTime)
		timeLeft := 120*time.Second - elapsed

		v.onUpdate("Map vote in progress:")
		for _, mapName := range v.mapChoices {
			count := counts[mapName]
			v.onUpdate(fmt.Sprintf("  %s: %d votes", mapName, count))
		}
		v.onUpdate(fmt.Sprintf("%d seconds remaining", int(timeLeft.Seconds())))
	}

	return true
}

func (v *Votemap) IsActive() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.active
}

func (v *Votemap) GetStatus() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.active {
		return "No active vote"
	}

	counts := make(map[string]int)
	for _, mapName := range v.votes {
		counts[mapName]++
	}

	status := "Map vote:\n"
	for _, mapName := range v.mapChoices {
		count := counts[mapName]
		status += fmt.Sprintf("  %s: %d votes\n", mapName, count)
	}

	return status
}

func (v *Votemap) Timeout() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.active {
		return
	}

	v.active = false

	counts := make(map[string]int)
	for _, mapName := range v.votes {
		counts[mapName]++
	}

	winner := ""
	maxVotes := 0
	for mapName, count := range counts {
		if count > maxVotes {
			maxVotes = count
			winner = mapName
		}
	}

	if winner == "" || maxVotes == 0 {
		if v.onTimeout != nil {
			v.onTimeout()
		}
		if v.onUpdate != nil {
			v.onUpdate("Map vote failed: no votes cast")
		}
		return
	}

	if v.onSuccess != nil {
		v.onSuccess(winner)
	}

	if v.onUpdate != nil {
		v.onUpdate(fmt.Sprintf("Map vote succeeded: %s wins with %d votes", winner, maxVotes))
	}
}

func (v *Votemap) selectMaps() []string {
	rotation := v.getMapRotation()
	if len(rotation) == 0 {
		return []string{}
	}

	choices := make([]string, 0, 5)

	if len(rotation) <= 5 {
		choices = append(choices, rotation...)
	} else {
		available := make([]string, 0, len(rotation))
		for _, m := range rotation {
			if m != v.currentMap {
				available = append(available, m)
			}
		}

		rand.Shuffle(len(available), func(i, j int) {
			available[i], available[j] = available[j], available[i]
		})

		count := 5
		if v.allowExtend {
			count = 4
		}

		if len(available) < count {
			count = len(available)
		}

		choices = append(choices, available[:count]...)
	}

	if v.allowExtend {
		choices = append(choices, "extend")
	}

	return choices
}

func (v *Votemap) getRequiredVotes() int {
	playerCount := v.getPlayerCount()
	if playerCount == 0 {
		return 0
	}
	required := (playerCount * v.percentage) / 100
	if required == 0 && playerCount > 0 {
		required = 1
	}
	return required
}

func (v *Votemap) checkMajority(mapChoice string) bool {
	count := 0
	for _, m := range v.votes {
		if m == mapChoice {
			count++
		}
	}

	required := v.getRequiredVotes()
	return count >= required
}

func (v *Votemap) succeed(winner string) {
	v.active = false

	if v.onSuccess != nil {
		v.onSuccess(winner)
	}

	if v.onUpdate != nil {
		if winner == "extend" {
			v.onUpdate("Map extended by 15 minutes")
		} else {
			v.onUpdate(fmt.Sprintf("Map changed to %s", winner))
		}
	}
}

func (v *Votemap) GetMapChoices() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.mapChoices
}
