package artifact

import (
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/registry"
	"go-drive/common/types"
)

// Registration describes one artifact type exposed by a Handler. The handler
// itself is the Processor for every type it registers.
type Registration struct {
	Type      ArtifactType
	Policy    Policy
	TaskGroup string
	// Concurrency is the optional limit for TaskGroup. Zero uses the runner's
	// global limit.
	Concurrency int
}

// Processor produces one artifact type. Persistence, locking, failure cache,
// and task scheduling stay in Service. Wrap Produce errors with Cacheable
// when a failed attempt should be stored so later lookups return not-found.
// Produce must call Writer.WriteMeta with Name, MimeType, and ModTime
// before writing the body. Type, fingerprint, and created time are filled
// by Store.
type Processor interface {
	Identity(ArtifactRequest) (Identity, error)
	Produce(types.TaskCtx, ArtifactRequest, Writer) error
}

type Writer = driveutil.Writer[Meta]

// Handler is a constructed artifact plugin and the Processor for every type
// it registers. One handler may expose several artifact types; archive does
// this for its index and content outputs.
type Handler interface {
	Processor
	Registrations() []Registration
	// Config is the client-facing entry under artifact in /config. The
	// RegisterHandler name is the key. A nil map means this handler does not
	// publish configuration.
	Config() types.M
}

// HandlerContext is the shared constructor input used when the artifact
// service creates registered handlers.
type HandlerContext struct {
	Config     common.Config
	Store      *Store
	Service    *Service
	Components *registry.ComponentsHolder
}

// HandlerFactory constructs a handler from the shared artifact runtime.
type HandlerFactory func(HandlerContext) (Handler, error)

type handlerFactoryEntry struct {
	name    string
	factory HandlerFactory
}

var handlerFactories []handlerFactoryEntry

// RegisterHandler registers a named artifact handler factory. A later call
// with the same name replaces the previous factory. Factories must be
// registered before NewService; init() is the expected call site.
func RegisterHandler(name string, factory HandlerFactory) {
	if name == "" {
		panic("artifact handler name is required")
	}
	if factory == nil {
		panic("artifact handler factory " + name + " is nil")
	}
	for i, registered := range handlerFactories {
		if registered.name == name {
			handlerFactories[i].factory = factory
			return
		}
	}
	handlerFactories = append(handlerFactories, handlerFactoryEntry{name: name, factory: factory})
}
