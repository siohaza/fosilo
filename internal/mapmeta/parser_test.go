package mapmeta

import (
	"testing"
)

func TestParseMetadataExpressions(t *testing.T) {
	source := `
name = 'Test Map'
version = "1.2"
author = 'Author'
description = ('A multi-line description')
fog = (128, 200, 64)
murderball = True

extensions = {
    'water_damage': 100,
    'push_blue_spawn': (10, 20, 30)
}

AREA = (156, 190, 354, 322)
PAD = 18
WIDTH = 18
BLUE_RECT = (AREA[0] + PAD, AREA[1], AREA[0] + PAD + WIDTH, AREA[3])
GREEN_RECT = (AREA[2] - PAD - WIDTH, AREA[1], AREA[2] - PAD, AREA[3])
area = AREA
ball = (AREA[0] + (AREA[2] - AREA[0]) / 2, AREA[1] + (AREA[3] - AREA[1]) / 2, 40)
blue_goal = (
    AREA[0],
    AREA[1],
    40,
    AREA[0] + 5,
    AREA[1] + 5,
    64,
)
green_goal = (
    AREA[2],
    AREA[1],
    40,
    AREA[2] + 5,
    AREA[1] + 5,
    64,
)
penalty_areas = [
    (AREA[0], AREA[1], AREA[0] + 12, AREA[3]),
    (AREA[2] - 12, AREA[1], AREA[2], AREA[3]),
]

spawn_locations_blue = [
    (1, 2, 3),
    (4, 5, 6),
]

spawn_locations_green = [
    (7, 8, 9),
]

protected = ['A1', 'B2']
`

	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if meta.Metadata.Name != "Test Map" {
		t.Fatalf("unexpected name %q", meta.Metadata.Name)
	}
	if meta.Metadata.Version != "1.2" {
		t.Fatalf("unexpected version %q", meta.Metadata.Version)
	}
	if meta.Metadata.Author != "Author" {
		t.Fatalf("unexpected author %q", meta.Metadata.Author)
	}
	if meta.Fog == nil || meta.Fog.R != 128 || meta.Fog.G != 200 || meta.Fog.B != 64 {
		t.Fatalf("unexpected fog %#v", meta.Fog)
	}
	if meta.Rules == nil {
		t.Fatalf("rules not populated")
	}
	if len(meta.Rules.Protected) != 2 || meta.Rules.Protected[0] != "A1" {
		t.Fatalf("unexpected protected %#v", meta.Rules.Protected)
	}
	if len(meta.Spawns.Blue) != 2 || len(meta.Spawns.Green) != 1 {
		t.Fatalf("unexpected spawns %#v %#v", meta.Spawns.Blue, meta.Spawns.Green)
	}
	if len(meta.Spawns.BlueArea) != 4 || len(meta.Spawns.GreenArea) != 4 {
		t.Fatalf("missing spawn areas")
	}
	if len(meta.Rules.Area) != 4 || len(meta.Rules.Ball) != 3 {
		t.Fatalf("area/ball not parsed")
	}
	if len(meta.Rules.BlueGoal) != 6 || len(meta.Rules.GreenGoal) != 6 {
		t.Fatalf("goals not parsed")
	}
	if len(meta.Rules.PenaltyAreas) != 2 {
		t.Fatalf("penalty areas missing")
	}
	if meta.Extensions["water_damage"] != int64(100) {
		t.Fatalf("water damage extension missing")
	}
	if _, ok := meta.Extensions["murderball"]; !ok {
		t.Fatalf("murderball flag missing")
	}
}

func TestCaseInsensitiveFields(t *testing.T) {
	source := `
Name = 'Capitalized'
Version = '1.0'
Author = 'Someone'
`
	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if meta.Metadata.Name != "Capitalized" {
		t.Fatalf("expected name %q, got %q", "Capitalized", meta.Metadata.Name)
	}
	if meta.Metadata.Version != "1.0" {
		t.Fatalf("expected version %q, got %q", "1.0", meta.Metadata.Version)
	}
	if meta.Metadata.Author != "Someone" {
		t.Fatalf("expected author %q, got %q", "Someone", meta.Metadata.Author)
	}
}

func TestTypoFields(t *testing.T) {
	source := `nname = 'typo name'`
	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if meta.Metadata.Name != "typo name" {
		t.Fatalf("expected name %q, got %q", "typo name", meta.Metadata.Name)
	}
}

func TestMissingNameNoError(t *testing.T) {
	source := `version = '1.0'`
	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("expected no error for missing name, got: %v", err)
	}
	if meta.Metadata.Name != "" {
		t.Fatalf("expected empty name, got %q", meta.Metadata.Name)
	}
}

func TestSpawnLocations2D(t *testing.T) {
	source := `
name = 'test'
spawn_locations_blue = [
    (100, 200),
    (150, 250),
]
spawn_locations_green = [
    (300, 400),
]
`
	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(meta.Spawns.Blue) != 2 {
		t.Fatalf("expected 2 blue spawns, got %d", len(meta.Spawns.Blue))
	}
	if meta.Spawns.Blue[0][0] != 100 || meta.Spawns.Blue[0][1] != 200 || meta.Spawns.Blue[0][2] != 0 {
		t.Fatalf("unexpected blue spawn[0]: %v", meta.Spawns.Blue[0])
	}
	if len(meta.Spawns.Green) != 1 {
		t.Fatalf("expected 1 green spawn, got %d", len(meta.Spawns.Green))
	}
	if meta.Spawns.Green[0][2] != 0 {
		t.Fatalf("expected z=0 for 2D spawn, got %v", meta.Spawns.Green[0][2])
	}
}

func TestExtensionArenaSpawns(t *testing.T) {
	source := `
name = 'arena test'
extensions = {
    'arena_blue_spawn': (10, 20, 30),
    'arena_green_spawn': (40, 50, 60),
}
`
	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(meta.Spawns.Blue) != 1 || meta.Spawns.Blue[0][0] != 10 {
		t.Fatalf("arena blue spawn not extracted: %v", meta.Spawns.Blue)
	}
	if len(meta.Spawns.Green) != 1 || meta.Spawns.Green[0][0] != 40 {
		t.Fatalf("arena green spawn not extracted: %v", meta.Spawns.Green)
	}
}

func TestExtensionPushSpawns(t *testing.T) {
	source := `
name = 'push test'
extensions = {
    'push_blue_spawn': (11, 22, 33),
    'push_green_spawn': (44, 55, 66),
    'push_blue_cp': (100, 200, 50),
    'push_green_cp': (300, 400, 50),
}
`
	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(meta.Spawns.Blue) != 1 || meta.Spawns.Blue[0][0] != 11 {
		t.Fatalf("push blue spawn not extracted: %v", meta.Spawns.Blue)
	}
	if len(meta.Spawns.Green) != 1 || meta.Spawns.Green[0][0] != 44 {
		t.Fatalf("push green spawn not extracted: %v", meta.Spawns.Green)
	}
	if len(meta.Entities.Blue.Base) != 3 || meta.Entities.Blue.Base[0] != 100 {
		t.Fatalf("push blue cp not extracted as base: %v", meta.Entities.Blue.Base)
	}
	if len(meta.Entities.Green.Base) != 3 || meta.Entities.Green.Base[0] != 300 {
		t.Fatalf("push green cp not extracted as base: %v", meta.Entities.Green.Base)
	}
}

func TestExplicitSpawnsNotOverriddenByExtensions(t *testing.T) {
	source := `
name = 'test'
spawn_locations_blue = [(1, 2, 3)]
spawn_locations_green = [(4, 5, 6)]
extensions = {
    'arena_blue_spawn': (99, 99, 99),
    'arena_green_spawn': (88, 88, 88),
}
`
	meta, err := Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if meta.Spawns.Blue[0][0] != 1 {
		t.Fatalf("explicit spawn overridden by extension: %v", meta.Spawns.Blue)
	}
	if meta.Spawns.Green[0][0] != 4 {
		t.Fatalf("explicit spawn overridden by extension: %v", meta.Spawns.Green)
	}
}
