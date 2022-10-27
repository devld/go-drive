package utils

import "sync"

type IdPoolID interface {
	uint | uint32 | uint64
}

type IdPool[T IdPoolID] struct {
	max  T
	pool [][2]T
	mu   sync.Mutex
}

func NewIdPool[T IdPoolID]() *IdPool[T] {
	return &IdPool[T]{max: 0, pool: make([][2]T, 0), mu: sync.Mutex{}}
}

func (p *IdPool[T]) Next() T {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pool) > 0 {
		id := p.pool[0][0]
		p.pool[0][0]++
		if p.pool[0][0] > p.pool[0][1] {
			p.pool = p.pool[1:]
		}
		return id
	}

	p.max++
	return p.max
}

func (p *IdPool[T]) Release(id T) {
	if id <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.pool) == 0 {
		p.pool = append(p.pool, [2]T{id, id})
		return
	}

	s, e := 0, len(p.pool)*2-1
	var m int
	var me T
	for s <= e {
		m = s + ((e - s) >> 1)
		if m%2 == 0 {
			me = p.pool[m>>1][0]
		} else {
			me = p.pool[m>>1][1]
		}
		if id < me {
			e = m - 1
		} else if id > me {
			s = m + 1
		} else {
			break
		}
	}
	if id > me {
		m++
	}

	if m%2 == 1 {
		// found. already released
		return
	}

	// it's always the start of every range item
	m = m >> 1

	if m == 0 {
		if id == p.pool[0][0]-1 {
			p.pool[0][0] = id
		} else {
			p.pool = append(p.pool[:1], p.pool...)
			p.pool[0] = [2]T{id, id}
		}
		return
	}
	if m == len(p.pool) {
		if p.pool[m-1][1]+1 == id {
			p.pool[m-1][1] = id
		} else {
			p.pool = append(p.pool, [2]T{id, id})
		}
		return
	}
	if id == p.pool[m][0] {
		return
	}

	mergeLeft := id == p.pool[m-1][1]+1
	mergeRight := id == p.pool[m][0]-1

	if !mergeLeft && !mergeRight {
		p.pool = append(p.pool[:m+1], p.pool[m:]...)
		p.pool[m] = [2]T{id, id}
		return
	}

	if mergeRight {
		p.pool[m][0] = id
	} else if mergeLeft {
		p.pool[m-1][1] = id
	}
	if p.pool[m][0] == p.pool[m-1][1]+1 {
		deleted := p.pool[m]
		p.pool = append(p.pool[:m], p.pool[m+1:]...)
		p.pool[m-1][1] = deleted[1]
	}
}
