package event

import (
	eb "github.com/asaskevich/EventBus"
	"go-drive/common/registry"
)

// Bus is the event bus, all events are published on this bus
// all event handlers must return as soon as possible
// long-running event handlers should be run in task.Runner
type Bus interface {
	Publish(topic string, args ...interface{})
	Subscribe(topic string, handler interface{})
	SubscribeOnce(topic string, handler interface{})
	Unsubscribe(topic string, handler interface{}) bool
}

func NewBus(ch *registry.ComponentsHolder) Bus {
	b := &bus{eb.New()}
	ch.Add("eventBus", b)
	return b
}

type bus struct {
	b eb.Bus
}

func (b *bus) Publish(topic string, args ...interface{}) {
	b.b.Publish(topic, args...)
}

func (b *bus) Subscribe(topic string, handler interface{}) {
	if e := b.b.Subscribe(topic, handler); e != nil {
		panic(e)
	}
}

func (b *bus) SubscribeOnce(topic string, handler interface{}) {
	if e := b.b.SubscribeOnce(topic, handler); e != nil {
		panic(e)
	}
}

func (b *bus) Unsubscribe(topic string, handler interface{}) bool {
	return b.b.Unsubscribe(topic, handler) == nil
}
