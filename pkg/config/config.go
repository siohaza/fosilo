package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server    ServerConfig
	Teams     TeamsConfig
	Passwords PasswordsConfig
	RateLimit RateLimitConfig
	Voting    VotingConfig
	Gamemode  GamemodeConfig `toml:"gamemode"`
}

type MasterHost struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type ServerConfig struct {
	Name             string       `toml:"name"`
	Port             int          `toml:"port"`
	Gamemode         int          `toml:"gamemode"`
	CaptureLimit     int          `toml:"capture_limit"`
	Master           bool         `toml:"master"`
	MasterHosts      []MasterHost `toml:"master_hosts"`
	Maps             []string     `toml:"maps"`
	WelcomeMessages  []string     `toml:"welcome_messages"`
	PeriodicMessages []string     `toml:"periodic_messages"`
	MaxPlayers       int          `toml:"max_players"`
	RespawnTime      int          `toml:"respawn_time"`

	// logging configuration
	LogToFile bool `toml:"log_to_file"`

	// ctf specific
	CaptureTimeBonus float64 `toml:"capture_time_bonus"`
	FlagReturnTime   float64 `toml:"flag_return_time"`

	// tdm specific
	KillLimit          int     `toml:"kill_limit"`
	IntelPoints        int     `toml:"intel_points"`
	RemoveIntel        bool    `toml:"remove_intel"`
	HeadshotMultiplier float64 `toml:"headshot_multiplier"`
	EnableKillstreaks  bool    `toml:"enable_killstreaks"`

	// babel specific
	BabelReverse      bool    `toml:"babel_reverse"`
	BabelCaptureLimit int     `toml:"babel_capture_limit"`
	RegenerateTower   bool    `toml:"regenerate_tower"`
	RegenerationRate  float64 `toml:"regeneration_rate"`

	// arena specific
	ArenaScoreLimit    int  `toml:"arena_score_limit"`
	ArenaTimeoutIsDraw bool `toml:"timeout_is_draw"`
	ArenaSuddenDeath   bool `toml:"sudden_death_enabled"`

	// tc specific
	TCMaxScore        int     `toml:"tc_max_score"`
	TCCaptureDistance float64 `toml:"tc_capture_distance"`
	TCCaptureRate     float64 `toml:"tc_capture_rate"`
}

type TeamsConfig struct {
	Team1 TeamInfo `toml:"team1"`
	Team2 TeamInfo `toml:"team2"`
}

type TeamInfo struct {
	Name  string `toml:"name"`
	Color [3]int `toml:"color"`
}

type PasswordsConfig struct {
	Manager   string `toml:"manager"`
	Admin     string `toml:"admin"`
	Moderator string `toml:"moderator"`
	Guard     string `toml:"guard"`
	Trusted   string `toml:"trusted"`
}

type RateLimitConfig struct {
	Enabled               bool `toml:"enabled"`
	PacketsPerSecond      int  `toml:"packets_per_second"`
	BurstSize             int  `toml:"burst_size"`
	PositionPacketsPerSec int  `toml:"position_packets_per_sec"`
	OrientPacketsPerSec   int  `toml:"orient_packets_per_sec"`
	BlockPacketsPerSec    int  `toml:"block_packets_per_sec"`
}

type VotingConfig struct {
	VotekickEnabled     bool `toml:"votekick_enabled"`
	VotekickPercentage  int  `toml:"votekick_percentage"`
	VotekickBanDuration int  `toml:"votekick_ban_duration"`
	VoteCooldown        int  `toml:"vote_cooldown"`
	VoteTimeout         int  `toml:"vote_timeout"`
	VotemapEnabled      bool `toml:"votemap_enabled"`
	VotemapPercentage   int  `toml:"votemap_percentage"`
	VotemapChoices      int  `toml:"votemap_choices"`
	VotemapAllowExtend  bool `toml:"votemap_allow_extend"`
}

type GamemodeConfig struct {
	CTF   *CTFConfig   `toml:"ctf"`
	TC    *TCConfig    `toml:"tc"`
	TDM   *TDMConfig   `toml:"tdm"`
	Babel *BabelConfig `toml:"babel"`
	Arena *ArenaConfig `toml:"arena"`
}

type CTFConfig struct {
	CaptureLimit     *int     `toml:"capture_limit"`
	CaptureTimeBonus *float64 `toml:"capture_time_bonus"`
	FlagReturnTime   *float64 `toml:"flag_return_time"`
}

type TCConfig struct {
	MaxScore        *int     `toml:"max_score"`
	CaptureDistance *float64 `toml:"capture_distance"`
	CaptureRate     *float64 `toml:"capture_rate"`
}

type TDMConfig struct {
	KillLimit          *int     `toml:"kill_limit"`
	IntelPoints        *int     `toml:"intel_points"`
	RemoveIntel        *bool    `toml:"remove_intel"`
	HeadshotMultiplier *float64 `toml:"headshot_multiplier"`
	EnableKillstreaks  *bool    `toml:"enable_killstreaks"`
}

type BabelConfig struct {
	Reverse          *bool    `toml:"reverse"`
	CaptureLimit     *int     `toml:"capture_limit"`
	RegenerateTower  *bool    `toml:"regenerate_tower"`
	RegenerationRate *float64 `toml:"regeneration_rate"`
}

type ArenaConfig struct {
	ScoreLimit         *int  `toml:"score_limit"`
	TimeoutIsDraw      *bool `toml:"timeout_is_draw"`
	SuddenDeathEnabled *bool `toml:"sudden_death_enabled"`
}

type MapConfig struct {
	Map          MapInfo           `toml:"map"`
	SpawnPoints  SpawnPointsConfig `toml:"spawnpoints"`
	Water        WaterConfig       `toml:"water"`
	Intel        IntelConfig       `toml:"intel"`
	Tents        TentsConfig       `toml:"tents"`
	Extensions   MapExtensions     `toml:"extensions,omitempty"`
	Protected    []string          `toml:"protected,omitempty"`
	Area         []float64         `toml:"area,omitempty"`
	Ball         []float64         `toml:"ball,omitempty"`
	BlueGoal     []float64         `toml:"blue_goal,omitempty"`
	GreenGoal    []float64         `toml:"green_goal,omitempty"`
	PenaltyAreas [][]float64       `toml:"penalty_areas,omitempty"`
}

type MapInfo struct {
	Author      string `toml:"author"`
	Description string `toml:"description"`
	FogColor    [3]int `toml:"fog_color"`
}

type WaterConfig struct {
	Enabled bool    `toml:"enabled"`
	Damage  int     `toml:"damage"`
	Level   float32 `toml:"level"`
}

type IntelConfig struct {
	Team1Position [3]float64 `toml:"team1_position"`
	Team2Position [3]float64 `toml:"team2_position"`
	Team1Base     [3]float64 `toml:"team1_base"`
	Team2Base     [3]float64 `toml:"team2_base"`
}

type SpawnPointsConfig struct {
	Team1       SpawnArea   `toml:"team1"`
	Team2       SpawnArea   `toml:"team2"`
	Team1Points [][]float64 `toml:"team1_points,omitempty"`
	Team2Points [][]float64 `toml:"team2_points,omitempty"`
}

type SpawnArea struct {
	Start [3]int `toml:"start"`
	End   [3]int `toml:"end"`
}

type MapExtensions struct {
	WaterDamage      *int
	BoundaryDamage   *BoundaryDamage
	TimeLimit        *int
	CapLimit         *int
	DisabledCommands []string

	Babel        *bool
	Push         *bool
	Arena        *bool
	TDM          *bool
	TC           *bool
	Infiltration *bool
	Murderball   *bool
	Boss         *bool

	PushSpawnRange *int
	PushBlueSpawn  []float64
	PushBlueCP     []float64
	PushGreenSpawn []float64
	PushGreenCP    []float64

	Extras map[string]any
}

type BoundaryDamage struct {
	Left   int `toml:"left"`
	Right  int `toml:"right"`
	Top    int `toml:"top"`
	Bottom int `toml:"bottom"`
	Damage int `toml:"damage"`
}

func (m *MapExtensions) UnmarshalTOML(data interface{}) error {
	if data == nil {
		return nil
	}

	table, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("extensions must be a table")
	}

	for key, raw := range table {
		switch key {
		case "water_damage":
			if val, ok := toInt(raw); ok {
				m.WaterDamage = &val
			} else {
				m.addExtra(key, raw)
			}
		case "boundary_damage":
			if bd, ok := toBoundary(raw); ok {
				m.BoundaryDamage = bd
			} else {
				m.addExtra(key, raw)
			}
		case "time_limit":
			if val, ok := toInt(raw); ok {
				m.TimeLimit = &val
			} else {
				m.addExtra(key, raw)
			}
		case "cap_limit":
			if val, ok := toInt(raw); ok {
				m.CapLimit = &val
			} else {
				m.addExtra(key, raw)
			}
		case "disabled_commands":
			if list, ok := toStringSlice(raw); ok {
				m.DisabledCommands = list
			} else {
				m.addExtra(key, raw)
			}
		case "babel":
			if val, ok := toBool(raw); ok {
				m.Babel = &val
			} else {
				m.addExtra(key, raw)
			}
		case "push":
			if val, ok := toBool(raw); ok {
				m.Push = &val
			} else {
				m.addExtra(key, raw)
			}
		case "arena":
			if val, ok := toBool(raw); ok {
				m.Arena = &val
			} else {
				m.addExtra(key, raw)
			}
		case "tdm":
			if val, ok := toBool(raw); ok {
				m.TDM = &val
			} else {
				m.addExtra(key, raw)
			}
		case "tc":
			if val, ok := toBool(raw); ok {
				m.TC = &val
			} else {
				m.addExtra(key, raw)
			}
		case "infiltration":
			if val, ok := toBool(raw); ok {
				m.Infiltration = &val
			} else {
				m.addExtra(key, raw)
			}
		case "murderball":
			if val, ok := toBool(raw); ok {
				m.Murderball = &val
			} else {
				m.addExtra(key, raw)
			}
		case "boss":
			if val, ok := toBool(raw); ok {
				m.Boss = &val
			} else {
				m.addExtra(key, raw)
			}
		case "push_spawn_range":
			if val, ok := toInt(raw); ok {
				m.PushSpawnRange = &val
			} else {
				m.addExtra(key, raw)
			}
		case "push_blue_spawn":
			if vals, ok := toFloatSlice(raw, 3); ok {
				m.PushBlueSpawn = vals
			} else {
				m.addExtra(key, raw)
			}
		case "push_blue_cp":
			if vals, ok := toFloatSlice(raw, 3); ok {
				m.PushBlueCP = vals
			} else {
				m.addExtra(key, raw)
			}
		case "push_green_spawn":
			if vals, ok := toFloatSlice(raw, 3); ok {
				m.PushGreenSpawn = vals
			} else {
				m.addExtra(key, raw)
			}
		case "push_green_cp":
			if vals, ok := toFloatSlice(raw, 3); ok {
				m.PushGreenCP = vals
			} else {
				m.addExtra(key, raw)
			}
		default:
			m.addExtra(key, raw)
		}
	}

	return nil
}

func (m *MapExtensions) addExtra(key string, value interface{}) {
	if m.Extras == nil {
		m.Extras = make(map[string]any)
	}
	m.Extras[key] = value
}

func toInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func toBool(value interface{}) (bool, bool) {
	b, ok := value.(bool)
	return b, ok
}

func toStringSlice(value interface{}) ([]string, bool) {
	list, ok := value.([]interface{})
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		str, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, str)
	}
	return result, true
}

func toFloatSlice(value interface{}, expected int) ([]float64, bool) {
	list, ok := value.([]interface{})
	if !ok {
		return nil, false
	}
	if expected > 0 && len(list) != expected {
		return nil, false
	}

	result := make([]float64, 0, len(list))
	for _, item := range list {
		switch v := item.(type) {
		case int64:
			result = append(result, float64(v))
		case float64:
			result = append(result, v)
		default:
			return nil, false
		}
	}

	return result, true
}

func toBoundary(value interface{}) (*BoundaryDamage, bool) {
	table, ok := value.(map[string]interface{})
	if !ok {
		return nil, false
	}

	left, lok := toInt(table["left"])
	right, rok := toInt(table["right"])
	top, tok := toInt(table["top"])
	bottom, bok := toInt(table["bottom"])
	damage, dok := toInt(table["damage"])
	if !(lok && rok && tok && bok && dok) {
		return nil, false
	}

	return &BoundaryDamage{
		Left:   left,
		Right:  right,
		Top:    top,
		Bottom: bottom,
		Damage: damage,
	}, true
}

func firstNonEmptyStrings(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func firstNonEmptyFloats(primary, fallback []float64) []float64 {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func firstNonEmptyMatrix(primary, fallback [][]float64) [][]float64 {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

type TentsConfig struct {
	Team1 TentArea `toml:"team1"`
	Team2 TentArea `toml:"team2"`
}

type TentArea struct {
	Start [3]int `toml:"start"`
	End   [3]int `toml:"end"`
}

func LoadConfig(path string) (*Config, error) {
	var config Config

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.applyGamemodeOverrides()

	if config.Server.Port == 0 {
		config.Server.Port = 32887
	}

	if config.Server.MaxPlayers == 0 {
		config.Server.MaxPlayers = 32
	}

	if config.Server.RespawnTime == 0 {
		config.Server.RespawnTime = 5
	}

	if config.Server.CaptureLimit == 0 {
		config.Server.CaptureLimit = 10
	}

	// ctf defaults
	if config.Server.Gamemode == 0 {
		if config.Server.FlagReturnTime == 0 {
			config.Server.FlagReturnTime = 30.0
		}
	}

	// tc defaults
	if config.Server.Gamemode == 1 {
		if config.Server.TCMaxScore == 0 {
			config.Server.TCMaxScore = 10
		}
		if config.Server.TCCaptureDistance == 0 {
			config.Server.TCCaptureDistance = 16.0
		}
		if config.Server.TCCaptureRate == 0 {
			config.Server.TCCaptureRate = 0.05
		}
	}

	// babel defaults
	if config.Server.Gamemode == 2 {
		if config.Server.BabelCaptureLimit == 0 {
			config.Server.BabelCaptureLimit = 10
		}
		if config.Server.RegenerationRate == 0 {
			config.Server.RegenerationRate = 1.0
		}
	}

	// tdm defaults
	if config.Server.Gamemode == 3 {
		if config.Server.KillLimit == 0 {
			config.Server.KillLimit = 100
		}
		if config.Server.IntelPoints == 0 {
			config.Server.IntelPoints = 10
		}
	}

	// arena defaults
	if config.Server.Gamemode == 4 {
		if config.Server.ArenaScoreLimit == 0 {
			config.Server.ArenaScoreLimit = 5
		}
	}

	if config.Server.Master && len(config.Server.MasterHosts) == 0 {
		config.Server.MasterHosts = []MasterHost{
			{Host: "master.buildandshoot.com", Port: 32886},
			{Host: "master1.aos.coffee", Port: 32886},
			{Host: "master2.aos.coffee", Port: 32886},
		}
	}

	// rate limit defaults
	if config.RateLimit.PacketsPerSecond == 0 {
		config.RateLimit.PacketsPerSecond = 100
	}
	if config.RateLimit.BurstSize == 0 {
		config.RateLimit.BurstSize = 150
	}
	if config.RateLimit.PositionPacketsPerSec == 0 {
		config.RateLimit.PositionPacketsPerSec = 60
	}
	if config.RateLimit.OrientPacketsPerSec == 0 {
		config.RateLimit.OrientPacketsPerSec = 60
	}
	if config.RateLimit.BlockPacketsPerSec == 0 {
		config.RateLimit.BlockPacketsPerSec = 30
	}

	// voting defaults
	if config.Voting.VotekickPercentage == 0 {
		config.Voting.VotekickPercentage = 35
	}
	if config.Voting.VotekickBanDuration == 0 {
		config.Voting.VotekickBanDuration = 30
	}
	if config.Voting.VoteCooldown == 0 {
		config.Voting.VoteCooldown = 120
	}
	if config.Voting.VoteTimeout == 0 {
		config.Voting.VoteTimeout = 120
	}
	if config.Voting.VotemapPercentage == 0 {
		config.Voting.VotemapPercentage = 80
	}
	if config.Voting.VotemapChoices == 0 {
		config.Voting.VotemapChoices = 5
	}

	return &config, nil
}

func (c *Config) applyGamemodeOverrides() {
	if c.Gamemode.CTF != nil {
		if c.Gamemode.CTF.CaptureLimit != nil {
			c.Server.CaptureLimit = *c.Gamemode.CTF.CaptureLimit
		}
		if c.Gamemode.CTF.CaptureTimeBonus != nil {
			c.Server.CaptureTimeBonus = *c.Gamemode.CTF.CaptureTimeBonus
		}
		if c.Gamemode.CTF.FlagReturnTime != nil {
			c.Server.FlagReturnTime = *c.Gamemode.CTF.FlagReturnTime
		}
	}

	if c.Gamemode.TDM != nil {
		if c.Gamemode.TDM.KillLimit != nil {
			c.Server.KillLimit = *c.Gamemode.TDM.KillLimit
		}
		if c.Gamemode.TDM.IntelPoints != nil {
			c.Server.IntelPoints = *c.Gamemode.TDM.IntelPoints
		}
		if c.Gamemode.TDM.RemoveIntel != nil {
			c.Server.RemoveIntel = *c.Gamemode.TDM.RemoveIntel
		}
		if c.Gamemode.TDM.HeadshotMultiplier != nil {
			c.Server.HeadshotMultiplier = *c.Gamemode.TDM.HeadshotMultiplier
		}
		if c.Gamemode.TDM.EnableKillstreaks != nil {
			c.Server.EnableKillstreaks = *c.Gamemode.TDM.EnableKillstreaks
		}
	}

	if c.Gamemode.Babel != nil {
		if c.Gamemode.Babel.Reverse != nil {
			c.Server.BabelReverse = *c.Gamemode.Babel.Reverse
		}
		if c.Gamemode.Babel.CaptureLimit != nil {
			c.Server.BabelCaptureLimit = *c.Gamemode.Babel.CaptureLimit
		}
		if c.Gamemode.Babel.RegenerateTower != nil {
			c.Server.RegenerateTower = *c.Gamemode.Babel.RegenerateTower
		}
		if c.Gamemode.Babel.RegenerationRate != nil {
			c.Server.RegenerationRate = *c.Gamemode.Babel.RegenerationRate
		}
	}

	if c.Gamemode.Arena != nil {
		if c.Gamemode.Arena.ScoreLimit != nil {
			c.Server.ArenaScoreLimit = *c.Gamemode.Arena.ScoreLimit
		}
		if c.Gamemode.Arena.TimeoutIsDraw != nil {
			c.Server.ArenaTimeoutIsDraw = *c.Gamemode.Arena.TimeoutIsDraw
		}
		if c.Gamemode.Arena.SuddenDeathEnabled != nil {
			c.Server.ArenaSuddenDeath = *c.Gamemode.Arena.SuddenDeathEnabled
		}
	}

	if c.Gamemode.TC != nil {
		if c.Gamemode.TC.MaxScore != nil {
			c.Server.TCMaxScore = *c.Gamemode.TC.MaxScore
		}
		if c.Gamemode.TC.CaptureDistance != nil {
			c.Server.TCCaptureDistance = *c.Gamemode.TC.CaptureDistance
		}
		if c.Gamemode.TC.CaptureRate != nil {
			c.Server.TCCaptureRate = *c.Gamemode.TC.CaptureRate
		}
	}
}

func LoadMapConfig(path string) (*MapConfig, error) {
	tomlPath := path[:len(path)-4] + ".toml"

	if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
		fmt.Printf("WARNING: No map metadata found for %s, using defaults\n", path)
		return getDefaultMapConfig(), nil
	}

	return LoadMapConfigToml(tomlPath)
}

func getDefaultMapConfig() *MapConfig {
	config := &MapConfig{
		Map: MapInfo{
			FogColor: [3]int{128, 232, 255},
		},
		SpawnPoints: SpawnPointsConfig{
			Team1: SpawnArea{
				Start: [3]int{0, 0, 0},
				End:   [3]int{256, 256, 63},
			},
			Team2: SpawnArea{
				Start: [3]int{256, 256, 0},
				End:   [3]int{511, 511, 63},
			},
		},
		Water: WaterConfig{
			Enabled: false,
			Level:   63,
			Damage:  1,
		},
		Intel: IntelConfig{
			Team1Position: [3]float64{128, 128, 60},
			Team2Position: [3]float64{384, 384, 60},
			Team1Base:     [3]float64{128, 128, 60},
			Team2Base:     [3]float64{384, 384, 60},
		},
	}

	applyMapConfigDefaults(config)
	return config
}

// returns a fresh map configuration populated with default values
func DefaultMapConfig() *MapConfig {
	return getDefaultMapConfig()
}

func LoadMapConfigToml(path string) (*MapConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read toml file: %w", err)
	}

	type tomlMetadata struct {
		Metadata struct {
			Name        string `toml:"name"`
			Version     string `toml:"version"`
			Author      string `toml:"author"`
			Description string `toml:"description"`
		} `toml:"metadata"`
		Fog *struct {
			R uint8 `toml:"r"`
			G uint8 `toml:"g"`
			B uint8 `toml:"b"`
		} `toml:"fog"`
		Extensions MapExtensions `toml:"extensions"`
		Rules      struct {
			Protected    []string    `toml:"protected"`
			Area         []float64   `toml:"area"`
			Ball         []float64   `toml:"ball"`
			BlueGoal     []float64   `toml:"blue_goal"`
			GreenGoal    []float64   `toml:"green_goal"`
			PenaltyAreas [][]float64 `toml:"penalty_areas"`
		} `toml:"rules"`
		Protected    []string    `toml:"protected"`
		Area         []float64   `toml:"area"`
		Ball         []float64   `toml:"ball"`
		BlueGoal     []float64   `toml:"blue_goal"`
		GreenGoal    []float64   `toml:"green_goal"`
		PenaltyAreas [][]float64 `toml:"penalty_areas"`
		Spawns       struct {
			Blue      [][]float64 `toml:"blue"`
			Green     [][]float64 `toml:"green"`
			BlueArea  []float64   `toml:"blue_area"`
			GreenArea []float64   `toml:"green_area"`
		} `toml:"spawns"`
		Entities struct {
			Blue struct {
				Flag []float64 `toml:"flag"`
				Base []float64 `toml:"base"`
			} `toml:"blue"`
			Green struct {
				Flag []float64 `toml:"flag"`
				Base []float64 `toml:"base"`
			} `toml:"green"`
		} `toml:"entities"`
	}

	var meta tomlMetadata
	if err := toml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse toml: %w", err)
	}

	config := &MapConfig{}

	config.Map.Author = meta.Metadata.Author
	config.Map.Description = meta.Metadata.Description

	if meta.Fog != nil {
		config.Map.FogColor = [3]int{int(meta.Fog.R), int(meta.Fog.G), int(meta.Fog.B)}
	}

	config.SpawnPoints.Team1Points = meta.Spawns.Blue
	config.SpawnPoints.Team2Points = meta.Spawns.Green

	if len(meta.Spawns.BlueArea) == 4 {
		config.SpawnPoints.Team1.Start = [3]int{int(meta.Spawns.BlueArea[0]), int(meta.Spawns.BlueArea[1]), 0}
		config.SpawnPoints.Team1.End = [3]int{int(meta.Spawns.BlueArea[2]), int(meta.Spawns.BlueArea[3]), 63}
	}
	if len(meta.Spawns.GreenArea) == 4 {
		config.SpawnPoints.Team2.Start = [3]int{int(meta.Spawns.GreenArea[0]), int(meta.Spawns.GreenArea[1]), 0}
		config.SpawnPoints.Team2.End = [3]int{int(meta.Spawns.GreenArea[2]), int(meta.Spawns.GreenArea[3]), 63}
	}

	config.Protected = firstNonEmptyStrings(meta.Protected, meta.Rules.Protected)
	config.Area = firstNonEmptyFloats(meta.Area, meta.Rules.Area)
	config.Ball = firstNonEmptyFloats(meta.Ball, meta.Rules.Ball)
	config.BlueGoal = firstNonEmptyFloats(meta.BlueGoal, meta.Rules.BlueGoal)
	config.GreenGoal = firstNonEmptyFloats(meta.GreenGoal, meta.Rules.GreenGoal)
	config.PenaltyAreas = firstNonEmptyMatrix(meta.PenaltyAreas, meta.Rules.PenaltyAreas)

	if len(meta.Entities.Blue.Flag) == 3 {
		config.Intel.Team1Position = [3]float64{
			meta.Entities.Blue.Flag[0],
			meta.Entities.Blue.Flag[1],
			meta.Entities.Blue.Flag[2],
		}
	}

	if len(meta.Entities.Blue.Base) == 3 {
		config.Intel.Team1Base = [3]float64{
			meta.Entities.Blue.Base[0],
			meta.Entities.Blue.Base[1],
			meta.Entities.Blue.Base[2],
		}
	}

	if len(meta.Entities.Green.Flag) == 3 {
		config.Intel.Team2Position = [3]float64{
			meta.Entities.Green.Flag[0],
			meta.Entities.Green.Flag[1],
			meta.Entities.Green.Flag[2],
		}
	}

	if len(meta.Entities.Green.Base) == 3 {
		config.Intel.Team2Base = [3]float64{
			meta.Entities.Green.Base[0],
			meta.Entities.Green.Base[1],
			meta.Entities.Green.Base[2],
		}
	}

	config.Extensions.WaterDamage = meta.Extensions.WaterDamage
	config.Extensions.BoundaryDamage = meta.Extensions.BoundaryDamage
	config.Extensions.TimeLimit = meta.Extensions.TimeLimit
	config.Extensions.CapLimit = meta.Extensions.CapLimit
	config.Extensions.DisabledCommands = meta.Extensions.DisabledCommands

	config.Extensions.Babel = meta.Extensions.Babel
	config.Extensions.Push = meta.Extensions.Push
	config.Extensions.Arena = meta.Extensions.Arena
	config.Extensions.TDM = meta.Extensions.TDM
	config.Extensions.TC = meta.Extensions.TC
	config.Extensions.Infiltration = meta.Extensions.Infiltration
	config.Extensions.Murderball = meta.Extensions.Murderball
	config.Extensions.Boss = meta.Extensions.Boss

	config.Extensions.PushSpawnRange = meta.Extensions.PushSpawnRange
	config.Extensions.PushBlueSpawn = meta.Extensions.PushBlueSpawn
	config.Extensions.PushBlueCP = meta.Extensions.PushBlueCP
	config.Extensions.PushGreenSpawn = meta.Extensions.PushGreenSpawn
	config.Extensions.PushGreenCP = meta.Extensions.PushGreenCP
	config.Extensions.Extras = meta.Extensions.Extras

	if meta.Extensions.WaterDamage != nil {
		config.Water.Enabled = true
		config.Water.Damage = *meta.Extensions.WaterDamage
	}

	applyMapConfigDefaults(config)

	return config, nil
}

func applyMapConfigDefaults(config *MapConfig) {
	if config.Water.Level == 0 {
		config.Water.Level = 63
	}
	if config.Water.Damage == 0 {
		config.Water.Damage = 1
	}

	if config.Intel.Team1Position[0] == 0 && config.Intel.Team1Position[1] == 0 {
		config.Intel.Team1Position[0] = float64(config.SpawnPoints.Team1.Start[0]+config.SpawnPoints.Team1.End[0]) / 2
		config.Intel.Team1Position[1] = float64(config.SpawnPoints.Team1.Start[1]+config.SpawnPoints.Team1.End[1]) / 2
		config.Intel.Team1Position[2] = float64(config.SpawnPoints.Team1.Start[2])
	}

	if config.Intel.Team2Position[0] == 0 && config.Intel.Team2Position[1] == 0 {
		config.Intel.Team2Position[0] = float64(config.SpawnPoints.Team2.Start[0]+config.SpawnPoints.Team2.End[0]) / 2
		config.Intel.Team2Position[1] = float64(config.SpawnPoints.Team2.Start[1]+config.SpawnPoints.Team2.End[1]) / 2
		config.Intel.Team2Position[2] = float64(config.SpawnPoints.Team2.Start[2])
	}

	if config.Intel.Team1Base[0] == 0 && config.Intel.Team1Base[1] == 0 {
		config.Intel.Team1Base = config.Intel.Team1Position
	}

	if config.Intel.Team2Base[0] == 0 && config.Intel.Team2Base[1] == 0 {
		config.Intel.Team2Base = config.Intel.Team2Position
	}

	if config.Tents.Team1.Start[0] == 0 && config.Tents.Team1.End[0] == 0 {
		config.Tents.Team1 = TentArea{
			Start: config.SpawnPoints.Team1.Start,
			End:   config.SpawnPoints.Team1.End,
		}
	}

	if config.Tents.Team2.Start[0] == 0 && config.Tents.Team2.End[0] == 0 {
		config.Tents.Team2 = TentArea{
			Start: config.SpawnPoints.Team2.Start,
			End:   config.SpawnPoints.Team2.End,
		}
	}
}

func (c *Config) Validate() error {
	if c.Server.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Server.MaxPlayers <= 0 || c.Server.MaxPlayers > 32 {
		return fmt.Errorf("max_players must be between 1 and 32")
	}

	if len(c.Server.Maps) == 0 {
		return fmt.Errorf("at least one map must be specified")
	}

	if c.Teams.Team1.Name == "" || c.Teams.Team2.Name == "" {
		return fmt.Errorf("team names cannot be empty")
	}

	return nil
}

type GamemodeID int

const (
	GamemodeCTF   GamemodeID = 0
	GamemodeTC    GamemodeID = 1
	GamemodeBabel GamemodeID = 2
	GamemodeTDM   GamemodeID = 3
	GamemodeArena GamemodeID = 4
)

func (g GamemodeID) String() string {
	switch g {
	case GamemodeCTF:
		return "ctf"
	case GamemodeTC:
		return "tc"
	case GamemodeBabel:
		return "babel"
	case GamemodeTDM:
		return "tdm"
	case GamemodeArena:
		return "arena"
	default:
		return "unknown"
	}
}

func ParseGamemode(id int) (GamemodeID, error) {
	switch id {
	case 0:
		return GamemodeCTF, nil
	case 1:
		return GamemodeTC, nil
	case 2:
		return GamemodeBabel, nil
	case 3:
		return GamemodeTDM, nil
	case 4:
		return GamemodeArena, nil
	default:
		return 0, fmt.Errorf("invalid gamemode ID: %d", id)
	}
}
