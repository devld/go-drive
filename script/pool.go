package script

import (
	"context"
	"errors"
	"go-drive/common/logging"
	"sync"
	"time"
)

var ErrVMPoolClosed = errors.New("VM pool is closed")

type VMInitializer func(context.Context, *VM) error

type VMPoolConfig struct {
	MaxTotal int
	MaxIdle  int
	MinIdle  int
	IdleTime time.Duration
	// Classes is merged onto every pool member after NewVM.
	Classes *ClassSet
}

type idleVM struct {
	vm         *VM
	returnedAt time.Time
}

type VMPool struct {
	initialize VMInitializer
	config     VMPoolConfig
	mu         sync.Mutex
	idle       []idleVM
	members    map[*VM]bool
	total      int
	closed     bool
	changed    chan struct{}
	stop       chan struct{}
	stopOnce   sync.Once
	cleaner    sync.WaitGroup
}

// NewVMPool creates members with NewVM, then calls initialize.
func NewVMPool(ctx context.Context, initialize VMInitializer, config *VMPoolConfig) (*VMPool, error) {
	validateVMPoolConfig(config)
	p := &VMPool{
		initialize: initialize,
		config:     *config,
		idle:       make([]idleVM, 0, config.MaxIdle),
		members:    make(map[*VM]bool, config.MaxTotal),
		changed:    make(chan struct{}),
		stop:       make(chan struct{}),
	}

	initialized := false
	defer func() {
		if !initialized {
			_ = p.Dispose()
		}
	}()

	now := time.Now()
	for range config.MinIdle {
		vm, e := p.newVM(ctx)
		if e != nil {
			return nil, e
		}
		p.idle = append(p.idle, idleVM{vm: vm, returnedAt: now})
		p.members[vm] = false
		p.total++
	}
	if config.IdleTime > 0 {
		p.cleaner.Add(1)
		go p.runCleaner(cleanPeriod(config.IdleTime))
	}
	initialized = true
	return p, nil
}

func (p *VMPool) newVM(ctx context.Context) (*VM, error) {
	if ctx != nil {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
	}
	vm, e := NewVM()
	if e != nil {
		return nil, e
	}
	initialized := false
	defer func() {
		if !initialized {
			_ = vm.Dispose()
		}
	}()
	if p.config.Classes != nil {
		if e := vm.AddClassSet(p.config.Classes); e != nil {
			return nil, e
		}
	}
	if p.initialize != nil {
		if e := p.initialize(ctx, vm); e != nil {
			return nil, e
		}
	}
	if e := vm.Do(ctx, vm.freezeGlobal); e != nil {
		return nil, e
	}
	if !vm.Reusable() {
		return nil, errors.New("VM initializer produced a non-reusable VM")
	}
	initialized = true
	return vm, nil
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
	if ctx == nil {
		ctx = context.Background()
	}
	var waitStarted time.Time
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			logging.For("script").Debugf("VM pool get rejected reason=closed")
			return nil, ErrVMPoolClosed
		}
		if n := len(p.idle); n > 0 {
			item := p.idle[n-1]
			p.idle[n-1] = idleVM{}
			p.idle = p.idle[:n-1]
			p.members[item.vm] = true
			p.mu.Unlock()
			if !waitStarted.IsZero() {
				logging.For("script").Debugf("VM pool wait completed duration=%s", time.Since(waitStarted))
			}
			return item.vm, nil
		}
		if p.total < p.config.MaxTotal {
			p.total++ // Reserve capacity before creating outside the lock.
			p.mu.Unlock()
			vm, e := p.createReservedVM(ctx)
			if e != nil {
				return nil, e
			}
			p.mu.Lock()
			closed := p.closed
			if closed {
				p.total--
				p.signalLocked()
			} else {
				p.members[vm] = true
			}
			p.mu.Unlock()
			if closed {
				_ = vm.Dispose()
				return nil, ErrVMPoolClosed
			}
			if !waitStarted.IsZero() {
				logging.For("script").Debugf("VM pool wait completed duration=%s", time.Since(waitStarted))
			}
			return vm, nil
		}
		changed := p.changed
		p.mu.Unlock()
		if waitStarted.IsZero() {
			waitStarted = time.Now()
			logging.For("script").Debugf("VM pool exhausted max=%d", p.config.MaxTotal)
		}

		select {
		case <-ctx.Done():
			logging.For("script").Debugf("VM pool wait canceled duration=%s: %v", time.Since(waitStarted), ctx.Err())
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

// Release the reserved slot on errors, panics, and Goexit. Native panics
// continue to the caller after cleanup.
func (p *VMPool) createReservedVM(ctx context.Context) (*VM, error) {
	created := false
	defer func() {
		if !created {
			p.mu.Lock()
			p.total--
			p.signalLocked()
			p.mu.Unlock()
		}
	}()
	vm, e := p.newVM(ctx)
	created = e == nil
	return vm, e
}

func (p *VMPool) Return(_ context.Context, vm *VM) error {
	if vm == nil {
		return errors.New("cannot return a nil VM")
	}
	p.mu.Lock()
	borrowed, belongs := p.members[vm]
	if !belongs || !borrowed {
		p.mu.Unlock()
		return errors.New("VM does not belong to this pool or was already returned")
	}
	p.members[vm] = false
	p.mu.Unlock()

	cleanupErr := vm.disposeDisposables()

	p.mu.Lock()
	if p.closed || !vm.Reusable() || len(p.idle) >= p.config.MaxIdle {
		delete(p.members, vm)
		p.total--
		p.signalLocked()
		p.mu.Unlock()
		return errors.Join(cleanupErr, vm.Dispose())
	}
	p.idle = append(p.idle, idleVM{vm: vm, returnedAt: time.Now()})
	p.signalLocked()
	p.mu.Unlock()
	return cleanupErr
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
	for _, item := range idle {
		delete(p.members, item.vm)
	}
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
	clear(p.idle[len(p.idle)-removeCount:])
	p.idle = p.idle[:len(p.idle)-removeCount]
	p.total -= removeCount
	for _, item := range expired {
		delete(p.members, item.vm)
	}
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
