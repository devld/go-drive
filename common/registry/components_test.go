package registry

import "testing"

type sample struct{ n int }

func TestAddRejectsTheSameComponentTwice(t *testing.T) {
	holder := NewComponentHolder()
	item := &sample{n: 1}
	holder.Add(item)
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate pointer to panic")
		}
	}()
	holder.Add(item)
}

func TestAddAllowsDistinctPointersOfTheSameType(t *testing.T) {
	holder := NewComponentHolder()
	holder.Add(&sample{n: 1})
	holder.Add(&sample{n: 1})
	if got := len(Gets[*sample](holder)); got != 2 {
		t.Fatalf("components = %d, want 2", got)
	}
}

func TestAddRejectsEqualValues(t *testing.T) {
	holder := NewComponentHolder()
	holder.Add(sample{n: 1})
	defer func() {
		if recover() == nil {
			t.Fatal("expected equal value to panic")
		}
	}()
	holder.Add(sample{n: 1})
}

func TestGetsReturnsInterfaceImplementations(t *testing.T) {
	holder := NewComponentHolder()
	holder.Add(&sample{n: 1})
	holder.Add(sample{n: 2})
	if got := len(Gets[*sample](holder)); got != 1 {
		t.Fatalf("pointers = %d, want 1", got)
	}
}
