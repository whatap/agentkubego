package value

import (
	"bytes"
	"fmt"
	"github.com/whatap/kube/node/src/whatap/io"
	"github.com/whatap/kube/node/src/whatap/util/compare"
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
	if this.Val == nil {
		return ""
	}
	var buffer bytes.Buffer
	for i := 0; i < len(this.Val); i++ {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(fmt.Sprintf("%f", this.Val[i]))
	}
	return buffer.String()
}
