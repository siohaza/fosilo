package vxl

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
)

const (
	chunkSize    = 16
	chunkGrowth  = 2
	chunkShrink  = 4
	defaultColor = 0x674028
)

type position uint32

func newPosition(x, y, z uint32) position {
	return position((y << 20) | (x << 8) | z)
}

func (p position) X() uint32          { return (uint32(p) >> 8) & 0xFFF }
func (p position) Y() uint32          { return (uint32(p) >> 20) & 0xFFF }
func (p position) Z() uint32          { return uint32(p) & 0xFF }
func (p position) WithoutZ() position { return position(uint32(p) & 0xFFFFFF00) }

type span struct {
	length     uint8
	colorStart uint8
	colorEnd   uint8
	airStart   uint8
}

func (s span) dataLength() int {
	if s.length > 0 {
		return int(s.length) * 4
	}
	return (int(s.colorEnd) + 2 - int(s.colorStart)) * 4
}

type block struct {
	pos   position
	color uint32
}

type chunk struct {
	blocks []block
	count  int
}

func newChunk() *chunk {
	return &chunk{
		blocks: make([]block, chunkSize*chunkSize*2),
		count:  0,
	}
}

func (c *chunk) append(pos position, color uint32) {
	if c.count == len(c.blocks) {
		newBlocks := make([]block, len(c.blocks)*chunkGrowth)
		copy(newBlocks, c.blocks)
		c.blocks = newBlocks
	}
	c.blocks[c.count] = block{pos: pos, color: color}
	c.count++
}

func (c *chunk) insert(pos position, color uint32) {
	idx := sort.Search(c.count, func(i int) bool {
		return c.blocks[i].pos >= pos
	})

	if idx < c.count && c.blocks[idx].pos == pos {
		c.blocks[idx].color = color
		return
	}

	if c.count == len(c.blocks) {
		newBlocks := make([]block, len(c.blocks)*chunkGrowth)
		copy(newBlocks, c.blocks)
		c.blocks = newBlocks
	}

	copy(c.blocks[idx+1:c.count+1], c.blocks[idx:c.count])
	c.blocks[idx] = block{pos: pos, color: color}
	c.count++
}

func (c *chunk) remove(pos position) bool {
	idx := sort.Search(c.count, func(i int) bool {
		return c.blocks[i].pos >= pos
	})

	if idx >= c.count || c.blocks[idx].pos != pos {
		return false
	}

	copy(c.blocks[idx:c.count-1], c.blocks[idx+1:c.count])
	c.count--

	if c.count*chunkShrink <= len(c.blocks) {
		newSize := len(c.blocks) / chunkGrowth
		if newSize > 0 {
			newBlocks := make([]block, newSize)
			copy(newBlocks, c.blocks[:c.count])
			c.blocks = newBlocks
		}
	}

	return true
}

func (c *chunk) get(pos position) (uint32, bool) {
	idx := sort.Search(c.count, func(i int) bool {
		return c.blocks[i].pos >= pos
	})

	if idx < c.count && c.blocks[idx].pos == pos {
		return c.blocks[idx].color, true
	}

	return 0, false
}

type Map struct {
	width    int
	height   int
	depth    int
	chunks   []*chunk
	geometry []uint64
}

func (m *Map) Width() int  { return m.width }
func (m *Map) Height() int { return m.height }
func (m *Map) Depth() int  { return m.depth }

func Size(data []byte) (size, depth int, err error) {
	offset := 0
	columns := 0
	maxDepth := 0

	for offset+4 <= len(data) {
		s := span{
			length:     data[offset],
			colorStart: data[offset+1],
			colorEnd:   data[offset+2],
			airStart:   data[offset+3],
		}

		if int(s.colorEnd)+1 > maxDepth {
			maxDepth = int(s.colorEnd) + 1
		}

		if s.length == 0 {
			columns++
		}

		length := s.dataLength()
		offset += length
		if offset > len(data) {
			break
		}
	}

	depth = 1 << int(math.Ceil(math.Log2(float64(maxDepth))))
	size = int(math.Sqrt(float64(columns)))
	return size, depth, nil
}

func Create(width, height, depth int, data []byte) (*Map, error) {
	m := newMap(width, height, depth)

	if data == nil {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				m.Set(x, y, depth-1, defaultColor)
			}
		}
		return m, nil
	}

	for i := range m.geometry {
		m.geometry[i] = 0xFFFFFFFFFFFFFFFF
	}

	if err := m.loadFromData(data); err != nil {
		return nil, err
	}

	m.addEdgeBlocks()

	return m, nil
}

func NewEmpty(width, height, depth int) (*Map, error) {
	return newMap(width, height, depth), nil
}

func newMap(width, height, depth int) *Map {
	m := &Map{
		width:  width,
		height: height,
		depth:  depth,
	}

	chunkCountX := (width + chunkSize - 1) / chunkSize
	chunkCountY := (height + chunkSize - 1) / chunkSize
	m.chunks = make([]*chunk, chunkCountX*chunkCountY)
	for i := range m.chunks {
		m.chunks[i] = newChunk()
	}

	geometrySize := (width*height*depth + 63) / 64
	m.geometry = make([]uint64, geometrySize)
	return m
}

func (m *Map) loadFromData(data []byte) error {
	offset := 0

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			c := m.chunkAt(x, y)
			if c == nil {
				continue
			}

			for {
				if offset+4 > len(data) {
					return fmt.Errorf("unexpected end of data")
				}

				s := span{
					length:     data[offset],
					colorStart: data[offset+1],
					colorEnd:   data[offset+2],
					airStart:   data[offset+3],
				}

				length := s.dataLength()
				if offset+length > len(data) {
					return fmt.Errorf("span exceeds data length")
				}

				colorData := data[offset+4 : offset+length]

				for z := int(s.airStart); z < int(s.colorStart); z++ {
					m.setGeometry(x, y, z, false)
				}

				for z := int(s.colorStart); z <= int(s.colorEnd); z++ {
					idx := (z - int(s.colorStart)) * 4
					color := uint32(colorData[idx]) |
						uint32(colorData[idx+1])<<8 |
						uint32(colorData[idx+2])<<16 |
						uint32(colorData[idx+3])<<24
					c.append(newPosition(uint32(x), uint32(y), uint32(z)), color&0xFFFFFF)
				}

				offset += length

				if s.length == 0 {
					break
				}
			}
		}
	}

	return nil
}

func (m *Map) addEdgeBlocks() {
	for z := 0; z < m.depth; z++ {
		for x := 0; x < m.width; x++ {
			if m.hasGeometry(x, 0, z) && !m.OnSurface(x, 0, z) {
				if c := m.chunkAt(x, 0); c != nil {
					c.insert(newPosition(uint32(x), 0, uint32(z)), defaultColor)
				}
			}
			if m.hasGeometry(x, m.height-1, z) && !m.OnSurface(x, m.height-1, z) {
				if c := m.chunkAt(x, m.height-1); c != nil {
					c.insert(newPosition(uint32(x), uint32(m.height-1), uint32(z)), defaultColor)
				}
			}
		}

		for y := 0; y < m.height; y++ {
			if m.hasGeometry(0, y, z) && !m.OnSurface(0, y, z) {
				if c := m.chunkAt(0, y); c != nil {
					c.insert(newPosition(0, uint32(y), uint32(z)), defaultColor)
				}
			}
			if m.hasGeometry(m.width-1, y, z) && !m.OnSurface(m.width-1, y, z) {
				if c := m.chunkAt(m.width-1, y); c != nil {
					c.insert(newPosition(uint32(m.width-1), uint32(y), uint32(z)), defaultColor)
				}
			}
		}
	}
}

func (m *Map) chunkAt(x, y int) *chunk {
	if x < 0 || y < 0 || x >= m.width || y >= m.height {
		return nil
	}
	chunkCountX := (m.width + chunkSize - 1) / chunkSize
	chunkX := x / chunkSize
	chunkY := y / chunkSize
	return m.chunks[chunkX+chunkY*chunkCountX]
}

func (m *Map) geometryOffset(x, y, z int) int {
	return z + (x+y*m.width)*m.depth
}

func (m *Map) hasGeometry(x, y, z int) bool {
	if x < 0 || y < 0 || z < 0 || x >= m.width || y >= m.height || z >= m.depth {
		return false
	}
	offset := m.geometryOffset(x, y, z)
	return (m.geometry[offset/64] & (1 << (offset % 64))) > 0
}

func (m *Map) setGeometry(x, y, z int, solid bool) {
	if x < 0 || y < 0 || z < 0 || x >= m.width || y >= m.height || z >= m.depth {
		return
	}
	offset := m.geometryOffset(x, y, z)
	idx := offset / 64
	bit := offset % 64

	if solid {
		m.geometry[idx] |= (1 << bit)
	} else {
		m.geometry[idx] &= ^(1 << bit)
	}
}

func (m *Map) IsInside(x, y, z int) bool {
	return x >= 0 && y >= 0 && z >= 0 && x < m.width && y < m.height && z < m.depth
}

func (m *Map) IsSolid(x, y, z int) bool {
	if z < 0 {
		return false
	}
	if z >= m.depth {
		return true
	}
	if x < 0 || x >= m.width {
		x = ((x % m.width) + m.width) % m.width
	}
	if y < 0 || y >= m.height {
		y = ((y % m.height) + m.height) % m.height
	}
	return m.hasGeometry(x, y, z)
}

func (m *Map) OnSurface(x, y, z int) bool {
	return !m.IsSolid(x, y+1, z) ||
		!m.IsSolid(x, y-1, z) ||
		!m.IsSolid(x+1, y, z) ||
		!m.IsSolid(x-1, y, z) ||
		!m.IsSolid(x, y, z+1) ||
		!m.IsSolid(x, y, z-1)
}

func (m *Map) HasNeighbors(x, y, z int) bool {
	return m.IsSolid(x, y+1, z) ||
		m.IsSolid(x, y-1, z) ||
		m.IsSolid(x+1, y, z) ||
		m.IsSolid(x-1, y, z) ||
		m.IsSolid(x, y, z+1) ||
		m.IsSolid(x, y, z-1)
}

func (m *Map) Get(x, y, z int) uint32 {
	if !m.IsInside(x, y, z) || !m.hasGeometry(x, y, z) {
		return 0
	}

	c := m.chunkAt(x, y)
	if c == nil {
		return 0
	}

	pos := newPosition(uint32(x), uint32(y), uint32(z))
	if color, ok := c.get(pos); ok {
		return color
	}

	return defaultColor
}

func (m *Map) Set(x, y, z int, color uint32) {
	if !m.IsInside(x, y, z) {
		return
	}

	m.setGeometry(x, y, z, true)

	if c := m.chunkAt(x, y); c != nil {
		c.insert(newPosition(uint32(x), uint32(y), uint32(z)), color)
	}

	m.updateNeighborSurfaces(x, y, z)
}

func (m *Map) SetNoOptimize(x, y, z int, color uint32) {
	if !m.IsInside(x, y, z) {
		return
	}

	m.setGeometry(x, y, z, true)

	if c := m.chunkAt(x, y); c != nil {
		c.insert(newPosition(uint32(x), uint32(y), uint32(z)), color)
	}
}

func (m *Map) updateNeighborSurfaces(x, y, z int) {
	neighbors := [][3]int{
		{x, y + 1, z},
		{x, y - 1, z},
		{x + 1, y, z},
		{x - 1, y, z},
		{x, y, z + 1},
		{x, y, z - 1},
	}

	for _, n := range neighbors {
		if m.IsSolid(n[0], n[1], n[2]) && !m.OnSurface(n[0], n[1], n[2]) {
			m.removeAir(n[0], n[1], n[2])
		}
	}
}

func (m *Map) removeAir(x, y, z int) {
	if !m.IsInside(x, y, z) || z == m.depth-1 {
		return
	}

	if !m.hasGeometry(x, y, z) {
		return
	}

	if c := m.chunkAt(x, y); c != nil {
		pos := newPosition(uint32(x), uint32(y), uint32(z))
		if c.remove(pos) {
			m.setGeometry(x, y, z, false)
		}
	}
}

func (m *Map) SetAir(x, y, z int) {
	if !m.IsInside(x, y, z) || z == m.depth-2 {
		return
	}

	type neighbor struct {
		x, y, z    int
		wasSurface bool
	}

	neighbors := []neighbor{
		{x, y + 1, z, m.IsSolid(x, y+1, z) && m.OnSurface(x, y+1, z)},
		{x, y - 1, z, m.IsSolid(x, y-1, z) && m.OnSurface(x, y-1, z)},
		{x + 1, y, z, m.IsSolid(x+1, y, z) && m.OnSurface(x+1, y, z)},
		{x - 1, y, z, m.IsSolid(x-1, y, z) && m.OnSurface(x-1, y, z)},
		{x, y, z + 1, m.IsSolid(x, y, z+1) && m.OnSurface(x, y, z+1)},
		{x, y, z - 1, m.IsSolid(x, y, z-1) && m.OnSurface(x, y, z-1)},
	}

	m.removeAir(x, y, z)

	for _, n := range neighbors {
		if !n.wasSurface && m.OnSurface(n.x, n.y, n.z) {
			if c := m.chunkAt(n.x, n.y); c != nil {
				c.insert(newPosition(uint32(n.x), uint32(n.y), uint32(n.z)), defaultColor)
			}
		}
	}
}

func (m *Map) FindTopBlock(x, y int) int {
	if !m.IsInside(x, y, 0) {
		return m.depth - 1
	}

	for z := 0; z < m.depth; z++ {
		if m.hasGeometry(x, y, z) {
			return z
		}
	}

	return m.depth - 1
}

func (m *Map) FindGroundLevel(x, y int) int {
	if !m.IsInside(x, y, 0) {
		return m.depth - 1
	}

	minTerrainDepth := 5

	for z := 0; z < m.depth-minTerrainDepth; z++ {
		if !m.hasGeometry(x, y, z) && m.hasGeometry(x, y, z+1) {
			solidCount := 0
			for dz := 1; dz <= minTerrainDepth && z+dz < m.depth; dz++ {
				if m.hasGeometry(x, y, z+dz) {
					solidCount++
				} else {
					break
				}
			}

			if solidCount >= minTerrainDepth {
				return z
			}
		}
	}

	for z := 1; z < m.depth; z++ {
		if !m.hasGeometry(x, y, z-1) && m.hasGeometry(x, y, z) {
			return z - 1
		}
	}

	return m.depth - 1
}

func (m *Map) Write() ([]byte, error) {
	var buf bytes.Buffer

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			if err := m.writeColumn(x, y, &buf); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

func (m *Map) writeColumn(x, y int, w io.Writer) error {
	for k := 0; k < m.depth; {
		airStart := uint8(k)
		for k < m.depth && !m.IsSolid(x, y, k) {
			k++
		}

		topStart := uint8(k)
		for k < m.depth && m.OnSurface(x, y, k) {
			k++
		}
		topEnd := uint8(k)

		for k < m.depth && m.IsSolid(x, y, k) && !m.OnSurface(x, y, k) {
			k++
		}

		z := k
		bottomStart := uint8(k)
		for z < m.depth && m.OnSurface(x, y, z) {
			z++
		}
		if z != m.depth {
			for k < m.depth && m.OnSurface(x, y, k) {
				k++
			}
		}
		bottomEnd := uint8(k)

		topLength := topEnd - topStart
		bottomLength := bottomEnd - bottomStart
		colorsLength := topLength + bottomLength

		s := span{
			airStart:   airStart,
			colorStart: topStart,
			colorEnd:   topEnd - 1,
		}

		if k == m.depth {
			s.length = 0
		} else {
			s.length = uint8(colorsLength + 1)
		}

		if err := binary.Write(w, binary.LittleEndian, s); err != nil {
			return err
		}

		for z := int(topStart); z < int(topEnd); z++ {
			color := m.Get(x, y, z) | 0x7F000000
			if err := binary.Write(w, binary.LittleEndian, color); err != nil {
				return err
			}
		}

		for z := int(bottomStart); z < int(bottomEnd); z++ {
			color := m.Get(x, y, z) | 0x7F000000
			if err := binary.Write(w, binary.LittleEndian, color); err != nil {
				return err
			}
		}

		if k == m.depth {
			break
		}
	}

	return nil
}

func (m *Map) WriteCompressed(w io.Writer) error {
	data, err := m.Write()
	if err != nil {
		return err
	}

	zw := zlib.NewWriter(w)
	defer zw.Close()

	_, err = zw.Write(data)
	return err
}
