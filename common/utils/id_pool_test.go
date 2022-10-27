package utils

import (
	"testing"
)

func TestIdPool(t *testing.T) {
	p := NewIdPool[uint]()

	for i := 0; i < 100; i++ {
		p.Next()
	}
	if p.max != 100 {
		t.Errorf("expect p.max = %d", 100)
		return
	}

	for i := uint(0); i < 10; i++ {
		p.Release(i)
	}
	for i := uint(20); i < 30; i++ {
		p.Release(i)
	}
	for i := uint(50); i < 60; i++ {
		p.Release(i)
	}

	t.Log(p.max, p.pool)

	a := p.Next()
	if a != 1 {
		t.Errorf("expected next is 1, but %d returned", a)
		return
	}

	t.Log(p.max, p.pool)

	for i := uint(2); i <= 9; i++ {
		a := p.Next()
		if a != i {
			t.Errorf("expected next is %d, but %d returned", i, a)
		}
	}
	t.Log(p.max, p.pool)

}
