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
