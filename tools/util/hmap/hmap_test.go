package hmap

import "testing"

func TestIntKeyLinkedMapBasic(t *testing.T) {
	m := NewIntKeyLinkedMapDefault()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	if m.Size() != 3 {
		t.Fatalf("size = %d, want 3", m.Size())
	}
	if v := m.Get(2); v != "b" {
		t.Fatalf("Get(2) = %v, want b", v)
	}
	// overwrite keeps size
	m.Put(2, "B")
	if m.Size() != 3 {
		t.Fatalf("size after overwrite = %d, want 3", m.Size())
	}
	if v := m.Get(2); v != "B" {
		t.Fatalf("Get(2) after overwrite = %v, want B", v)
	}
	m.Remove(1)
	if m.Size() != 2 || m.ContainsKey(1) {
		t.Fatalf("Remove(1) failed: size=%d contains=%v", m.Size(), m.ContainsKey(1))
	}
	// insertion-order iteration (linked map contract)
	keys := []int32{}
	for e := m.Keys(); e.HasMoreElements(); {
		keys = append(keys, e.NextInt())
	}
	if len(keys) != 2 || keys[0] != 2 || keys[1] != 3 {
		t.Fatalf("key order = %v, want [2 3]", keys)
	}
	m.Clear()
	if m.Size() != 0 {
		t.Fatalf("size after Clear = %d, want 0", m.Size())
	}
}

func TestStringKeyLinkedMapBasic(t *testing.T) {
	m := NewStringKeyLinkedMap()
	m.Put("x", 10)
	m.Put("y", 20)
	if m.Size() != 2 {
		t.Fatalf("size = %d, want 2", m.Size())
	}
	if v := m.Get("x"); v != 10 {
		t.Fatalf("Get(x) = %v, want 10", v)
	}
	m.Remove("x")
	if m.ContainsKey("x") {
		t.Fatal("ContainsKey(x) after Remove = true, want false")
	}
	m.Clear()
	if m.Size() != 0 {
		t.Fatalf("size after Clear = %d, want 0", m.Size())
	}
}

func TestStringKeyLinkedMapSetMaxEviction(t *testing.T) {
	m := NewStringKeyLinkedMap().SetMax(2)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3) // exceeds max=2 → oldest evicted
	if m.Size() != 2 {
		t.Fatalf("size = %d, want 2 (max eviction)", m.Size())
	}
	if m.ContainsKey("a") {
		t.Fatal("oldest entry 'a' should have been evicted")
	}
}
