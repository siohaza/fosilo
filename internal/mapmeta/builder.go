package mapmeta

import (
	"math"
)

func buildMetadata(env map[string]Value) *Metadata {
	meta := &Metadata{
		Metadata: MetadataInfo{},
		Spawns: Spawns{
			Blue:  [][]float64{},
			Green: [][]float64{},
		},
		Extensions: make(map[string]any),
	}

	if v, ok := env["name"]; ok {
		if s, err := v.asString(); err == nil {
			meta.Metadata.Name = s
		}
	}
	if v, ok := env["version"]; ok {
		if s, err := v.asString(); err == nil {
			meta.Metadata.Version = s
		}
	}
	if v, ok := env["author"]; ok {
		if s, err := v.asString(); err == nil {
			meta.Metadata.Author = s
		}
	}
	if v, ok := env["description"]; ok {
		if s, err := v.asString(); err == nil {
			meta.Metadata.Description = s
		}
	} else if v, ok := env["desc"]; ok {
		if s, err := v.asString(); err == nil {
			meta.Metadata.Description = s
		}
	}

	if v, ok := env["fog"]; ok {
		if fogVals := toFloatSlice(v, 3); len(fogVals) == 3 {
			meta.Fog = &Fog{
				R: uint8(clampFloat(fogVals[0], 0, 255)),
				G: uint8(clampFloat(fogVals[1], 0, 255)),
				B: uint8(clampFloat(fogVals[2], 0, 255)),
			}
		}
	}

	if v, ok := env["extensions"]; ok && v.kind == valueDict {
		for key, entry := range v.dict {
			if val := entry.toInterface(); val != nil {
				meta.Extensions[key] = val
			}
		}
	}

	for _, key := range []string{"murderball", "boss"} {
		if v, ok := env[key]; ok {
			if b, err := v.asBool(); err == nil && b {
				meta.Extensions[key] = b
			}
		}
	}

	if v, ok := env["cap_limit"]; ok {
		if n, err := v.asNumber(); err == nil {
			meta.Extensions["cap_limit"] = int(n)
		}
	}

	if v, ok := env["spawn_locations_blue"]; ok {
		meta.Spawns.Blue = toMatrix(v, 3)
	}
	if v, ok := env["spawn_locations_green"]; ok {
		meta.Spawns.Green = toMatrix(v, 3)
	}

	if v, ok := env["BLUE_RECT"]; ok {
		if rect := toFloatSlice(v, 4); len(rect) == 4 {
			meta.Spawns.BlueArea = rect
		}
	}
	if v, ok := env["GREEN_RECT"]; ok {
		if rect := toFloatSlice(v, 4); len(rect) == 4 {
			meta.Spawns.GreenArea = rect
		}
	}

	var rules Rules
	hasRules := false

	if v, ok := env["protected"]; ok {
		if list := toStringSlice(v); len(list) > 0 {
			rules.Protected = list
			hasRules = true
		}
	}

	if v, ok := env["area"]; ok {
		if vals := toFloatSlice(v, 4); len(vals) == 4 {
			rules.Area = vals
			hasRules = true
		}
	}
	if v, ok := env["ball"]; ok {
		if vals := toFloatSlice(v, 3); len(vals) == 3 {
			rules.Ball = vals
			hasRules = true
		}
	}
	if v, ok := env["blue_goal"]; ok {
		if vals := toFloatSlice(v, 6); len(vals) == 6 {
			rules.BlueGoal = vals
			hasRules = true
		}
	}
	if v, ok := env["green_goal"]; ok {
		if vals := toFloatSlice(v, 6); len(vals) == 6 {
			rules.GreenGoal = vals
			hasRules = true
		}
	}
	if v, ok := env["penalty_areas"]; ok {
		if vals := toMatrix(v, 4); len(vals) > 0 {
			rules.PenaltyAreas = vals
			hasRules = true
		}
	}

	if hasRules {
		meta.Rules = &rules
	}

	if v, ok := env["intel_locations_blue"]; ok {
		if positions := toMatrix(v, 3); len(positions) > 0 {
			meta.Entities.Blue.Flag = positions[0]
		}
	}
	if v, ok := env["intel_locations_green"]; ok {
		if positions := toMatrix(v, 3); len(positions) > 0 {
			meta.Entities.Green.Flag = positions[0]
		}
	}

	if v, ok := env["base_locations_blue"]; ok {
		if positions := toMatrix(v, 3); len(positions) > 0 {
			meta.Entities.Blue.Base = positions[0]
		}
	}
	if v, ok := env["base_locations_green"]; ok {
		if positions := toMatrix(v, 3); len(positions) > 0 {
			meta.Entities.Green.Base = positions[0]
		}
	}

	return meta
}

func toFloatSlice(v Value, expected int) []float64 {
	if v.kind != valueList {
		return nil
	}
	if expected > 0 && len(v.list) != expected {
		return nil
	}
	result := make([]float64, 0, len(v.list))
	for _, item := range v.list {
		f, err := item.asNumber()
		if err != nil {
			return nil
		}
		result = append(result, f)
	}
	return result
}

func toMatrix(v Value, rowLen int) [][]float64 {
	if v.kind != valueList {
		return nil
	}
	rows := make([][]float64, 0, len(v.list))
	for _, item := range v.list {
		if item.kind != valueList {
			continue
		}
		values := toFloatSlice(item, 0)
		if len(values) < rowLen {
			continue
		}
		rows = append(rows, values[:rowLen])
	}
	if len(rows) == 0 {
		return nil
	}
	return rows
}

func toStringSlice(v Value) []string {
	if v.kind != valueList {
		return nil
	}
	result := make([]string, 0, len(v.list))
	for _, item := range v.list {
		if item.kind != valueString {
			return nil
		}
		result = append(result, item.str)
	}
	return result
}

func clampFloat(v float64, min, max float64) float64 {
	return math.Min(math.Max(v, min), max)
}
