package compare

func CompareToBytes(l []byte, r []byte) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil:
		return -1
	case r == nil:
		return 1
	}
	l_sz := len(l)
	r_sz := len(r)
	for i := 0; i < l_sz && i < r_sz; i++ {
		if l[i] > r[i] {
			return 1
		}
		if l[i] < r[i] {
			return -1
		}
	}
	return l_sz - r_sz
}
func CompareToShorts(l []int16, r []int16) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil:
		return -1
	case r == nil:
		return 1
	}
	l_sz := len(l)
	r_sz := len(r)
	for i := 0; i < l_sz && i < r_sz; i++ {
		if l[i] > r[i] {
			return 1
		}
		if l[i] < r[i] {
			return -1
		}
	}
	return l_sz - r_sz
}
func CompareToInts(l []int32, r []int32) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil:
		return -1
	case r == nil:
		return 1
	}
	l_sz := len(l)
	r_sz := len(r)
	for i := 0; i < l_sz && i < r_sz; i++ {
		if l[i] > r[i] {
			return 1
		}
		if l[i] < r[i] {
			return -1
		}
	}
	return l_sz - r_sz
}
func CompareToFloats(l []float32, r []float32) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil:
		return -1
	case r == nil:
		return 1
	}
	l_sz := len(l)
	r_sz := len(r)
	for i := 0; i < l_sz && i < r_sz; i++ {
		if l[i] > r[i] {
			return 1
		}
		if l[i] < r[i] {
			return -1
		}
	}
	return l_sz - r_sz
}

func CompareToLongs(l []int64, r []int64) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil:
		return -1
	case r == nil:
		return 1
	}
	l_sz := len(l)
	r_sz := len(r)
	for i := 0; i < l_sz && i < r_sz; i++ {
		if l[i] > r[i] {
			return 1
		}
		if l[i] < r[i] {
			return -1
		}
	}
	return l_sz - r_sz
}

func CompareToDoubles(l []float64, r []float64) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil:
		return -1
	case r == nil:
		return 1
	}
	l_sz := len(l)
	r_sz := len(r)
	for i := 0; i < l_sz && i < r_sz; i++ {
		if l[i] > r[i] {
			return 1
		}
		if l[i] < r[i] {
			return -1
		}
	}
	return l_sz - r_sz
}

func CompareToStrings(l []string, r []string) int {
	switch {
	case l == nil && r == nil:
		return 0
	case l == nil:
		return -1
	case r == nil:
		return 1
	}
	l_sz := len(l)
	r_sz := len(r)
	for i := 0; i < l_sz && i < r_sz; i++ {
		if l[i] > r[i] {
			return 1
		}
		if l[i] < r[i] {
			return -1
		}
	}
	return l_sz - r_sz
}

func EqualBytes(l []byte, r []byte) bool {
	return CompareToBytes(l, r) == 0
}

func EqualShorts(l []int16, r []int16) bool {
	return CompareToShorts(l, r) == 0
}

func EqualInts(l []int32, r []int32) bool {
	return CompareToInts(l, r) == 0
}

func EqualFloats(l []float32, r []float32) bool {
	return CompareToFloats(l, r) == 0
}

func EqualLongs(l []int64, r []int64) bool {
	return CompareToLongs(l, r) == 0
}

func EqualDoubles(l []float64, r []float64) bool {
	return CompareToDoubles(l, r) == 0
}

func EqualStrings(l []string, r []string) bool {
	return CompareToStrings(l, r) == 0
}
