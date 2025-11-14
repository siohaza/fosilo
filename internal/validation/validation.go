package validation

import (
	"math"

	"github.com/siohaza/fosilo/internal/protocol"
)

func IsValidPlayerPosition(x, y, z float32) bool {
	if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) || math.IsNaN(float64(z)) {
		return false
	}
	if math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) || math.IsInf(float64(z), 0) {
		return false
	}

	return x >= -8.0 && x <= 520.0 &&
		y >= -8.0 && y <= 520.0 &&
		z >= -8.0 && z <= 72.0
}

func IsValidOrientation(x, y, z float32) bool {
	if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) || math.IsNaN(float64(z)) {
		return false
	}
	if math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) || math.IsInf(float64(z), 0) {
		return false
	}

	length := math.Sqrt(float64(x*x + y*y + z*z))
	return length >= 0.9 && length <= 1.1
}

func IsWeaponInRange(weapon protocol.WeaponType, distance float32) bool {
	switch weapon {
	case protocol.WeaponTypeRifle:
		return distance <= 128.0
	case protocol.WeaponTypeSMG:
		return distance <= 128.0
	case protocol.WeaponTypeShotgun:
		return distance <= 64.0
	default:
		return false
	}
}

func IsMeleeInRange(distance float32) bool {
	return distance <= 5.0
}

func IsValidBlockPosition(x, y, z int) bool {
	return x >= 0 && x < 512 &&
		y >= 0 && y < 512 &&
		z >= 0 && z < 64
}

func IsValidTeam(team uint8) bool {
	return team <= 1
}

func IsValidWeapon(weapon protocol.WeaponType) bool {
	return weapon <= protocol.WeaponTypeShotgun
}

func IsValidTool(tool protocol.ItemType) bool {
	return tool <= protocol.ItemTypeGrenade
}

func IsValidHP(hp uint8) bool {
	return hp <= 100
}

func IsValidBlocks(blocks uint8) bool {
	return blocks <= 50
}

func IsValidGrenades(grenades uint8) bool {
	return grenades <= 3
}

func CalculateDistance(x1, y1, z1, x2, y2, z2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

func CalculateDistanceSquared(x1, y1, z1, x2, y2, z2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	return dx*dx + dy*dy + dz*dz
}

func NormalizeVector(x, y, z float32) (float32, float32, float32) {
	length := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	if length == 0 {
		return 0, 0, 0
	}
	return x / length, y / length, z / length
}
