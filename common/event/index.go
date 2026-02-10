package event

import (
	"go-drive/common/registry"

	eb "github.com/asaskevich/EventBus"
)

// Bus is the event bus, all events are published on this bus
// all event handlers must return as soon as possible
// long-running event handlers should be run in task.Runner
type Bus interface {
	Publish(topic string, args ...any)
	Subscribe(topic string, handler any)
	SubscribeOnce(topic string, handler any)
	Unsubscribe(topic string, handler any) bool
}

func NewBus(ch *registry.ComponentsHolder) Bus {
	b := &bus{eb.New()}
	ch.Add("eventBus", b)
	return b
}

type bus struct {
	b eb.Bus
}

func (b *bus) Publish(topic string, args ...any) {
	b.b.Publish(topic, args...)
}

func (b *bus) Subscribe(topic string, handler any) {
	if e := b.b.Subscribe(topic, handler); e != nil {
		panic(e)
	}
}

func (b *bus) SubscribeOnce(topic string, handler any) {
	if e := b.b.SubscribeOnce(topic, handler); e != nil {
		panic(e)
	}
}

func (b *bus) Unsubscribe(topic string, handler any) bool {
	return b.b.Unsubscribe(topic, handler) == nil
}
