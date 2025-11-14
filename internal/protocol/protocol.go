package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"golang.org/x/text/encoding/charmap"
)

const (
	MaxPlayers        = 32
	PlayerNameLen     = 16
	TeamNameLen       = 10
	ProtocolVersion75 = 3
)

type PacketType uint8

const (
	PacketTypePositionData     PacketType = 0
	PacketTypeOrientationData  PacketType = 1
	PacketTypeWorldUpdate      PacketType = 2
	PacketTypeInputData        PacketType = 3
	PacketTypeWeaponInput      PacketType = 4
	PacketTypeHit              PacketType = 5
	PacketTypeSetHP            PacketType = 5
	PacketTypeGrenade          PacketType = 6
	PacketTypeSetTool          PacketType = 7
	PacketTypeSetColor         PacketType = 8
	PacketTypeExistingPlayer   PacketType = 9
	PacketTypeShortPlayerData  PacketType = 10
	PacketTypeMoveObject       PacketType = 11
	PacketTypeCreatePlayer     PacketType = 12
	PacketTypeBlockAction      PacketType = 13
	PacketTypeBlockLine        PacketType = 14
	PacketTypeStateData        PacketType = 15
	PacketTypeKillAction       PacketType = 16
	PacketTypeChatMessage      PacketType = 17
	PacketTypeMapStart         PacketType = 18
	PacketTypeMapChunk         PacketType = 19
	PacketTypePlayerLeft       PacketType = 20
	PacketTypeTerritoryCapture PacketType = 21
	PacketTypeProgressBar      PacketType = 22
	PacketTypeIntelCapture     PacketType = 23
	PacketTypeIntelPickup      PacketType = 24
	PacketTypeIntelDrop        PacketType = 25
	PacketTypeRestock          PacketType = 26
	PacketTypeFogColor         PacketType = 27
	PacketTypeWeaponReload     PacketType = 28
	PacketTypeChangeTeam       PacketType = 29
	PacketTypeChangeWeapon     PacketType = 30

	PacketTypeHandShakeInit    PacketType = 31
	PacketTypeHandShakeReturn  PacketType = 32
	PacketTypeVersionRequest   PacketType = 33
	PacketTypeVersionResponse  PacketType = 34
	PacketTypeExtensionInfo    PacketType = 60
	PacketTypePlayerProperties PacketType = 64
)

type ExtensionID uint8

const (
	ExtensionIDPlayerProperties ExtensionID = 0
	ExtensionID256Players       ExtensionID = 192
	ExtensionIDMessageTypes     ExtensionID = 193
	ExtensionIDKickReason       ExtensionID = 194
)

type WeaponType uint8

const (
	WeaponTypeRifle   WeaponType = 0
	WeaponTypeSMG     WeaponType = 1
	WeaponTypeShotgun WeaponType = 2
)

type ItemType uint8

const (
	ItemTypeSpade   ItemType = 0
	ItemTypeBlock   ItemType = 1
	ItemTypeGun     ItemType = 2
	ItemTypeGrenade ItemType = 3
)

type HitType uint8

const (
	HitTypeTorso HitType = 0
	HitTypeHead  HitType = 1
	HitTypeArms  HitType = 2
	HitTypeLegs  HitType = 3
	HitTypeMelee HitType = 4
)

type KillType uint8

const (
	KillTypeWeapon      KillType = 0
	KillTypeHeadshot    KillType = 1
	KillTypeMelee       KillType = 2
	KillTypeGrenade     KillType = 3
	KillTypeFall        KillType = 4
	KillTypeTeamChange  KillType = 5
	KillTypeClassChange KillType = 6
)

type ChatType uint8

const (
	ChatTypeAll     ChatType = 0
	ChatTypeTeam    ChatType = 1
	ChatTypeSystem  ChatType = 2
	ChatTypeBig     ChatType = 3
	ChatTypeInfo    ChatType = 4
	ChatTypeWarning ChatType = 5
	ChatTypeError   ChatType = 6
)

type BlockActionType uint8

const (
	BlockActionTypeBuild                 BlockActionType = 0
	BlockActionTypeSpadeGunDestroy       BlockActionType = 1
	BlockActionTypeSpadeSecondaryDestroy BlockActionType = 2
	BlockActionTypeGrenadeDestroy        BlockActionType = 3
)

type DisconnectReason uint8

const (
	DisconnectReasonUndefined    DisconnectReason = 0
	DisconnectReasonBanned       DisconnectReason = 1
	DisconnectReasonIPLimit      DisconnectReason = 2
	DisconnectReasonWrongVersion DisconnectReason = 3
	DisconnectReasonServerFull   DisconnectReason = 4
	DisconnectReasonShutdown     DisconnectReason = 5
	DisconnectReasonKicked       DisconnectReason = 10
	DisconnectReasonInvalidName  DisconnectReason = 20
)

type GamemodeType uint8

const (
	GamemodeTypeCTF GamemodeType = 0
	GamemodeTypeTC  GamemodeType = 1
)

type KeyState uint8

const (
	KeyStateForward  KeyState = 1 << 0
	KeyStateBackward KeyState = 1 << 1
	KeyStateLeft     KeyState = 1 << 2
	KeyStateRight    KeyState = 1 << 3
	KeyStateJump     KeyState = 1 << 4
	KeyStateCrouch   KeyState = 1 << 5
	KeyStateSneak    KeyState = 1 << 6
	KeyStateSprint   KeyState = 1 << 7
)

type WeaponInput uint8

const (
	WeaponInputPrimary   WeaponInput = 1 << 0
	WeaponInputSecondary WeaponInput = 1 << 1
)

type Vector3f struct {
	X, Y, Z float32
}

type Vector3i struct {
	X, Y, Z int32
}

type Color3b struct {
	B, G, R uint8
}

type PacketPositionData struct {
	PacketID uint8
	X, Y, Z  float32
}

type PacketOrientationData struct {
	PacketID uint8
	X, Y, Z  float32
}

type PlayerPositionData struct {
	X, Y, Z    float32
	OX, OY, OZ float32
}

type PacketWorldUpdate struct {
	PacketID uint8
	Players  [MaxPlayers]PlayerPositionData
}

type PacketInputData struct {
	PacketID  uint8
	PlayerID  uint8
	KeyStates KeyState
}

type PacketWeaponInput struct {
	PacketID    uint8
	PlayerID    uint8
	WeaponInput WeaponInput
}

type PacketHit struct {
	PacketID uint8
	PlayerID uint8
	HitType  HitType
}

type PacketSetHP struct {
	PacketID uint8
	HP       uint8
	Type     uint8
	SourceX  float32
	SourceY  float32
	SourceZ  float32
}

type PacketGrenade struct {
	PacketID   uint8
	PlayerID   uint8
	FuseLength float32
	X, Y, Z    float32
	VX, VY, VZ float32
}

type PacketSetTool struct {
	PacketID uint8
	PlayerID uint8
	Tool     ItemType
}

type PacketSetColor struct {
	PacketID uint8
	PlayerID uint8
	Color    Color3b
}

type PacketExistingPlayer struct {
	PacketID uint8
	PlayerID uint8
	Team     uint8
	Weapon   WeaponType
	Item     ItemType
	Kills    uint32
	Color    Color3b
	Name     [PlayerNameLen]byte
}

type PacketShortPlayerData struct {
	PacketID uint8
	PlayerID uint8
	Team     uint8
	Weapon   WeaponType
}

type PacketMoveObject struct {
	PacketID uint8
	ObjectID uint8
	Team     uint8
	X, Y, Z  float32
}

type PacketCreatePlayer struct {
	PacketID uint8
	PlayerID uint8
	Weapon   WeaponType
	Team     uint8
	X, Y, Z  float32
	Name     [PlayerNameLen]byte
}

type PacketBlockAction struct {
	PacketID uint8
	PlayerID uint8
	Action   BlockActionType
	X, Y, Z  int32
}

type PacketBlockLine struct {
	PacketID               uint8
	PlayerID               uint8
	StartX, StartY, StartZ uint32
	EndX, EndY, EndZ       uint32
}

type CTFStateData struct {
	Team1Score   uint8
	Team2Score   uint8
	CaptureLimit uint8
	HeldIntels   uint8
	CarrierIDs   [2]uint8
	Team1Intel   Vector3f
	Team2Intel   Vector3f
	Team1Base    Vector3f
	Team2Base    Vector3f
}

type Territory struct {
	X, Y, Z float32
	Team    uint8
}

type TCStateData struct {
	TerritoryCount uint8
	Territories    [16]Territory
}

type PacketStateData struct {
	PacketID   uint8
	PlayerID   uint8
	FogColor   Color3b
	Team1Color Color3b
	Team2Color Color3b
	Team1Name  [TeamNameLen]byte
	Team2Name  [TeamNameLen]byte
	Gamemode   GamemodeType
	CTFState   CTFStateData
	TCState    TCStateData
}

type PacketKillAction struct {
	PacketID    uint8
	PlayerID    uint8
	KillerID    uint8
	KillType    KillType
	RespawnTime uint8
}

type PacketChatMessage struct {
	PacketID uint8
	PlayerID uint8
	Type     ChatType
	Message  []byte
}

type PacketMapStart struct {
	PacketID uint8
	MapSize  uint32
}

func (p *PacketMapStart) Write(w io.Writer) error {
	var buf [5]byte
	buf[0] = p.PacketID
	binary.LittleEndian.PutUint32(buf[1:], p.MapSize)
	_, err := w.Write(buf[:])
	return err
}

type PacketMapChunk struct {
	PacketID uint8
	Data     []byte
}

func (p *PacketMapChunk) Write(w io.Writer) error {
	if _, err := w.Write([]byte{p.PacketID}); err != nil {
		return err
	}
	_, err := w.Write(p.Data)
	return err
}

type PacketPlayerLeft struct {
	PacketID uint8
	PlayerID uint8
}

type PacketTerritoryCapture struct {
	PacketID uint8
	PlayerID uint8
	EntityID uint8
	Winning  uint8
	State    uint8
}

type PacketProgressBar struct {
	PacketID      uint8
	EntityID      uint8
	CapturingTeam uint8
	Rate          int8
	Progress      float32
}

type PacketIntelCapture struct {
	PacketID uint8
	PlayerID uint8
	Winning  uint8
}

type PacketIntelPickup struct {
	PacketID uint8
	PlayerID uint8
}

type PacketIntelDrop struct {
	PacketID uint8
	PlayerID uint8
	Position Vector3f
}

type PacketRestock struct {
	PacketID uint8
	PlayerID uint8
}

type PacketFogColor struct {
	PacketID uint8
	A        uint8
	Color    Color3b
}

type PacketWeaponReload struct {
	PacketID     uint8
	PlayerID     uint8
	MagazineAmmo uint8
	ReserveAmmo  uint8
}

type PacketChangeTeam struct {
	PacketID uint8
	PlayerID uint8
	TeamID   uint8
}

type PacketChangeWeapon struct {
	PacketID uint8
	PlayerID uint8
	WeaponID WeaponType
}

type PacketHandShakeInit struct {
	PacketID  uint8
	Challenge uint32
}

func (p *PacketHandShakeInit) Write(w io.Writer) error {
	var buf [5]byte
	buf[0] = p.PacketID
	binary.LittleEndian.PutUint32(buf[1:], p.Challenge)
	_, err := w.Write(buf[:])
	return err
}

type PacketHandShakeReturn struct {
	PacketID  uint8
	Challenge uint32
}

type PacketVersionRequest struct {
	PacketID uint8
}

type PacketVersionResponse struct {
	PacketID         uint8
	ClientIdentifier byte
	VersionMajor     int8
	VersionMinor     int8
	VersionRevision  int8
	OSInfoRaw        []byte
	OSInfo           string
}

type ExtensionEntry struct {
	ExtensionID      ExtensionID
	ExtensionVersion uint8
}

type PacketExtensionInfo struct {
	PacketID uint8
	Length   uint8
	Entries  []ExtensionEntry
}

func (p *PacketExtensionInfo) Write(w io.Writer) error {
	buf := make([]byte, 2+len(p.Entries)*2)
	buf[0] = p.PacketID
	buf[1] = p.Length

	for i, entry := range p.Entries {
		offset := 2 + i*2
		buf[offset] = uint8(entry.ExtensionID)
		buf[offset+1] = entry.ExtensionVersion
	}

	_, err := w.Write(buf)
	return err
}

func (p *PacketExtensionInfo) Read(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("packet too short")
	}

	p.Length = data[1]
	expectedLen := 2 + int(p.Length)*2

	if len(data) < expectedLen {
		return fmt.Errorf("packet too short for extension entries")
	}

	p.Entries = make([]ExtensionEntry, p.Length)
	for i := 0; i < int(p.Length); i++ {
		offset := 2 + i*2
		p.Entries[i] = ExtensionEntry{
			ExtensionID:      ExtensionID(data[offset]),
			ExtensionVersion: data[offset+1],
		}
	}

	return nil
}

type PacketPlayerProperties struct {
	PacketID     uint8
	SubPacketID  uint8
	PlayerID     uint8
	HP           uint8
	Blocks       uint8
	Grenades     uint8
	MagazineAmmo uint8
	ReserveAmmo  uint8
	Score        uint32
}

func (p *PacketPlayerProperties) Write(w io.Writer) error {
	var buf [12]byte
	buf[0] = p.PacketID
	buf[1] = p.SubPacketID
	buf[2] = p.PlayerID
	buf[3] = p.HP
	buf[4] = p.Blocks
	buf[5] = p.Grenades
	buf[6] = p.MagazineAmmo
	buf[7] = p.ReserveAmmo
	binary.LittleEndian.PutUint32(buf[8:], p.Score)
	_, err := w.Write(buf[:])
	return err
}

func (p *PacketVersionRequest) Write(w io.Writer) error {
	_, err := w.Write([]byte{p.PacketID})
	return err
}

var cp437Decoder = charmap.CodePage437.NewDecoder()
var cp437Encoder = charmap.CodePage437.NewEncoder()

func StringToCP437(s string) ([]byte, error) {
	return cp437Encoder.Bytes([]byte(s))
}

func CP437ToString(b []byte) (string, error) {
	trimmed := bytes.TrimRight(b, "\x00")
	decoded, err := cp437Decoder.Bytes(trimmed)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func WritePacket(w io.Writer, packet interface{}) error {
	return binary.Write(w, binary.LittleEndian, packet)
}

func ReadPacket(r io.Reader, packet interface{}) error {
	return binary.Read(r, binary.LittleEndian, packet)
}

func ReadPacketType(data []byte) (PacketType, error) {
	if len(data) < 1 {
		return 0, fmt.Errorf("packet too small")
	}
	return PacketType(data[0]), nil
}

func (p *PacketPositionData) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, p)
}

func (p *PacketPositionData) Read(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, p)
}

func (p *PacketOrientationData) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, p)
}

func (p *PacketOrientationData) Read(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, p)
}

func (p *PacketWorldUpdate) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, p)
}

func (p *PacketWorldUpdate) Read(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, p)
}

func (p *PacketStateData) Write(w io.Writer) error {
	var buf bytes.Buffer

	buf.WriteByte(p.PacketID)
	buf.WriteByte(p.PlayerID)

	buf.WriteByte(p.FogColor.B)
	buf.WriteByte(p.FogColor.G)
	buf.WriteByte(p.FogColor.R)

	buf.WriteByte(p.Team1Color.B)
	buf.WriteByte(p.Team1Color.G)
	buf.WriteByte(p.Team1Color.R)

	buf.WriteByte(p.Team2Color.B)
	buf.WriteByte(p.Team2Color.G)
	buf.WriteByte(p.Team2Color.R)

	buf.Write(p.Team1Name[:])
	buf.Write(p.Team2Name[:])

	buf.WriteByte(uint8(p.Gamemode))

	if p.Gamemode == GamemodeTypeCTF {
		team1HasEnemyIntel := (p.CTFState.HeldIntels & 2) != 0
		team2HasEnemyIntel := (p.CTFState.HeldIntels & 1) != 0

		var err error
		buf.WriteByte(p.CTFState.Team1Score)
		buf.WriteByte(p.CTFState.Team2Score)
		buf.WriteByte(p.CTFState.CaptureLimit)

		intelFlags := uint8(0)
		if team1HasEnemyIntel {
			intelFlags |= 1
		}
		if team2HasEnemyIntel {
			intelFlags |= 2
		}
		buf.WriteByte(intelFlags)

		writeFloat := func(value float32) {
			if err != nil {
				return
			}
			err = binary.Write(&buf, binary.LittleEndian, value)
		}

		if team2HasEnemyIntel {
			carrier := p.CTFState.CarrierIDs[0]
			buf.WriteByte(carrier)
			buf.Write(make([]byte, 11))
		} else {
			writeFloat(p.CTFState.Team1Intel.X)
			writeFloat(p.CTFState.Team1Intel.Y)
			writeFloat(p.CTFState.Team1Intel.Z)
		}

		if team1HasEnemyIntel {
			carrier := p.CTFState.CarrierIDs[1]
			buf.WriteByte(carrier)
			buf.Write(make([]byte, 11))
		} else {
			writeFloat(p.CTFState.Team2Intel.X)
			writeFloat(p.CTFState.Team2Intel.Y)
			writeFloat(p.CTFState.Team2Intel.Z)
		}

		writeFloat(p.CTFState.Team1Base.X)
		writeFloat(p.CTFState.Team1Base.Y)
		writeFloat(p.CTFState.Team1Base.Z)
		writeFloat(p.CTFState.Team2Base.X)
		writeFloat(p.CTFState.Team2Base.Y)
		writeFloat(p.CTFState.Team2Base.Z)

		for err == nil && buf.Len() < 104 {
			buf.WriteByte(0)
		}

		if err != nil {
			return err
		}
	} else if p.Gamemode == GamemodeTypeTC {
		var err error
		buf.WriteByte(p.TCState.TerritoryCount)

		writeFloat := func(value float32) {
			if err != nil {
				return
			}
			err = binary.Write(&buf, binary.LittleEndian, value)
		}

		for i := uint8(0); i < p.TCState.TerritoryCount && i < 16; i++ {
			territory := p.TCState.Territories[i]
			writeFloat(territory.X)
			writeFloat(territory.Y)
			writeFloat(territory.Z)
			buf.WriteByte(territory.Team)
		}

		if err != nil {
			return err
		}
	}

	_, err := w.Write(buf.Bytes())
	return err
}

func (p *PacketChatMessage) Write(w io.Writer) error {
	var buf bytes.Buffer
	buf.WriteByte(p.PacketID)
	buf.WriteByte(p.PlayerID)
	buf.WriteByte(uint8(p.Type))
	buf.Write(p.Message)
	_, err := w.Write(buf.Bytes())
	return err
}

func (p *PacketChatMessage) Read(data []byte) error {
	if len(data) < 3 {
		return fmt.Errorf("chat message packet too small")
	}
	p.PacketID = data[0]
	p.PlayerID = data[1]
	p.Type = ChatType(data[2])
	p.Message = data[3:]
	return nil
}

func (p *PacketMoveObject) Write(w io.Writer) error {
	buf := make([]byte, 15)
	buf[0] = p.PacketID
	buf[1] = p.ObjectID
	buf[2] = p.Team
	binary.LittleEndian.PutUint32(buf[3:7], math.Float32bits(p.X))
	binary.LittleEndian.PutUint32(buf[7:11], math.Float32bits(p.Y))
	binary.LittleEndian.PutUint32(buf[11:15], math.Float32bits(p.Z))
	_, err := w.Write(buf)
	return err
}

func (p *PacketMoveObject) Read(data []byte) error {
	if len(data) < 15 {
		return fmt.Errorf("move object packet too small")
	}
	p.PacketID = data[0]
	p.ObjectID = data[1]
	p.Team = data[2]
	p.X = math.Float32frombits(binary.LittleEndian.Uint32(data[3:7]))
	p.Y = math.Float32frombits(binary.LittleEndian.Uint32(data[7:11]))
	p.Z = math.Float32frombits(binary.LittleEndian.Uint32(data[11:15]))
	return nil
}

func (p *PacketTerritoryCapture) Write(w io.Writer) error {
	buf := make([]byte, 5)
	buf[0] = p.PacketID
	buf[1] = p.PlayerID
	buf[2] = p.EntityID
	buf[3] = p.Winning
	buf[4] = p.State
	_, err := w.Write(buf)
	return err
}

func (p *PacketTerritoryCapture) Read(data []byte) error {
	if len(data) < 5 {
		return fmt.Errorf("territory capture packet too small")
	}
	p.PacketID = data[0]
	p.PlayerID = data[1]
	p.EntityID = data[2]
	p.Winning = data[3]
	p.State = data[4]
	return nil
}

func (p *PacketProgressBar) Write(w io.Writer) error {
	buf := make([]byte, 8)
	buf[0] = p.PacketID
	buf[1] = p.EntityID
	buf[2] = p.CapturingTeam
	buf[3] = byte(p.Rate)
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(p.Progress))
	_, err := w.Write(buf)
	return err
}

func (p *PacketProgressBar) Read(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("progress bar packet too small")
	}
	p.PacketID = data[0]
	p.EntityID = data[1]
	p.CapturingTeam = data[2]
	p.Rate = int8(data[3])
	p.Progress = math.Float32frombits(binary.LittleEndian.Uint32(data[4:8]))
	return nil
}

func (p *PacketHandShakeReturn) Read(data []byte) error {
	if len(data) < 5 {
		return fmt.Errorf("handshake return packet too small")
	}
	p.PacketID = data[0]
	p.Challenge = binary.LittleEndian.Uint32(data[1:5])
	return nil
}

func (p *PacketVersionResponse) Read(data []byte) error {
	if len(data) < 5 {
		return fmt.Errorf("version response packet too small")
	}

	p.PacketID = data[0]
	p.ClientIdentifier = data[1]
	p.VersionMajor = int8(data[2])
	p.VersionMinor = int8(data[3])
	p.VersionRevision = int8(data[4])

	if len(data) > 5 {
		raw := data[5:]
		p.OSInfoRaw = make([]byte, len(raw))
		copy(p.OSInfoRaw, raw)

		trimmed := bytes.TrimRight(raw, "\x00")
		if decoded, err := CP437ToString(trimmed); err == nil {
			p.OSInfo = decoded
		}
	}

	return nil
}

func IsValidPosition(x, y, z float32) bool {
	return !math.IsNaN(float64(x)) && !math.IsNaN(float64(y)) && !math.IsNaN(float64(z)) &&
		!math.IsInf(float64(x), 0) && !math.IsInf(float64(y), 0) && !math.IsInf(float64(z), 0)
}

func IsValidOrientation(x, y, z float32) bool {
	return IsValidPosition(x, y, z)
}

const (
	MagazineAmmoRifle75   = 10
	MagazineAmmoSMG75     = 30
	MagazineAmmoShotgun75 = 6

	ReserveAmmoRifle75   = 50
	ReserveAmmoSMG75     = 120
	ReserveAmmoShotgun75 = 48

	FireDelayMillisecondsRifle75   = 500
	FireDelayMillisecondsSMG75     = 100
	FireDelayMillisecondsShotgun75 = 1000

	PelletQuantityRifle75   = 1
	PelletQuantitySMG75     = 1
	PelletQuantityShotgun75 = 8

	InitialBlocks   = 50
	InitialGrenades = 3
	InitialHP       = 100

	MaxHP       = 100
	MaxBlocks   = 50
	MaxGrenades = 3
)

func GetDefaultMagazineAmmo(weapon WeaponType) uint8 {
	switch weapon {
	case WeaponTypeRifle:
		return MagazineAmmoRifle75
	case WeaponTypeSMG:
		return MagazineAmmoSMG75
	case WeaponTypeShotgun:
		return MagazineAmmoShotgun75
	}
	return 0
}

func GetDefaultReserveAmmo(weapon WeaponType) uint8 {
	switch weapon {
	case WeaponTypeRifle:
		return ReserveAmmoRifle75
	case WeaponTypeSMG:
		return ReserveAmmoSMG75
	case WeaponTypeShotgun:
		return ReserveAmmoShotgun75
	}
	return 0
}

func GetFireDelay(weapon WeaponType) int {
	switch weapon {
	case WeaponTypeRifle:
		return FireDelayMillisecondsRifle75
	case WeaponTypeSMG:
		return FireDelayMillisecondsSMG75
	case WeaponTypeShotgun:
		return FireDelayMillisecondsShotgun75
	}
	return 0
}

func GetPelletCount(weapon WeaponType) int {
	switch weapon {
	case WeaponTypeRifle:
		return PelletQuantityRifle75
	case WeaponTypeSMG:
		return PelletQuantitySMG75
	case WeaponTypeShotgun:
		return PelletQuantityShotgun75
	}
	return 0
}
