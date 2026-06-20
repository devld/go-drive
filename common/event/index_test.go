package event

import (
	"go-drive/common/registry"
	"go-drive/common/types"
	"sync"
	"sync/atomic"
	"testing"
)

func newTestBus() Bus {
	return NewBus(registry.NewComponentHolder())
}

func TestBusPublishesTypedEventsAndUnsubscribes(t *testing.T) {
	bus := newTestBus()
	var calls atomic.Int32
	unsubscribe := bus.SubscribeEntryUpdated(func(_ types.DriveListenerContext, path string, descendants bool) {
		if path != "/file" || !descendants {
			t.Errorf("unexpected event: path=%q descendants=%v", path, descendants)
		}
		calls.Add(1)
	})

	bus.PublishEntryUpdated(types.DriveListenerContext{}, "/file", true)
	unsubscribe()
	unsubscribe()
	bus.PublishEntryUpdated(types.DriveListenerContext{}, "/file", true)
	if calls.Load() != 1 {
		t.Fatalf("handler called %d times", calls.Load())
	}
}

func TestBusAllowsReentrantPublishAndUnsubscribe(t *testing.T) {
	bus := newTestBus()
	var updatedCalls, deletedCalls atomic.Int32
	var unsubscribe Unsubscribe
	unsubscribe = bus.SubscribeEntryUpdated(func(ctx types.DriveListenerContext, path string, _ bool) {
		updatedCalls.Add(1)
		unsubscribe()
		bus.PublishEntryDeleted(ctx, path)
	})
	bus.SubscribeEntryDeleted(func(_ types.DriveListenerContext, _ string) {
		deletedCalls.Add(1)
	})

	bus.PublishEntryUpdated(types.DriveListenerContext{}, "/file", false)
	bus.PublishEntryUpdated(types.DriveListenerContext{}, "/file", false)
	if updatedCalls.Load() != 1 || deletedCalls.Load() != 1 {
		t.Fatalf("unexpected calls: updated=%d deleted=%d", updatedCalls.Load(), deletedCalls.Load())
	}
}

func TestBusConcurrentPublishAndSubscribe(t *testing.T) {
	bus := newTestBus()
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			unsubscribe := bus.SubscribeEntryAccessed(func(types.DriveListenerContext, string) {})
			unsubscribe()
		}()
		go func() {
			defer wg.Done()
			bus.PublishEntryAccessed(types.DriveListenerContext{}, "/file")
		}()
	}
	wg.Wait()
}

func TestBusPreservesSubscriptionOrder(t *testing.T) {
	bus := newTestBus()
	order := make([]int, 0, 2)
	bus.SubscribeEntryDeleted(func(types.DriveListenerContext, string) { order = append(order, 1) })
	bus.SubscribeEntryDeleted(func(types.DriveListenerContext, string) { order = append(order, 2) })
	bus.PublishEntryDeleted(types.DriveListenerContext{}, "/file")
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("unexpected handler order: %v", order)
	}
}
