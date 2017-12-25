package simplelru

import "testing"

func TestLRU(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		if k != v {
			t.Fatalf("Evict values not equal (%v!=%v)", k, v)
		}
		evictCounter += 1
	}
	l, err := NewLRU(128, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		l.Add(i, i, nil)
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v != k || v != i+128 {
			t.Fatalf("bad key: %v", k)
		}
	}
	for i := 0; i < 128; i++ {
		_, ok := l.Get(i)
		if ok {
			t.Fatalf("should be evicted")
		}
	}
	for i := 128; i < 256; i++ {
		_, ok := l.Get(i)
		if !ok {
			t.Fatalf("should not be evicted")
		}
	}
	for i := 128; i < 192; i++ {
		ok := l.Remove(i)
		if !ok {
			t.Fatalf("should be contained")
		}
		ok = l.Remove(i)
		if ok {
			t.Fatalf("should not be contained")
		}
		_, ok = l.Get(i)
		if ok {
			t.Fatalf("should be deleted")
		}
	}

	l.Get(192) // expect 192 to be last key in l.Keys()

	for i, k := range l.Keys() {
		if (i < 63 && k != i+193) || (i == 63 && k != 192) {
			t.Fatalf("out of order key: %v", k)
		}
	}

	l.Purge()

	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if l.used != 0 {
		t.Fatal("bad used: %v", l.used)
	}
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}

	// Insert with different weights
	for i := 1; i <= 20; i++ {
		l.Add(i, i, &Option{i})
	}

	if l.Len() != 7 {
		t.Fatal("expect to contain the last 7 elements")
	}

	err, _ = l.Add(20, 20, &Option{1000})
	if err != ErrTooLargeWeight {
		t.Fatal("error should be return if the weight is too high")
	}
	l.Add(20, 20, &Option{100})
	if l.Len() != 2 {
		t.Fatal("expect to contain the last 2 elements")
	}
	l.Add(20, 20, &Option{110})
	if l.Len() != 1 {
		t.Fatal("expect to contain the last one element")
	}
}

func TestLRU_GetOldest_RemoveOldest(t *testing.T) {
	l, err := NewLRU(128, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for i := 0; i < 256; i++ {
		l.Add(i, i, nil)
	}
	k, _, ok := l.GetOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 129 {
		t.Fatalf("bad: %v", k)
	}
}

// Test that Add returns true/false if an eviction occurred
func TestLRU_Add(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter += 1
	}

	l, err := NewLRU(1, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	_, evict := l.Add(1, 1, nil)
	if evict == true || evictCounter != 0 {
		t.Errorf("should not have an eviction")
	}
	_, evict = l.Add(2, 2, nil)
	if evict == false || evictCounter != 1 {
		t.Errorf("should have an eviction")
	}
}

// Test that Contains doesn't update recent-ness
func TestLRU_Contains(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1, nil)
	l.Add(2, 2, nil)
	if !l.Contains(1) {
		t.Errorf("1 should be contained")
	}

	l.Add(3, 3, nil)
	if l.Contains(1) {
		t.Errorf("Contains should not have updated recent-ness of 1")
	}
}

// Test that Peek doesn't update recent-ness
func TestLRU_Peek(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1, nil)
	l.Add(2, 2, nil)
	if v, ok := l.Peek(1); !ok || v != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	l.Add(3, 3, nil)
	if l.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}
