package io

import (
	//"log"
	//"runtime/debug"
	//"io"
	"bytes"
	"math"
)

type DataInputX struct {
	buffer *bytes.Buffer
	offset int32
}

func NewDataInputX(buf []byte) *DataInputX {
	in := new(DataInputX)
	in.buffer = bytes.NewBuffer(buf)
	return in
}

func (in *DataInputX) ReadIntBytes() []byte {
	sz := in.ReadInt()
	return in.ReadBytes(sz)
}
func (in *DataInputX) ReadIntBytesLimit(max int) []byte {
	sz := in.ReadInt()
	if sz < 0 || sz > int32(max) {
		return nil
	}
	return in.ReadBytes(sz)
}

func (in *DataInputX) ReadBytes(sz int32) []byte {
	in.offset += sz
	buff, err := NewByteArray(int32(sz))
	if err != nil {
		return nil
	}

	if _, err := in.buffer.Read(buff); err != nil {
		return nil
	}

	return buff

}
func (in *DataInputX) ReadShortBytes() []byte {
	sz := int32(uint16(in.ReadShort()))
	return in.ReadBytes(sz)
}
func (in *DataInputX) ReadBlob() []byte {
	baselen := int32(uint8(in.ReadByte()))
	switch baselen {
	case 255:
		sz := int32(in.ReadUnsignedShort())
		return in.ReadBytes(sz)
	case 254:
		sz := in.ReadInt()
		return in.ReadBytes(sz)
	case 0:
		return []byte{}
	default:
		return in.ReadBytes(baselen)
	}
}
func (in *DataInputX) ReadBool() bool {
	if b := in.ReadBytes(1); b != nil {
		return b[0] == 1
	}
	return false
}
func (in *DataInputX) ReadByte() byte {
	if b := in.ReadBytes(1); b != nil {
		return b[0]
	}
	return 0
}
func (in *DataInputX) ReadShort() int16 {
	if b := in.ReadBytes(2); b != nil {
		return ToShort(b, 0)
	}
	return 0
}

func (in *DataInputX) ReadShortLittle() int16 {
	if b := in.ReadBytes(2); b != nil {
		return ToShortLittle(b, 0)
	}
	return 0
}
func (in *DataInputX) ReadUnsignedShort() uint16 {
	if b := in.ReadBytes(2); b != nil {
		return uint16(ToShort(b, 0))
	}
	return 0
}
func (in *DataInputX) ReadUnsignedShortLittle() uint16 {
	if b := in.ReadBytes(2); b != nil {
		return uint16(ToUshortLittle(b, 0))
	}
	return 0
}
func (in *DataInputX) ReadInt3() int32 {
	if b := in.ReadBytes(3); b != nil {
		return ToInt3(b, 0)
	}
	return 0
}
func (in *DataInputX) ReadInt() int32 {
	if b := in.ReadBytes(4); b != nil {
		return ToInt(b, 0)
	}
	return 0
}

func (in *DataInputX) ReadUnsignedInt() uint32 {
	if b := in.ReadBytes(4); b != nil {
		return ToUint(b, 0)
	}
	return 0
}

func (in *DataInputX) ReadIntLittle() int32 {
	if b := in.ReadBytes(4); b != nil {
		return ToIntLittle(b, 0)
	}
	return 0
}

func (in *DataInputX) ReadUintLittle() uint32 {
	if b := in.ReadBytes(4); b != nil {
		return ToUintLittle(b, 0)
	}
	return 0
}

func (in *DataInputX) ReadLong5() int64 {
	if b := in.ReadBytes(5); b != nil {
		return ToLong5(b, 0)
	}
	return 0
}
func (in *DataInputX) ReadLong() int64 {
	if b := in.ReadBytes(8); b != nil {
		return ToLong(b, 0)
	}
	return 0
}

func (in *DataInputX) ReadDecimal() int64 {
	sz := uint8(in.ReadByte())
	switch sz {
	case 0:
		return 0
	case 1:
		// -1 이 255로 계산되는 부분 보정.
		return int64(int8(in.ReadByte()))
	case 2:
		return int64(in.ReadShort())
	case 3:
		return int64(in.ReadInt3())
	case 4:
		return int64(in.ReadInt())
	case 5:
		return in.ReadLong5()
	default:
		return in.ReadLong()
	}
}

func (in *DataInputX) ReadDecimalLen(sz int) int64 {
	switch sz {
	case 0:
		return 0
	case 1:
		// -1 이 255로 계산되는 부분 보정.
		return int64(int8(in.ReadByte()))
	case 2:
		return int64(in.ReadShort())
	case 3:
		return int64(in.ReadInt3())
	case 4:
		return int64(in.ReadInt())
	case 5:
		return in.ReadLong5()
	case 8:
		return in.ReadLong()
	default:
		return in.ReadLong()
	}
}
func (in *DataInputX) ReadFloat() float32 {
	if b := in.ReadBytes(4); b != nil {
		return ToFloat(b, 0)
	}
	return 0
}
func (in *DataInputX) ReadDouble() float64 {
	if b := in.ReadBytes(8); b != nil {
		return ToDouble(b, 0)
	}
	return 0
}

func (in *DataInputX) ReadShortArray() []int16 {
	sz := int(in.ReadShort())
	if sz == 0 {
		return []int16{}
	}
	v, err := NewInt16Array(int32(sz))
	if err != nil {
		return []int16{}
	}
	for i := 0; i < sz; i++ {
		v[i] = in.ReadShort()
	}
	return v
}
func (in *DataInputX) ReadIntArray() []int32 {
	sz := int(in.ReadShort())
	if sz == 0 {
		return []int32{}
	}
	v, err := NewInt32Array(int32(sz))
	if err != nil {
		return []int32{}
	}
	for i := 0; i < sz; i++ {
		v[i] = in.ReadInt()
	}
	return v
}
func (in *DataInputX) ReadLongArray() []int64 {
	sz := int(in.ReadShort())
	if sz == 0 {
		return []int64{}
	}
	v, err := NewInt64Array(int32(sz))
	if err != nil {
		return []int64{}
	}
	for i := 0; i < sz; i++ {
		v[i] = in.ReadLong()
	}
	return v
}
func (in *DataInputX) ReadFloatArray() []float32 {
	sz := int(in.ReadShort())
	if sz == 0 {
		return []float32{}
	}
	v, err := NewFloat32Array(int32(sz))
	if err != nil {
		return []float32{}
	}
	for i := 0; i < sz; i++ {
		v[i] = in.ReadFloat()
	}
	return v
}
func (in *DataInputX) ReadDoubleArray() []float64 {
	sz := int(in.ReadShort())
	if sz == 0 {
		return []float64{}
	}
	v, err := NewFloat64Array(int32(sz))
	if err != nil {
		return []float64{}
	}
	for i := 0; i < sz; i++ {
		v[i] = in.ReadDouble()
	}
	return v
}
func (in *DataInputX) ReadTextArray() []string {
	sz := int(in.ReadShort())
	if sz == 0 {
		return []string{}
	}
	v, err := NewStringArray(int32(sz))
	if err != nil {
		return []string{}
	}
	for i := 0; i < sz; i++ {
		v[i] = in.ReadText()
	}
	return v
}
func (in *DataInputX) ReadText() string {
	b := in.ReadBlob()
	return string(b)
}

func ToBool(buf []byte, pos int) bool {
	return buf[pos] != 0
}

func ToShort(buf []byte, pos int) int16 {
	ch1 := int16(buf[pos])
	ch2 := int16(buf[pos+1])
	return (int16)((ch1 << 8) + (ch2 << 0))
}

func ToUshort(buf []byte, pos int) uint16 {
	ch1 := int16(buf[pos])
	ch2 := int16(buf[pos+1])
	return (uint16)((ch1 << 8) + (ch2 << 0))
}

func ToShortLittle(buf []byte, pos int) int16 {
	ch2 := int16(buf[pos])
	ch1 := int16(buf[pos+1])
	return (int16)((ch1 << 8) + (ch2 << 0))
}

func ToUshortLittle(buf []byte, pos int) uint16 {
	ch2 := int16(buf[pos])
	ch1 := int16(buf[pos+1])
	return (uint16)((ch2 << 8) + (ch1 << 0))
}

func ToInt3(buf []byte, pos int) int32 {
	ch1 := int32(buf[pos])
	ch2 := int32(buf[pos+1])
	ch3 := int32(buf[pos+2])
	return int32((ch1<<24)+(ch2<<16)+(ch3<<8)) >> 8
}

func ToInt(buf []byte, pos int) int32 {
	ch1 := int32(buf[pos])
	ch2 := int32(buf[pos+1])
	ch3 := int32(buf[pos+2])
	ch4 := int32(buf[pos+3])
	return int32((ch1 << 24) + (ch2 << 16) + (ch3 << 8) + (ch4 << 0))
}

func ToUint(buf []byte, pos int) uint32 {
	ch1 := uint32(buf[pos])
	ch2 := uint32(buf[pos+1])
	ch3 := uint32(buf[pos+2])
	ch4 := uint32(buf[pos+3])
	return (ch1 << 24) + (ch2 << 16) + (ch3 << 8) + (ch4 << 0)
}

func ToIntLittle(buf []byte, pos int) int32 {
	ch4 := int32(buf[pos])
	ch3 := int32(buf[pos+1])
	ch2 := int32(buf[pos+2])
	ch1 := int32(buf[pos+3])
	return int32((ch1 << 24) + (ch2 << 16) + (ch3 << 8) + (ch4 << 0))
}

func ToUintLittle(buf []byte, pos int) uint32 {
	ch4 := uint32(buf[pos])
	ch3 := uint32(buf[pos+1])
	ch2 := uint32(buf[pos+2])
	ch1 := uint32(buf[pos+3])
	return uint32((ch1 << 24) + (ch2 << 16) + (ch3 << 8) + (ch4 << 0))
}

func ToLong(buf []byte, pos int) int64 {
	v := (int64(buf[pos]) << 56)
	v += (int64(buf[pos+1]) << 48)
	v += (int64(buf[pos+2]) << 40)
	v += (int64(buf[pos+3]) << 32)
	v += (int64(buf[pos+4]) << 24)
	v += (int64(buf[pos+5]) << 16)
	v += (int64(buf[pos+6]) << 8)
	v += (int64(buf[pos+7]) << 0)
	return v
}
func ToLong5(buf []byte, pos int) int64 {
	v := (int64(int8(buf[pos])) << 32)
	v += (int64(buf[pos+1]) << 24)
	v += (int64(buf[pos+2]) << 16)
	v += (int64(buf[pos+3]) << 8)
	v += (int64(buf[pos+4]) << 0)
	return v
}
func ToLong6(buf []byte, pos int) int64 {
	v := (int64(buf[pos]) << 40)
	v += (int64(buf[pos+1]) << 32)
	v += (int64(buf[pos+2]) << 24)
	v += (int64(buf[pos+3]) << 16)
	v += (int64(buf[pos+4]) << 8)
	v += (int64(buf[pos+5]) << 0)
	return v
}

func ToLongLittle(buf []byte, pos int) int64 {
	ch8 := int64(buf[pos])
	ch7 := int64(buf[pos+1])
	ch6 := int64(buf[pos+2])
	ch5 := int64(buf[pos+3])
	ch4 := int64(buf[pos+4])
	ch3 := int64(buf[pos+5])
	ch2 := int64(buf[pos+6])
	ch1 := int64(buf[pos+7])
	return int64((ch1 << 56) + (ch2 << 48) + (ch3 << 40) + (ch4 << 32) + (ch5 << 24) + (ch6 << 16) + (ch7 << 8) + (ch8 << 0))
}

func ToUlongLittle(buf []byte, pos int) uint64 {
	ch8 := uint64(buf[pos])
	ch7 := uint64(buf[pos+1])
	ch6 := uint64(buf[pos+2])
	ch5 := uint64(buf[pos+3])
	ch4 := uint64(buf[pos+4])
	ch3 := uint64(buf[pos+5])
	ch2 := uint64(buf[pos+6])
	ch1 := uint64(buf[pos+7])
	return uint64((ch1 << 56) + (ch2 << 48) + (ch3 << 40) + (ch4 << 32) + (ch5 << 24) + (ch6 << 16) + (ch7 << 8) + (ch8 << 0))
}

func ToFloat(buf []byte, pos int) float32 {
	return math.Float32frombits(uint32(ToInt(buf, pos)))
}

func ToDouble(buf []byte, pos int) float64 {
	return math.Float64frombits(uint64(ToLong(buf, pos)))
}
func Get(buf []byte, pos int, sz int) []byte {
	return buf[pos : pos+sz]
}

func (in *DataInputX) ReadDecimalArrayInt() []int32 {
	sz := int(in.ReadDecimal())
	data, err := NewInt32Array(int32(sz))
	if err != nil {
		return []int32{}
	}
	for i := 0; i < sz; i++ {
		data[i] = int32(in.ReadDecimal())
	}
	return data
}
func (in *DataInputX) ReadDecimalArray() []int64 {
	sz := int(in.ReadDecimal())
	data, err := NewInt64Array(int32(sz))
	if err != nil {
		return []int64{}
	}
	for i := 0; i < sz; i++ {
		data[i] = in.ReadDecimal()
	}
	return data
}
