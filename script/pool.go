package script

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrVMPoolClosed = errors.New("VM pool is closed")

type VMPoolConfig struct {
	MaxTotal int
	MaxIdle  int
	MinIdle  int
	IdleTime time.Duration
}

type idleVM struct {
	vm         *VM
	returnedAt time.Time
}

type VMPool struct {
	base     *VM
	config   VMPoolConfig
	mu       sync.Mutex
	idle     []idleVM
	total    int
	closed   bool
	changed  chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
	cleaner  sync.WaitGroup
}

func NewVMPool(baseVM *VM, config *VMPoolConfig) *VMPool {
	validateVMPoolConfig(config)
	p := &VMPool{
		base:    baseVM,
		config:  *config,
		idle:    make([]idleVM, 0, config.MaxIdle),
		changed: make(chan struct{}),
		stop:    make(chan struct{}),
	}

	now := time.Now()
	for range config.MinIdle {
		p.idle = append(p.idle, idleVM{vm: baseVM.Fork(), returnedAt: now})
		p.total++
	}
	if config.IdleTime > 0 {
		p.cleaner.Add(1)
		go p.runCleaner(cleanPeriod(config.IdleTime))
	}
	return p
}

func validateVMPoolConfig(config *VMPoolConfig) {
	if config == nil {
		panic("VM pool config is required")
	}
	if config.MaxTotal <= 0 {
		panic("MaxTotal must be greater than zero")
	}
	if config.MaxIdle < 0 {
		panic("MaxIdle must not be negative")
	}
	if config.MinIdle < 0 {
		panic("MinIdle must not be negative")
	}
	if config.MaxIdle < config.MinIdle {
		panic("MaxIdle must be greater than or equal to MinIdle")
	}
	if config.MaxTotal < config.MinIdle {
		panic("MaxTotal must be greater than or equal to MinIdle")
	}
}

func cleanPeriod(idleTime time.Duration) time.Duration {
	period := idleTime / 2
	if period <= 0 {
		period = time.Millisecond
	}
	if period > time.Minute {
		period = time.Minute
	}
	return period
}

func (p *VMPool) Get(ctx context.Context) (*VM, error) {
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, ErrVMPoolClosed
		}
		if n := len(p.idle); n > 0 {
			item := p.idle[n-1]
			p.idle = p.idle[:n-1]
			p.mu.Unlock()
			return item.vm, nil
		}
		if p.total < p.config.MaxTotal {
			p.total++
			p.mu.Unlock()
			return p.base.Fork(), nil
		}
		changed := p.changed
		p.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

func (p *VMPool) Return(_ context.Context, vm *VM) error {
	if vm == nil {
		return errors.New("cannot return a nil VM")
	}
	vm.DisposeDisposables()

	p.mu.Lock()
	if p.closed || len(p.idle) >= p.config.MaxIdle {
		p.total--
		p.signalLocked()
		p.mu.Unlock()
		return vm.Dispose()
	}
	p.idle = append(p.idle, idleVM{vm: vm, returnedAt: time.Now()})
	p.signalLocked()
	p.mu.Unlock()
	return nil
}

func (p *VMPool) Dispose() error {
	p.stopOnce.Do(func() { close(p.stop) })
	p.cleaner.Wait()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	idle := p.idle
	p.idle = nil
	p.total -= len(idle)
	p.signalLocked()
	p.mu.Unlock()

	var disposeErr error
	for _, item := range idle {
		disposeErr = errors.Join(disposeErr, item.vm.Dispose())
	}
	return disposeErr
}

func (p *VMPool) runCleaner(period time.Duration) {
	defer p.cleaner.Done()
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.cleanExpired()
		case <-p.stop:
			return
		}
	}
}

func (p *VMPool) cleanExpired() {
	now := time.Now()
	p.mu.Lock()
	removeCount := 0
	for removeCount < len(p.idle)-p.config.MinIdle &&
		now.Sub(p.idle[removeCount].returnedAt) >= p.config.IdleTime {
		removeCount++
	}
	if removeCount == 0 {
		p.mu.Unlock()
		return
	}
	expired := append([]idleVM(nil), p.idle[:removeCount]...)
	copy(p.idle, p.idle[removeCount:])
	p.idle = p.idle[:len(p.idle)-removeCount]
	p.total -= removeCount
	p.signalLocked()
	p.mu.Unlock()

	for _, item := range expired {
		_ = item.vm.Dispose()
	}
}

func (p *VMPool) signalLocked() {
	close(p.changed)
	p.changed = make(chan struct{})
}
