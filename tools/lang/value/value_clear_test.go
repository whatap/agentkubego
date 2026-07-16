package value

import "testing"

// Regression: MapValue.Clear()/IntMapValue.Clear() used to call themselves
// (infinite recursion → stack overflow) instead of clearing the backing table.
func TestMapValueClear(t *testing.T) {
	mv := NewMapValue()
	mv.PutString("k1", "v1")
	mv.PutLong("k2", 42)
	if mv.Size() != 2 {
		t.Fatalf("size = %d, want 2", mv.Size())
	}
	mv.Clear()
	if mv.Size() != 0 {
		t.Fatalf("size after Clear = %d, want 0", mv.Size())
	}
	// map must remain usable after Clear
	mv.PutString("k3", "v3")
	if mv.Size() != 1 {
		t.Fatalf("size after re-put = %d, want 1", mv.Size())
	}
}

func TestIntMapValueClear(t *testing.T) {
	mv := NewIntMapValue()
	mv.PutLong(1, 100)
	mv.PutString(2, "s")
	if mv.Size() != 2 {
		t.Fatalf("size = %d, want 2", mv.Size())
	}
	mv.Clear()
	if mv.Size() != 0 {
		t.Fatalf("size after Clear = %d, want 0", mv.Size())
	}
}
