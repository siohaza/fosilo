package mapmeta

import (
	"fmt"
	"strings"
)

type Metadata struct {
	Metadata   MetadataInfo   `toml:"metadata"`
	Fog        *Fog           `toml:"fog,omitempty"`
	Rules      *Rules         `toml:"rules,omitempty"`
	Spawns     Spawns         `toml:"spawns"`
	Extensions map[string]any `toml:"extensions,omitempty"`
	Entities   Entities       `toml:"entities"`
}

type MetadataInfo struct {
	Name        string `toml:"name"`
	Version     string `toml:"version,omitempty"`
	Author      string `toml:"author,omitempty"`
	Description string `toml:"description,omitempty"`
}

type Fog struct {
	R uint8 `toml:"r"`
	G uint8 `toml:"g"`
	B uint8 `toml:"b"`
}

type Spawns struct {
	Blue      [][]float64 `toml:"blue,omitempty"`
	Green     [][]float64 `toml:"green,omitempty"`
	BlueArea  []float64   `toml:"blue_area,omitempty"`
	GreenArea []float64   `toml:"green_area,omitempty"`
}

type Rules struct {
	Protected    []string    `toml:"protected,omitempty"`
	Area         []float64   `toml:"area,omitempty"`
	Ball         []float64   `toml:"ball,omitempty"`
	BlueGoal     []float64   `toml:"blue_goal,omitempty"`
	GreenGoal    []float64   `toml:"green_goal,omitempty"`
	PenaltyAreas [][]float64 `toml:"penalty_areas,omitempty"`
}

type Entities struct {
	Blue  EntityLocations `toml:"blue,omitempty"`
	Green EntityLocations `toml:"green,omitempty"`
}

type EntityLocations struct {
	Flag []float64 `toml:"flag,omitempty"`
	Base []float64 `toml:"base,omitempty"`
}

func Parse(content []byte) (*Metadata, error) {
	text := normalizeInput(string(content))
	assignments := scanAssignments(text)

	env := make(map[string]Value, len(assignments))
	for _, a := range assignments {
		val, err := evaluateExpression(a.Expr, env)
		if err != nil {
			continue
		}
		env[a.Name] = val
	}

	meta := buildMetadata(env)
	if meta.Metadata.Name == "" {
		return nil, fmt.Errorf("no name found")
	}

	return meta, nil
}

func normalizeInput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimPrefix(s, "\ufeff")
	return s
}
