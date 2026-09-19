package artifact

import (
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/types"
)

// CacheSpec is one persistence bucket. Name is ResolvedRequest.Cache. An empty
// name is the default bucket (directory handler) and is enough when the
// handler has only one cache. Named caches use handler-<name>. TTL and
// MaxBytes apply only to that bucket.
type CacheSpec struct {
	Name   string
	Policy Policy
}

// Spec declares this handler's caches, optional task concurrency, and
// client-facing /config entry. Service always runs generation in group
// artifact/<RegisterHandler name>. RegisterGroup is only called when
// Concurrency is greater than zero. Caches must contain at least one
// CacheSpec. A nil Config means this handler does not publish configuration.
// The RegisterHandler name is the /config key.
type Spec struct {
	Caches []CacheSpec
	// Concurrency is the optional limit for this handler's task group. Zero
	// uses the runner's global limit.
	Concurrency int
	Config      types.M
}

type Writer = driveutil.Writer[Meta]

type Request struct {
	Source  types.IEntry
	Handler string
	Args    string
}

// ResolvedRequest is Handler.Resolve's result: which Spec.Caches bucket, and
// the handler's own key fragment and generator identity. Service prepends the
// source entry's stable slot (real path) and changing identity (path,
// type, size, mtime) before talking to Store. Handler Key is extra slot
// identity that is not the source entry, such as an archive member. Handler
// Fingerprint is generator version and settings (for example v1). Cache
// selects a Spec.Caches bucket; empty uses the handler's only cache.
type ResolvedRequest struct {
	Key         string
	Fingerprint string
	Cache       string
}

// Handler is a constructed artifact plugin. Persistence, locking, failure
// cache, and task scheduling stay in Service. Wrap Produce errors with
// Cacheable when a failed attempt should be stored so later lookups return
// not-found. Produce must call Writer.WriteMeta with Name, MimeType, and
// ModTime before writing the body. ResolvedRequest.Cache selects a registered
// cache. Store records the service-composed fingerprint and created time.
type Handler interface {
	Resolve(Request) (ResolvedRequest, error)
	Produce(types.TaskCtx, Request, Writer) error
	Spec() Spec
}

// Cache reads a published artifact produced by this handler, without starting
// generation. A miss or unusable record is reported as not-found. The lookup
// is scoped to the handler that received this Cache; it cannot name another
// handler.
type Cache interface {
	Get(source types.IEntry, args string) (*Artifact, error)
}

// HandlerContext is the shared constructor input used when the artifact
// service creates registered handlers.
type HandlerContext struct {
	Config common.Config
	// Cache is bound to this handler's name by Service. Get must not
	// schedule Produce.
	Cache Cache
}

// HandlerFactory constructs a handler from the shared artifact runtime.
type HandlerFactory func(HandlerContext) (Handler, error)

type handlerFactoryEntry struct {
	name    string
	factory HandlerFactory
}

var handlerFactories []handlerFactoryEntry

// RegisterHandler registers a named artifact handler factory. A later call
// with the same name panics. Factories must be registered before NewService;
// init() is the expected call site.
func RegisterHandler(name string, factory HandlerFactory) {
	if factory == nil {
		panic("artifact handler factory " + name + " is nil")
	}
	for _, registered := range handlerFactories {
		if registered.name == name {
			panic("artifact handler " + name + " already registered")
		}
	}
	handlerFactories = append(handlerFactories, handlerFactoryEntry{name: name, factory: factory})
}
