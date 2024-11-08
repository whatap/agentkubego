package lang

import (
	"fmt"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/hmap"
)

type PKIND struct {
	PCode int64
	OKind int32
}

func NewPKIND(pcode int64, oid int32) *PKIND {
	p := new(PKIND)
	p.PCode = pcode
	p.OKind = oid

	return p
}

func (this *PKIND) GetPCode() int64 {
	return this.PCode
}

func (this *PKIND) GetOid() int32 {
	return this.OKind
}

func (this *PKIND) Hash() uint {
	prime := 31
	result := 1
	result = prime*result + int(this.OKind^int32(uint32(this.OKind)>>32))
	result = prime*result + int(this.PCode^int64(uint64(this.PCode)>>32))
	return uint(result)
}

func (this *PKIND) Equals(obj hmap.LinkedKey) bool {
	if this == obj.(*PKIND) {
		return true
	}
	if obj == nil {
		return false
	}

	//		if (getClass() != obj.getClass())
	//			return false;
	other := obj.(*PKIND)
	if this.OKind != other.OKind {
		return false
	}
	if this.PCode != other.PCode {
		return false
	}
	return true
}

func (this *PKIND) ToString() string {
	return fmt.Sprintln("[", this.PCode, ",", this.OKind, "]")
}

func (this *PKIND) CompareTo(n *PKIND) int {
	o := n
	v1 := this.PCode - o.PCode
	if v1 != 0 {
		if v1 > 0 {
			return 1
		} else {
			return -1
		}
	}
	v1 = int64(this.OKind - o.OKind)
	if v1 != 0 {
		if v1 > 0 {
			return 1
		} else {
			return -1
		}
	}
	return 0
}
