package artifact

import (
	"context"
	"errors"
	"fmt"
	apierr "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/task"
	"go-drive/common/types"
	"maps"
	"time"
)

var artifactLog = logging.For("artifact")

type registeredType struct {
	Registration
	handler Handler
}

type installedHandler struct {
	name    string
	handler Handler
}

// Service is the shared artifact runtime. It registers processors, persists
// complete outputs through Store, and schedules generation on the task runner.
type Service struct {
	store  *Store
	runner task.Runner

	registrations map[ArtifactType]registeredType
	taskGroups    map[string]int
	handlers      []installedHandler
}

var _ types.ISysConfig = (*Service)(nil)

func NewService(store *Store, runner task.Runner, ctx HandlerContext, handlers ...Handler) (*Service, error) {
	if store == nil {
		return nil, errors.New("artifact store is required")
	}
	service := &Service{
		store:         store,
		runner:        runner,
		registrations: make(map[ArtifactType]registeredType),
		taskGroups:    make(map[string]int),
	}
	ctx.Store = store
	ctx.Service = service
	for _, entry := range handlerFactories {
		handler, err := entry.factory(ctx)
		if err != nil {
			return nil, fmt.Errorf("create artifact handler %q: %w", entry.name, err)
		}
		if err := service.install(entry.name, handler); err != nil {
			return nil, fmt.Errorf("install artifact handler %q: %w", entry.name, err)
		}
	}
	for _, handler := range handlers {
		if err := service.install("", handler); err != nil {
			return nil, err
		}
	}
	return service, nil
}

func (s *Service) install(name string, handler Handler) error {
	if handler == nil {
		return errors.New("artifact handler is nil")
	}
	registrations := handler.Registrations()
	if len(registrations) == 0 {
		return errors.New("artifact handler has no registrations")
	}
	for _, registration := range registrations {
		if err := s.register(handler, registration); err != nil {
			return err
		}
	}
	s.handlers = append(s.handlers, installedHandler{name: name, handler: handler})
	return nil
}

func (s *Service) register(handler Handler, registration Registration) error {
	if registration.Concurrency > 0 && registration.TaskGroup == "" {
		return fmt.Errorf("artifact task group for %q is required when concurrency is set", registration.Type)
	}
	if _, exists := s.registrations[registration.Type]; exists {
		return fmt.Errorf("artifact type %q is already registered", registration.Type)
	}
	if registration.Concurrency > 0 {
		if concurrency, exists := s.taskGroups[registration.TaskGroup]; exists && concurrency != registration.Concurrency {
			return fmt.Errorf("artifact task group %q has conflicting concurrency %d and %d",
				registration.TaskGroup, concurrency, registration.Concurrency)
		}
	}
	if err := s.store.registerType(registration.Type, registration.Policy); err != nil {
		return err
	}
	if registration.Concurrency > 0 {
		_, alreadyRegistered := s.taskGroups[registration.TaskGroup]
		if s.runner != nil && !alreadyRegistered {
			if err := s.runner.RegisterGroup(registration.TaskGroup, registration.Concurrency); err != nil {
				return fmt.Errorf("register artifact task group %q: %w", registration.TaskGroup, err)
			}
		}
		s.taskGroups[registration.TaskGroup] = registration.Concurrency
	}
	s.registrations[registration.Type] = registeredType{
		Registration: registration,
		handler:      handler,
	}
	return nil
}

func (s *Service) resolve(typ ArtifactType) (registeredType, error) {
	if s == nil {
		return registeredType{}, apierr.NewNotFoundMessageError("artifact preview is unavailable")
	}
	registration, ok := s.registrations[typ]
	if !ok {
		return registeredType{}, apierr.NewNotFoundMessageError("invalid artifact type")
	}
	return registration, nil
}

// GetResult is the outcome of Service.Get. Artifact is set when a body is
// ready to stream. Task is set when generation is still running after the wait.
type GetResult struct {
	Artifact *Artifact
	Task     task.Task
}

// EnsureResult is the outcome of Service.Ensure. Info is set when the artifact
// is already stored. Task is set when generation is still running after the wait.
type EnsureResult struct {
	Info Info
	Task task.Task
}

// Get serves a cached body when possible, otherwise generates and waits up to
// wait. A still-pending task is returned so HTTP GET can treat it as failure
// while generation continues in the background.
func (s *Service) Get(ctx context.Context, request ArtifactRequest, wait time.Duration) (GetResult, error) {
	registration, identity, created, err := s.schedule(ctx, request, wait)
	if err != nil {
		return GetResult{}, err
	}
	if created.Id == "" {
		file, err := s.openIdentity(request.Type, identity, registration.Policy.TTL)
		return GetResult{Artifact: file}, err
	}
	if created.Status == task.Done {
		file, err := s.openIdentity(request.Type, identity, registration.Policy.TTL)
		return GetResult{Artifact: file}, err
	}
	if created.Status == task.Error {
		if created.Error != nil {
			return GetResult{}, created.Error
		}
		return GetResult{}, errors.New("artifact preview task failed")
	}
	return GetResult{Task: created}, nil
}

// Ensure looks up a ready artifact or starts generation. It never opens the
// body. A ready cache hit or a task that finishes within wait returns Info.
func (s *Service) Ensure(ctx context.Context, request ArtifactRequest, wait time.Duration) (EnsureResult, error) {
	registration, identity, created, err := s.schedule(ctx, request, wait)
	if err != nil {
		return EnsureResult{}, err
	}
	if created.Id == "" {
		info, err := s.lookupInfo(request.Type, identity, registration.Policy.TTL)
		return EnsureResult{Info: info}, err
	}
	if created.Status == task.Done {
		info, ok := created.Result.(Info)
		if !ok {
			info, err = s.lookupInfo(request.Type, identity, registration.Policy.TTL)
			if err != nil {
				return EnsureResult{}, err
			}
		}
		return EnsureResult{Info: info}, nil
	}
	if created.Status == task.Error {
		if created.Error != nil {
			return EnsureResult{}, created.Error
		}
		return EnsureResult{}, errors.New("artifact preview task failed")
	}
	return EnsureResult{Task: created}, nil
}

func (s *Service) schedule(ctx context.Context, request ArtifactRequest, wait time.Duration) (Registration, Identity, task.Task, error) {
	registration, processor, err := s.processor(request.Type)
	if err != nil {
		return Registration{}, Identity{}, task.Task{}, err
	}
	identity, err := processor.Identity(request)
	if err != nil {
		return Registration{}, Identity{}, task.Task{}, err
	}
	_, err = s.lookupInfo(request.Type, identity, registration.Policy.TTL)
	if err == nil {
		return registration, identity, task.Task{}, nil
	}
	if !errors.Is(err, errCacheMiss) {
		return registration, identity, task.Task{}, err
	}
	if s.runner == nil {
		return Registration{}, Identity{}, task.Task{}, errors.New("artifact task runner is required")
	}
	created, err := s.runner.ExecuteAndWait(ctx, func(taskCtx types.TaskCtx) (any, error) {
		return s.Generate(taskCtx, request)
	}, wait, task.WithNameGroup(taskName(request), registration.TaskGroup))
	if err != nil {
		return Registration{}, Identity{}, task.Task{}, err
	}
	return registration, identity, created, nil
}

// Generate materializes an artifact on the current task.
func (s *Service) Generate(ctx types.TaskCtx, request ArtifactRequest) (Info, error) {
	registration, processor, err := s.processor(request.Type)
	if err != nil {
		return Info{}, err
	}
	identity, err := processor.Identity(request)
	if err != nil {
		return Info{}, err
	}
	return s.generateLocked(ctx, request, registration, processor, identity)
}

// Open streams a previously published artifact after checking it still
// belongs to this source identity.
func (s *Service) Open(request ArtifactRequest) (*Artifact, error) {
	registration, processor, err := s.processor(request.Type)
	if err != nil {
		return nil, err
	}
	identity, err := processor.Identity(request)
	if err != nil {
		return nil, err
	}
	return s.openIdentity(request.Type, identity, registration.Policy.TTL)
}

func (s *Service) generateLocked(ctx types.TaskCtx, request ArtifactRequest, registration Registration, processor Processor, identity Identity) (Info, error) {
	ttl := registration.Policy.TTL
	if info, err := s.lookupInfo(request.Type, identity, ttl); !errors.Is(err, errCacheMiss) {
		return info, err
	}
	unlock, err := s.store.Lock(request.Type, identity.Key)
	if err != nil {
		return Info{}, err
	}
	defer unlock()
	if info, err := s.lookupInfo(request.Type, identity, ttl); !errors.Is(err, errCacheMiss) {
		return info, err
	}
	writer, err := s.store.Create(request.Type, identity.Key, identity.Fingerprint)
	if err != nil {
		return Info{}, err
	}
	defer writer.Abort()
	if err := processor.Produce(ctx, request, writer); err != nil {
		return s.finishProduce(request.Type, identity, err)
	}
	if err := writer.Close(); err != nil {
		return s.finishProduce(request.Type, identity, err)
	}
	return writer.Info(), nil
}

func (s *Service) finishProduce(typ ArtifactType, identity Identity, err error) (Info, error) {
	s.storeFailure(typ, identity, err)
	if cached, ok := errors.AsType[cacheableError](err); ok {
		return Info{}, cached.error
	}
	return Info{}, err
}

func (s *Service) lookupInfo(typ ArtifactType, identity Identity, ttl time.Duration) (Info, error) {
	return s.store.Lookup(typ, identity.Key, identity.Fingerprint, ttl)
}

func (s *Service) openIdentity(typ ArtifactType, identity Identity, ttl time.Duration) (*Artifact, error) {
	if _, err := s.lookupInfo(typ, identity, ttl); err != nil {
		if errors.Is(err, errCacheMiss) {
			return nil, apierr.NewNotFoundMessageError("artifact not found")
		}
		return nil, err
	}
	return s.store.Open(typ, identity.Key)
}

func (s *Service) storeFailure(typ ArtifactType, identity Identity, err error) {
	if err == nil || !IsCacheable(err) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	if writeErr := s.store.WriteFailure(typ, identity.Key, identity.Fingerprint); writeErr != nil {
		artifactLog.Warnf("write artifact failure record type=%s: %v", typ, writeErr)
	}
}

func (s *Service) processor(typ ArtifactType) (Registration, Processor, error) {
	registration, err := s.resolve(typ)
	if err != nil {
		return Registration{}, nil, err
	}
	return registration.Registration, registration.handler, nil
}

func taskName(request ArtifactRequest) string {
	if request.Source == nil {
		return string(request.Type)
	}
	path := request.Source.Path()
	if request.Args == "" {
		return path
	}
	return path + "#" + request.Args
}

// SysConfig exposes live handler configuration through the single artifact
// entry in /config.
func (s *Service) SysConfig() (string, types.M, error) {
	config := make(types.M, len(s.handlers))
	for _, installed := range s.handlers {
		values := installed.handler.Config()
		if values == nil {
			continue
		}
		config[installed.name] = cloneConfig(values)
	}
	return "artifact", config, nil
}

// TODO: remove
func cloneConfig(values types.M) types.M {
	clone := make(types.M, len(values))
	maps.Copy(clone, values)
	return clone
}
