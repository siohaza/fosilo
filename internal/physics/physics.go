package physics

import (
	"math"

	"github.com/siohaza/fosilo/internal/player"
	"github.com/siohaza/fosilo/internal/protocol"
	"github.com/siohaza/fosilo/pkg/vxl"
)

const (
	PlayerRadius        = 0.45
	PlayerEyeHeight     = 0.9
	PlayerCrouchEye     = 0.45
	JumpVelocity        = -0.36
	Epsilon             = 1e-6
	DiagonalFactor      = 0.70710678
	FallDamageVel       = 0.58
	FallSlowDownVel     = 0.24
	FallDamageScalar    = 4096.0
	MaxVerticalVelocity = 3.0
)

func MovePlayer(p *player.Player, vxlMap *vxl.Map, dt float32, gameTime float32) int8 {
	if !p.Alive {
		return 0
	}

	p.RLock()
	pos := p.Position
	vel := p.Velocity
	keyStates := p.KeyStates
	crouching := p.Crouching
	sprinting := p.Sprinting
	wade := p.Wade
	airborne := p.Airborne
	lastClimb := p.LastClimb
	tool := p.Tool
	secondaryFire := p.SecondaryFire
	sneaking := p.Sneaking
	jumping := p.Jumping
	p.RUnlock()

	forward := keyStates&protocol.KeyStateForward != 0
	backward := keyStates&protocol.KeyStateBackward != 0
	left := keyStates&protocol.KeyStateLeft != 0
	right := keyStates&protocol.KeyStateRight != 0

	if jumping {
		vel.Z = JumpVelocity
		p.Lock()
		p.Jumping = false
		p.Unlock()
	}

	p.RLock()
	ori := p.Orientation
	p.RUnlock()

	frontDir := protocol.Vector3f{X: ori.X, Y: ori.Y, Z: 0}
	frontLen := float32(math.Sqrt(float64(frontDir.X*frontDir.X + frontDir.Y*frontDir.Y)))
	if frontLen > 0 {
		frontDir.X /= frontLen
		frontDir.Y /= frontLen
	}

	rightDir := protocol.Vector3f{X: -frontDir.Y, Y: frontDir.X, Z: 0}

	accel := dt
	if airborne {
		accel *= 0.1
	} else if crouching {
		accel *= 0.3
	} else if (secondaryFire && tool == protocol.ItemTypeGun) || sneaking {
		accel *= 0.5
	} else if sprinting {
		accel *= 1.3
	}

	if (forward || backward) && (left || right) {
		accel *= DiagonalFactor
	}

	if forward {
		vel.X += frontDir.X * accel
		vel.Y += frontDir.Y * accel
	} else if backward {
		vel.X -= frontDir.X * accel
		vel.Y -= frontDir.Y * accel
	}

	if left {
		vel.X -= rightDir.X * accel
		vel.Y -= rightDir.Y * accel
	} else if right {
		vel.X += rightDir.X * accel
		vel.Y += rightDir.Y * accel
	}

	oldVelZ := vel.Z
	friction := dt + 1
	vel.Z += dt
	vel.Z /= friction

	if wade {
		friction = dt*6 + 1
	} else if !airborne {
		friction = dt*4 + 1
	}
	vel.X /= friction
	vel.Y /= friction

	eyeHeight := float32(PlayerEyeHeight)
	bodyHeight := float32(1.35)
	if crouching {
		eyeHeight = float32(PlayerCrouchEye)
		bodyHeight = 0.9
	}

	pos, vel, lastClimb, airborne, wade = boxClipMove(vxlMap, pos, vel, dt, eyeHeight, bodyHeight, crouching, sprinting, ori.Z, lastClimb, gameTime)

	if pos.X < 0 {
		pos.X = 0
	}
	if pos.X >= float32(vxlMap.Width()) {
		pos.X = float32(vxlMap.Width() - 1)
	}
	if pos.Y < 0 {
		pos.Y = 0
	}
	if pos.Y >= float32(vxlMap.Height()) {
		pos.Y = float32(vxlMap.Height() - 1)
	}
	if pos.Z >= float32(vxlMap.Depth()-1) {
		pos.Z = float32(vxlMap.Depth() - 2)
	}

	clampedOldVelZ := oldVelZ
	if clampedOldVelZ > MaxVerticalVelocity {
		clampedOldVelZ = MaxVerticalVelocity
	}

	var fallDamage int8
	if vel.Z == 0 && clampedOldVelZ > FallSlowDownVel {
		vel.X *= 0.5
		vel.Y *= 0.5

		if clampedOldVelZ > FallDamageVel {
			damage := (clampedOldVelZ - FallDamageVel) * (clampedOldVelZ - FallDamageVel) * FallDamageScalar
			if damage > 127 {
				fallDamage = 127
			} else if damage < 0 {
				fallDamage = 0
			} else {
				fallDamage = int8(damage)
			}
		} else {
			fallDamage = -1
		}
	}

	repositionPlayer(p, pos, gameTime)

	p.Lock()
	p.Velocity = vel
	p.Airborne = airborne
	p.Wade = wade
	p.LastClimb = lastClimb
	p.Unlock()

	return fallDamage
}

func repositionPlayer(p *player.Player, position protocol.Vector3f, gameTime float32) {
	p.Lock()
	defer p.Unlock()

	p.EyePos = position
	p.Position = position

	f := p.LastClimb - gameTime
	if f > -0.25 {
		p.EyePos.Z += (f + 0.25) / 0.25
	}
}

func TryUncrouch(p *player.Player, vxlMap *vxl.Map) bool {
	p.RLock()
	pos := p.Position
	airborne := p.Airborne
	p.RUnlock()

	x1 := pos.X + PlayerRadius
	x2 := pos.X - PlayerRadius
	y1 := pos.Y + PlayerRadius
	y2 := pos.Y - PlayerRadius
	z1 := pos.Z + 2.25
	z2 := pos.Z - 1.35

	if airborne && !(clipBox(vxlMap, x1, y1, z1) || clipBox(vxlMap, x1, y2, z1) ||
		clipBox(vxlMap, x2, y1, z1) || clipBox(vxlMap, x2, y2, z1)) {
		return true
	} else if !(clipBox(vxlMap, x1, y1, z2) || clipBox(vxlMap, x1, y2, z2) ||
		clipBox(vxlMap, x2, y1, z2) || clipBox(vxlMap, x2, y2, z2)) {
		p.Lock()
		p.Position.Z -= 0.9
		p.EyePos.Z -= 0.9
		p.Unlock()
		return true
	}
	return false
}

func ValidateHit(shooter, target protocol.Vector3f, orientation protocol.Vector3f, tolerance float32) bool {
	f := float32(math.Sqrt(float64(orientation.X*orientation.X + orientation.Y*orientation.Y)))
	if math.Abs(float64(f)) < Epsilon {
		return false
	}

	strafe := protocol.Vector3f{
		X: -orientation.Y / f,
		Y: orientation.X / f,
		Z: 0,
	}

	height := protocol.Vector3f{
		X: -orientation.Z * strafe.Y,
		Y: orientation.Z * strafe.X,
		Z: orientation.X*strafe.Y - orientation.Y*strafe.X,
	}

	otherPos := protocol.Vector3f{
		X: target.X - shooter.X,
		Y: target.Y - shooter.Y,
		Z: target.Z - shooter.Z,
	}

	cz := otherPos.X*orientation.X + otherPos.Y*orientation.Y + otherPos.Z*orientation.Z
	if cz <= 0 {
		return false
	}

	r := 1.0 / cz
	cx := otherPos.X*strafe.X + otherPos.Y*strafe.Y + otherPos.Z*strafe.Z
	x := cx * r
	cy := otherPos.X*height.X + otherPos.Y*height.Y + otherPos.Z*height.Z
	y := cy * r
	r *= tolerance

	return x-r < 0 && x+r > 0 && y-r < 0 && y+r > 0
}

func checkAxisCollision(vxlMap *vxl.Map, coord1, coord2Fixed1, coord2Fixed2, nz, bodyHeight float32) bool {
	z := bodyHeight
	for z >= -1.36 {
		if clipBox(vxlMap, coord1, coord2Fixed1, nz+z) ||
			clipBox(vxlMap, coord1, coord2Fixed2, nz+z) {
			return true
		}
		z -= 0.9
	}
	return false
}

func checkAxisClimbCollision(vxlMap *vxl.Map, coord1, coord2Fixed1, coord2Fixed2, nz float32) bool {
	z := float32(0.35)
	for z >= -2.36 {
		if clipBox(vxlMap, coord1, coord2Fixed1, nz+z) ||
			clipBox(vxlMap, coord1, coord2Fixed2, nz+z) {
			return true
		}
		z -= 0.9
	}
	return false
}

func applyClimbEffects(vel *protocol.Vector3f, lastClimb, gameTime float32, nz *float32, bodyHeight *float32) float32 {
	vel.X *= 0.5
	vel.Y *= 0.5
	*nz--
	*bodyHeight = -1.35
	return gameTime
}

func checkVerticalCollision(vxlMap *vxl.Map, pos protocol.Vector3f, nz, bodyHeight float32) bool {
	return clipBox(vxlMap, pos.X-PlayerRadius, pos.Y-PlayerRadius, nz+bodyHeight) ||
		clipBox(vxlMap, pos.X-PlayerRadius, pos.Y+PlayerRadius, nz+bodyHeight) ||
		clipBox(vxlMap, pos.X+PlayerRadius, pos.Y-PlayerRadius, nz+bodyHeight) ||
		clipBox(vxlMap, pos.X+PlayerRadius, pos.Y+PlayerRadius, nz+bodyHeight)
}

func boxClipMove(vxlMap *vxl.Map, pos, vel protocol.Vector3f, dt, eyeHeight, bodyHeight float32, crouching, sprinting bool, orientZ, lastClimb, gameTime float32) (protocol.Vector3f, protocol.Vector3f, float32, bool, bool) {
	f := dt * 32
	nx := f*vel.X + pos.X
	ny := f*vel.Y + pos.Y
	nz := pos.Z + eyeHeight

	climb := false
	canClimb := !crouching && orientZ < 0.5 && !sprinting

	xDir := float32(-PlayerRadius)
	if vel.X >= 0 {
		xDir = PlayerRadius
	}

	hasCollisionX := checkAxisCollision(vxlMap, nx+xDir, pos.Y-PlayerRadius, pos.Y+PlayerRadius, nz, bodyHeight)

	if !hasCollisionX {
		pos.X = nx
	} else if canClimb {
		canClimbX := !checkAxisClimbCollision(vxlMap, nx+xDir, pos.Y-PlayerRadius, pos.Y+PlayerRadius, nz)
		if canClimbX {
			pos.X = nx
			climb = true
		} else {
			vel.X = 0
		}
	} else {
		vel.X = 0
	}

	yDir := float32(-PlayerRadius)
	if vel.Y >= 0 {
		yDir = PlayerRadius
	}

	hasCollisionY := checkAxisCollision(vxlMap, ny+yDir, pos.X-PlayerRadius, pos.X+PlayerRadius, nz, bodyHeight)

	if !hasCollisionY {
		pos.Y = ny
	} else if canClimb && !climb {
		canClimbY := !checkAxisClimbCollision(vxlMap, ny+yDir, pos.X-PlayerRadius, pos.X+PlayerRadius, nz)
		if canClimbY {
			pos.Y = ny
			climb = true
		} else {
			vel.Y = 0
		}
	} else if !climb {
		vel.Y = 0
	}

	if climb {
		lastClimb = applyClimbEffects(&vel, lastClimb, gameTime, &nz, &bodyHeight)
	} else {
		if vel.Z < 0 {
			bodyHeight = -bodyHeight
		}
		nz += vel.Z * dt * 32
	}

	airborne := true
	wade := false

	if checkVerticalCollision(vxlMap, pos, nz, bodyHeight) {
		if vel.Z >= 0 {
			wade = pos.Z > 61
			airborne = false
		}
		vel.Z = 0
	} else {
		pos.Z = nz - eyeHeight
	}

	return pos, vel, lastClimb, airborne, wade
}

func clipBox(vxlMap *vxl.Map, x, y, z float32) bool {
	ix := int(x)
	iy := int(y)
	iz := int(z)

	if ix < 0 || ix >= vxlMap.Width() || iy < 0 || iy >= vxlMap.Height() {
		return true
	}

	if iz < 0 {
		return false
	}

	if iz == vxlMap.Depth()-1 {
		iz = vxlMap.Depth() - 2
	} else if iz >= vxlMap.Depth() {
		return true
	}

	return vxlMap.IsSolid(ix, iy, iz)
}

func CalculateSpread(weapon protocol.WeaponType) float32 {
	switch weapon {
	case protocol.WeaponTypeRifle:
		return 0.006
	case protocol.WeaponTypeSMG:
		return 0.012
	case protocol.WeaponTypeShotgun:
		return 0.024
	}
	return 0.01
}

func CalculateDamage(weapon protocol.WeaponType, hitType protocol.HitType, distance float32) uint8 {
	switch weapon {
	case protocol.WeaponTypeRifle:
		switch hitType {
		case protocol.HitTypeHead:
			return 100
		case protocol.HitTypeTorso:
			return 49
		case protocol.HitTypeArms, protocol.HitTypeLegs:
			return 33
		case protocol.HitTypeMelee:
			return 80
		}
	case protocol.WeaponTypeSMG:
		switch hitType {
		case protocol.HitTypeHead:
			return 75
		case protocol.HitTypeTorso:
			return 29
		case protocol.HitTypeArms, protocol.HitTypeLegs:
			return 18
		case protocol.HitTypeMelee:
			return 80
		}
	case protocol.WeaponTypeShotgun:
		switch hitType {
		case protocol.HitTypeHead:
			return 37
		case protocol.HitTypeTorso:
			return 27
		case protocol.HitTypeArms, protocol.HitTypeLegs:
			return 16
		case protocol.HitTypeMelee:
			return 80
		}
	}

	return 0
}

func MoveGrenade(vxlMap *vxl.Map, position, velocity *protocol.Vector3f, dt float32) int {
	const BounceThreshold = 1.1

	oldPos := *position

	f := dt * 32
	velocity.Z += dt
	position.X += velocity.X * f
	position.Y += velocity.Y * f
	position.Z += velocity.Z * f

	newX := int(math.Floor(float64(position.X)))
	newY := int(math.Floor(float64(position.Y)))
	newZ := int(math.Floor(float64(position.Z)))

	if newX < 0 || newX >= vxlMap.Width() || newY < 0 || newY >= vxlMap.Height() {
		return 0
	}
	if newZ < 0 {
		return 0
	}
	sz := newZ
	if sz == vxlMap.Depth()-1 {
		sz = vxlMap.Depth() - 2
	} else if sz >= vxlMap.Depth() {
		return 0
	}

	if !vxlMap.IsSolid(newX, newY, sz) {
		return 0
	}

	ret := 1
	if math.Abs(float64(velocity.X)) > BounceThreshold ||
		math.Abs(float64(velocity.Y)) > BounceThreshold ||
		math.Abs(float64(velocity.Z)) > BounceThreshold {
		ret = 2
	}

	oldX := int(math.Floor(float64(oldPos.X)))
	oldY := int(math.Floor(float64(oldPos.Y)))
	oldZ := int(math.Floor(float64(oldPos.Z)))

	if newZ != oldZ && ((newX == oldX && newY == oldY) || !vxlMap.IsSolid(newX, newY, oldZ)) {
		velocity.Z = -velocity.Z
	} else if newX != oldX && ((newY == oldY && newZ == oldZ) || !vxlMap.IsSolid(oldX, newY, newZ)) {
		velocity.X = -velocity.X
	} else if newY != oldY && ((newX == oldX && newZ == oldZ) || !vxlMap.IsSolid(newX, oldY, newZ)) {
		velocity.Y = -velocity.Y
	}

	*position = oldPos
	velocity.X *= 0.36
	velocity.Y *= 0.36
	velocity.Z *= 0.36

	return ret
}

type raycastState struct {
	px, py, pz                float32
	dx, dy, dz                float32
	ix, iy, iz                int
	stepX, stepY, stepZ       int
	txMax, tyMax, tzMax       float32
	txDelta, tyDelta, tzDelta float32
	steppedIndex              int
	t                         float32
}

func normalizeDirection(direction protocol.Vector3f) (protocol.Vector3f, bool) {
	dx := direction.X
	dy := direction.Y
	dz := direction.Z

	ds := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if math.Abs(float64(ds)) < Epsilon {
		return protocol.Vector3f{}, false
	}

	return protocol.Vector3f{
		X: dx / ds,
		Y: dy / ds,
		Z: dz / ds,
	}, true
}

func calculateStepDirection(value float32) int {
	if value < 0 {
		return -1
	}
	return 1
}

func calculateAxisDistance(pos float32, iPos int, step int) float32 {
	if step > 0 {
		return float32(iPos+1) - pos
	}
	return pos - float32(iPos)
}

func calculateTMax(delta, dist float32) float32 {
	if delta == float32(math.Inf(1)) {
		return float32(math.Inf(1))
	}
	return delta * dist
}

func initializeRaycastState(start, direction protocol.Vector3f) *raycastState {
	state := &raycastState{
		px:           start.X,
		py:           start.Y,
		pz:           start.Z,
		dx:           direction.X,
		dy:           direction.Y,
		dz:           direction.Z,
		ix:           int(math.Floor(float64(start.X))),
		iy:           int(math.Floor(float64(start.Y))),
		iz:           int(math.Floor(float64(start.Z))),
		t:            0.0,
		steppedIndex: -1,
	}

	state.stepX = calculateStepDirection(state.dx)
	state.stepY = calculateStepDirection(state.dy)
	state.stepZ = calculateStepDirection(state.dz)

	state.txDelta = float32(math.Abs(1 / float64(state.dx)))
	state.tyDelta = float32(math.Abs(1 / float64(state.dy)))
	state.tzDelta = float32(math.Abs(1 / float64(state.dz)))

	xDist := calculateAxisDistance(state.px, state.ix, state.stepX)
	yDist := calculateAxisDistance(state.py, state.iy, state.stepY)
	zDist := calculateAxisDistance(state.pz, state.iz, state.stepZ)

	state.txMax = calculateTMax(state.txDelta, xDist)
	state.tyMax = calculateTMax(state.tyDelta, yDist)
	state.tzMax = calculateTMax(state.tzDelta, zDist)

	return state
}

func calculateHitNormal(state *raycastState) protocol.Vector3f {
	normal := protocol.Vector3f{}
	if state.steppedIndex == 0 {
		normal.X = -float32(state.stepX)
	} else if state.steppedIndex == 1 {
		normal.Y = -float32(state.stepY)
	} else if state.steppedIndex == 2 {
		normal.Z = -float32(state.stepZ)
	}
	return normal
}

func stepRaycast(state *raycastState) {
	if state.txMax < state.tyMax {
		if state.txMax < state.tzMax {
			state.ix += state.stepX
			state.t = state.txMax
			state.txMax += state.txDelta
			state.steppedIndex = 0
		} else {
			state.iz += state.stepZ
			state.t = state.tzMax
			state.tzMax += state.tzDelta
			state.steppedIndex = 2
		}
	} else {
		if state.tyMax < state.tzMax {
			state.iy += state.stepY
			state.t = state.tyMax
			state.tyMax += state.tyDelta
			state.steppedIndex = 1
		} else {
			state.iz += state.stepZ
			state.t = state.tzMax
			state.tzMax += state.tzDelta
			state.steppedIndex = 2
		}
	}
}

// ref http://www.cs.yorku.ca/~amana/research/grid.pdf
// https://github.com/fenomas/fast-voxel-raycast
func RaycastVXL(vxlMap *vxl.Map, start, direction protocol.Vector3f, maxDistance float32) (hit bool, hitPos protocol.Vector3f, hitBlock protocol.Vector3i, hitNormal protocol.Vector3f) {
	normalizedDir, ok := normalizeDirection(direction)
	if !ok {
		return false, protocol.Vector3f{}, protocol.Vector3i{}, protocol.Vector3f{}
	}

	state := initializeRaycastState(start, normalizedDir)

	for state.t <= maxDistance {
		if state.ix >= 0 && state.ix < vxlMap.Width() && state.iy >= 0 && state.iy < vxlMap.Height() && state.iz >= 0 && state.iz < vxlMap.Depth() {
			if vxlMap.IsSolid(state.ix, state.iy, state.iz) {
				hitPos = protocol.Vector3f{
					X: state.px + state.t*state.dx,
					Y: state.py + state.t*state.dy,
					Z: state.pz + state.t*state.dz,
				}
				hitBlock = protocol.Vector3i{
					X: int32(state.ix),
					Y: int32(state.iy),
					Z: int32(state.iz),
				}
				hitNormal = calculateHitNormal(state)
				return true, hitPos, hitBlock, hitNormal
			}
		}

		stepRaycast(state)
	}

	return false, protocol.Vector3f{}, protocol.Vector3i{}, protocol.Vector3f{}
}

func CanSee(vxlMap *vxl.Map, from, to protocol.Vector3f) bool {
	dx := to.X - from.X
	dy := to.Y - from.Y
	dz := to.Z - from.Z
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	if math.Abs(float64(distance)) < Epsilon {
		return true
	}

	direction := protocol.Vector3f{
		X: dx / distance,
		Y: dy / distance,
		Z: dz / distance,
	}

	hit, _, _, _ := RaycastVXL(vxlMap, from, direction, distance)
	return !hit
}
