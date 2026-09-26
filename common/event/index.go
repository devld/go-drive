package event

import (
	"go-drive/common/types"
	"sync"
)

type Unsubscribe func()

type EntryAccessedHandler func(types.DriveEvent, string)
type EntryUpdatedHandler func(types.DriveEvent, string, bool)
type EntryDeletedHandler func(types.DriveEvent, string)

// Bus is a synchronous, strongly typed in-process event bus. Publish takes a
// snapshot of subscribers before invoking them, so handlers may safely publish
// another event or unsubscribe without deadlocking the bus.
type Bus interface {
	PublishEntryAccessed(types.DriveEvent, string)
	PublishEntryUpdated(types.DriveEvent, string, bool)
	PublishEntryDeleted(types.DriveEvent, string)
	SubscribeEntryAccessed(EntryAccessedHandler) Unsubscribe
	SubscribeEntryUpdated(EntryUpdatedHandler) Unsubscribe
	SubscribeEntryDeleted(EntryDeletedHandler) Unsubscribe
}

func NewBus() Bus {
	b := &bus{}
	b.accessed.init()
	b.updated.init()
	b.deleted.init()
	return b
}

type bus struct {
	accessed subscriptions[EntryAccessedHandler]
	updated  subscriptions[EntryUpdatedHandler]
	deleted  subscriptions[EntryDeletedHandler]
}

func (b *bus) PublishEntryAccessed(event types.DriveEvent, path string) {
	for _, handler := range b.accessed.snapshot() {
		handler(event, path)
	}
}

func (b *bus) PublishEntryUpdated(event types.DriveEvent, path string, includeDescendants bool) {
	for _, handler := range b.updated.snapshot() {
		handler(event, path, includeDescendants)
	}
}

func (b *bus) PublishEntryDeleted(event types.DriveEvent, path string) {
	for _, handler := range b.deleted.snapshot() {
		handler(event, path)
	}
}

func (b *bus) SubscribeEntryAccessed(handler EntryAccessedHandler) Unsubscribe {
	return b.accessed.subscribe(handler)
}

func (b *bus) SubscribeEntryUpdated(handler EntryUpdatedHandler) Unsubscribe {
	return b.updated.subscribe(handler)
}

func (b *bus) SubscribeEntryDeleted(handler EntryDeletedHandler) Unsubscribe {
	return b.deleted.subscribe(handler)
}

type subscriptions[T any] struct {
	mu       sync.RWMutex
	nextID   uint64
	handlers map[uint64]T
	order    []uint64
}

func (s *subscriptions[T]) init() {
	s.handlers = make(map[uint64]T)
}

func (s *subscriptions[T]) subscribe(handler T) Unsubscribe {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.handlers[id] = handler
	s.order = append(s.order, id)
	s.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.handlers, id)
			for i, registeredID := range s.order {
				if registeredID == id {
					s.order = append(s.order[:i], s.order[i+1:]...)
					break
				}
			}
			s.mu.Unlock()
		})
	}
}

func (s *subscriptions[T]) snapshot() []T {
	s.mu.RLock()
	handlers := make([]T, 0, len(s.order))
	for _, id := range s.order {
		handlers = append(handlers, s.handlers[id])
	}
	s.mu.RUnlock()
	return handlers
}
