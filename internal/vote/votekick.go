package vote

import (
	"fmt"
	"sync"
	"time"

	"github.com/siohaza/fosilo/internal/player"
)

type Votekick struct {
	instigator     *player.Player
	victim         *player.Player
	reason         string
	votes          map[uint8]bool
	startTime      time.Time
	active         bool
	percentage     int
	banDuration    time.Duration
	publicVotes    bool
	mu             sync.RWMutex
	onSuccess      func(*player.Player, string, time.Duration)
	onCancel       func(string)
	onTimeout      func()
	onUpdate       func(string)
	getPlayerCount func() int
}

type VotekickConfig struct {
	Percentage     int
	BanDuration    time.Duration
	PublicVotes    bool
	OnSuccess      func(*player.Player, string, time.Duration)
	OnCancel       func(string)
	OnTimeout      func()
	OnUpdate       func(string)
	GetPlayerCount func() int
}

func NewVotekick(instigator, victim *player.Player, reason string, config VotekickConfig) *Votekick {
	return &Votekick{
		instigator:     instigator,
		victim:         victim,
		reason:         reason,
		votes:          make(map[uint8]bool),
		startTime:      time.Now(),
		active:         false,
		percentage:     config.Percentage,
		banDuration:    config.BanDuration,
		publicVotes:    config.PublicVotes,
		onSuccess:      config.OnSuccess,
		onCancel:       config.OnCancel,
		onTimeout:      config.OnTimeout,
		onUpdate:       config.OnUpdate,
		getPlayerCount: config.GetPlayerCount,
	}
}

func (v *Votekick) Type() VoteType {
	return VoteTypeKick
}

func (v *Votekick) Instigator() *player.Player {
	return v.instigator
}

func (v *Votekick) Start() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.victim == nil {
		return fmt.Errorf("victim is nil")
	}

	if v.instigator.ID == v.victim.ID {
		return fmt.Errorf("you cannot votekick yourself")
	}

	if v.victim.Permissions&uint64(1<<3) != 0 || v.victim.Permissions&uint64(1<<4) != 0 {
		return fmt.Errorf("cannot votekick moderators or admins")
	}

	required := v.getRequiredVotes()
	if required == 0 {
		return fmt.Errorf("not enough players to start a vote")
	}

	v.votes[v.instigator.ID] = true
	v.active = true

	if v.onUpdate != nil {
		msg := fmt.Sprintf("%s started a votekick against %s. Reason: %s",
			v.instigator.Name, v.victim.Name, v.reason)
		v.onUpdate(msg)

		msg = fmt.Sprintf("%d more votes needed (type /y to vote yes)", v.getVotesRemaining())
		v.onUpdate(msg)
	}

	return nil
}

func (v *Votekick) CastVote(p *player.Player, choice interface{}) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.active {
		return fmt.Errorf("vote is not active")
	}

	if p.ID == v.victim.ID {
		return fmt.Errorf("you cannot vote on your own votekick")
	}

	if _, hasVoted := v.votes[p.ID]; hasVoted {
		return fmt.Errorf("you have already voted")
	}

	voteYes, ok := choice.(bool)
	if !ok {
		return fmt.Errorf("invalid vote choice")
	}

	v.votes[p.ID] = voteYes

	if v.publicVotes && v.onUpdate != nil {
		vote := "no"
		if voteYes {
			vote = "yes"
		}
		v.onUpdate(fmt.Sprintf("%s voted %s", p.Name, vote))
	}

	if voteYes && v.getVotesRemaining() == 0 {
		v.succeed()
	} else if v.onUpdate != nil {
		v.onUpdate(fmt.Sprintf("%d more votes needed", v.getVotesRemaining()))
	}

	return nil
}

func (v *Votekick) Cancel() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.active {
		return fmt.Errorf("vote is not active")
	}

	v.active = false

	if v.onCancel != nil {
		v.onCancel("Vote cancelled")
	}

	return nil
}

func (v *Votekick) Update() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.active {
		return false
	}

	if v.onUpdate != nil {
		remaining := v.getVotesRemaining()
		elapsed := time.Since(v.startTime)
		timeLeft := 120*time.Second - elapsed

		msg := fmt.Sprintf("Votekick in progress: %s (Reason: %s)", v.victim.Name, v.reason)
		v.onUpdate(msg)

		msg = fmt.Sprintf("%d more votes needed, %d seconds remaining",
			remaining, int(timeLeft.Seconds()))
		v.onUpdate(msg)
	}

	return true
}

func (v *Votekick) IsActive() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.active
}

func (v *Votekick) GetStatus() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.active {
		return "No active vote"
	}

	yesVotes := 0
	for _, vote := range v.votes {
		if vote {
			yesVotes++
		}
	}

	required := v.getRequiredVotes()
	remaining := v.getVotesRemaining()

	return fmt.Sprintf("Votekick: %s (Reason: %s) - %d/%d votes, %d more needed",
		v.victim.Name, v.reason, yesVotes, required, remaining)
}

func (v *Votekick) Timeout() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.active {
		return
	}

	v.active = false

	if v.onTimeout != nil {
		v.onTimeout()
	}

	if v.onUpdate != nil {
		v.onUpdate("Votekick failed: not enough votes")
	}
}

func (v *Votekick) getRequiredVotes() int {
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

func (v *Votekick) getVotesRemaining() int {
	yesVotes := 0
	for _, vote := range v.votes {
		if vote {
			yesVotes++
		}
	}

	required := v.getRequiredVotes()
	remaining := required - yesVotes
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

func (v *Votekick) succeed() {
	v.active = false

	if v.onSuccess != nil {
		v.onSuccess(v.victim, v.reason, v.banDuration)
	}

	if v.onUpdate != nil {
		if v.banDuration > 0 {
			v.onUpdate(fmt.Sprintf("%s was banned for %s: %s",
				v.victim.Name, v.banDuration, v.reason))
		} else {
			v.onUpdate(fmt.Sprintf("%s was kicked: %s",
				v.victim.Name, v.reason))
		}
	}
}
