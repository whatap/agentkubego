package value

import (
	"bytes"
	"fmt"
	"github.com/whatap/kube/tools/io"
	"github.com/whatap/kube/tools/util/compare"
)

type IntArray struct {
	Val []int32
}

func NewIntArray(v []int32) *IntArray {
	m := new(IntArray)
	m.Val = v
	return m
}

func (this *IntArray) CompareTo(o Value) int {
	if o != nil && o.GetValueType() == this.GetValueType() {
		return compare.CompareToInts(this.Val, o.(*IntArray).Val)
	}
	if o == nil {
		return 1
	} else {
		return int(this.GetValueType() - o.GetValueType())
	}
}

func (this *IntArray) Equals(o Value) bool {
	if o != nil && o.GetValueType() == this.GetValueType() {
		return compare.EqualInts(this.Val, o.(*IntArray).Val)
	}
	return false
}

func (this *IntArray) GetValueType() byte {
	return ARRAY_INT
}
func (this *IntArray) Write(out *io.DataOutputX) {
	out.WriteIntArray(this.Val)
}
func (this *IntArray) Read(in *io.DataInputX) {
	this.Val = in.ReadIntArray()
}

func (this *IntArray) ToString() string {
	if this.Val == nil {
		return ""
	}
	var buffer bytes.Buffer
	for i := 0; i < len(this.Val); i++ {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(fmt.Sprintf("%d", this.Val[i]))
	}
	return buffer.String()
}
