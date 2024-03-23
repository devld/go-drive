package script

import (
	"context"
	"time"

	pool "github.com/jolestar/go-commons-pool/v2"
)

type VMPoolConfig struct {
	MaxTotal int
	MaxIdle  int
	MinIdle  int
	IdleTime time.Duration
}

func NewVMPool(baseVM *VM, config *VMPoolConfig) *VMPool {
	conf := pool.NewDefaultPoolConfig()
	conf.MaxTotal = config.MaxTotal
	conf.MaxIdle = config.MaxIdle
	conf.MinIdle = config.MinIdle
	conf.MinEvictableIdleTime = config.IdleTime

	if conf.MaxIdle < conf.MinIdle {
		panic("MaxIdle must be greater than or equal to MinIdle")
	}

	p := pool.NewObjectPool(
		context.Background(),
		&poolObjectFactory{baseVM},
		conf,
	)
	return &VMPool{p}
}

type VMPool struct {
	pool *pool.ObjectPool
}

func (p *VMPool) Get(ctx context.Context) (*VM, error) {
	vm, e := p.pool.BorrowObject(ctx)
	if e != nil {
		return nil, e
	}
	return vm.(*VM), nil
}

func (p *VMPool) Return(ctx context.Context, vm *VM) error {
	return p.pool.ReturnObject(ctx, vm)
}

func (p *VMPool) Dispose() error {
	p.pool.Close(context.Background())
	return nil
}

type poolObjectFactory struct {
	base *VM
}

func (pof *poolObjectFactory) MakeObject(ctx context.Context) (*pool.PooledObject, error) {
	return pool.NewPooledObject(pof.base.Fork()), nil
}

func (pof *poolObjectFactory) DestroyObject(ctx context.Context, object *pool.PooledObject) error {
	vm := object.Object.(*VM)
	return vm.Dispose()
}

func (pof *poolObjectFactory) ActivateObject(ctx context.Context, object *pool.PooledObject) error {
	return nil
}

func (pof *poolObjectFactory) PassivateObject(ctx context.Context, object *pool.PooledObject) error {
	object.Object.(*VM).DisposeDisposables()
	return nil
}

func (pof *poolObjectFactory) ValidateObject(ctx context.Context, object *pool.PooledObject) bool {
	return true
}

var _ pool.PooledObjectFactory = (*poolObjectFactory)(nil)
