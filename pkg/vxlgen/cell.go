package vxlgen

type CellType int

const (
	CellAir CellType = iota
	CellRegular
	CellRoomFloor
	CellFull
	CellStairBody
	CellStairEndHigh
	CellStairEndLow
)

type BalconyType int

const (
	BalconyNone BalconyType = iota
	BalconySimple
	BalconyBattlement
)

type Cell struct {
	HasLeftWall bool
	HasTopWall  bool
	HasFloor    bool
	Type        CellType
	Balcony     BalconyType
	Color       int
}

func (ct CellType) ShouldBeConnected() bool {
	switch ct {
	case CellRegular, CellRoomFloor, CellStairEndHigh, CellStairEndLow:
		return true
	default:
		return false
	}
}

func (ct CellType) AvailableForRoom() bool {
	switch ct {
	case CellRegular, CellFull, CellStairEndHigh:
		return true
	default:
		return false
	}
}

func (ct CellType) AvailableForStair() bool {
	return ct == CellRegular
}

func (ct CellType) IsStairPart() bool {
	switch ct {
	case CellStairBody, CellStairEndLow:
		return true
	default:
		return false
	}
}
