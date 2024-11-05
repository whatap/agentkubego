package value

import (
	"fmt"

	"github.com/whatap/go-api/common/io"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/compare"
)

type FloatArray struct {
	Val []float32
}

func NewFloatArray(v []float32) *FloatArray {
	m := new(FloatArray)
	m.Val = v
	return m
}

func (this *FloatArray) CompareTo(o Value) int {
	if o != nil && o.GetValueType() == this.GetValueType() {
		return compare.CompareToFloats(this.Val, o.(*FloatArray).Val)
	}
	if o == nil {
		return 1
	} else {
		return int(this.GetValueType() - o.GetValueType())
	}
}

func (this *FloatArray) Equals(o Value) bool {
	if o != nil && o.GetValueType() == this.GetValueType() {
		return compare.EqualFloats(this.Val, o.(*FloatArray).Val)
	}
	return false
}

func (this *FloatArray) GetValueType() byte {
	return ARRAY_FLOAT
}
func (this *FloatArray) Write(out *io.DataOutputX) {
	out.WriteFloatArray(this.Val)
}
func (this *FloatArray) Read(in *io.DataInputX) {
	this.Val = in.ReadFloatArray()
}
func (this *FloatArray) ToString() string {
	return fmt.Sprint(this.Val)
}
