package validation

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type DataStream struct {
	buffer *bytes.Buffer
	order  binary.ByteOrder
}

func NewDataStream(data []byte) *DataStream {
	return &DataStream{
		buffer: bytes.NewBuffer(data),
		order:  binary.LittleEndian,
	}
}

func NewDataStreamWriter() *DataStream {
	return &DataStream{
		buffer: new(bytes.Buffer),
		order:  binary.LittleEndian,
	}
}

func (ds *DataStream) ReadUint8() (uint8, error) {
	var val uint8
	err := binary.Read(ds.buffer, ds.order, &val)
	return val, err
}

func (ds *DataStream) ReadInt8() (int8, error) {
	var val int8
	err := binary.Read(ds.buffer, ds.order, &val)
	return val, err
}

func (ds *DataStream) ReadUint16() (uint16, error) {
	var val uint16
	err := binary.Read(ds.buffer, ds.order, &val)
	return val, err
}

func (ds *DataStream) ReadUint32() (uint32, error) {
	var val uint32
	err := binary.Read(ds.buffer, ds.order, &val)
	return val, err
}

func (ds *DataStream) ReadFloat32() (float32, error) {
	var val float32
	err := binary.Read(ds.buffer, ds.order, &val)
	return val, err
}

func (ds *DataStream) ReadBytes(n int) ([]byte, error) {
	data := make([]byte, n)
	_, err := io.ReadFull(ds.buffer, data)
	return data, err
}

func (ds *DataStream) WriteUint8(val uint8) error {
	return binary.Write(ds.buffer, ds.order, val)
}

func (ds *DataStream) WriteInt8(val int8) error {
	return binary.Write(ds.buffer, ds.order, val)
}

func (ds *DataStream) WriteUint16(val uint16) error {
	return binary.Write(ds.buffer, ds.order, val)
}

func (ds *DataStream) WriteUint32(val uint32) error {
	return binary.Write(ds.buffer, ds.order, val)
}

func (ds *DataStream) WriteFloat32(val float32) error {
	return binary.Write(ds.buffer, ds.order, val)
}

func (ds *DataStream) WriteBytes(data []byte) error {
	_, err := ds.buffer.Write(data)
	return err
}

func (ds *DataStream) Bytes() []byte {
	return ds.buffer.Bytes()
}

func (ds *DataStream) Len() int {
	return ds.buffer.Len()
}

func (ds *DataStream) Reset() {
	ds.buffer.Reset()
}

func (ds *DataStream) Available() int {
	return ds.buffer.Len()
}

func (ds *DataStream) CanRead(n int) bool {
	return ds.buffer.Len() >= n
}

func ReadPacketID(data []byte) (uint8, error) {
	if len(data) < 1 {
		return 0, errors.New("data too short for packet ID")
	}
	return data[0], nil
}

func ValidatePacketSize(data []byte, expectedSize int) error {
	if len(data) < expectedSize {
		return errors.New("packet too short")
	}
	return nil
}
