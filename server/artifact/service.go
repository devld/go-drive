package artifact

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var artifactLog = logging.For("artifact")

// cleanInterval is how often expired artifacts are removed from disk. Pack
// downloads stay fetchable for archive.pack-ttl (default one minute) even
// when this scan has not run yet. Cleanup skips artifacts that still have a
// reader.
const cleanInterval = 30 * time.Minute

// Service is the shared artifact runtime. It registers processors, persists
// complete outputs through Store, and schedules generation on the task runner.
type Service struct {
	store  *Store
	runner task.Runner

	handlers    map[string]Handler
	stopCleaner func()
}

var (
	_ types.ISysConfig  = (*Service)(nil)
	_ types.IDisposable = (*Service)(nil)
)

func NewService(config common.Config, runner task.Runner, options OptionReader, files *driveutil.DriveFS) (*Service, error) {
	tempDir := config.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	store, err := NewStore(filepath.Join(tempDir, "artifacts"))
	if err != nil {
		return nil, fmt.Errorf("create artifact store: %w", err)
	}
	service := &Service{
		store:    store,
		runner:   runner,
		handlers: make(map[string]Handler),
	}
	for _, entry := range handlerFactories {
		handlerDeps := HandlerDeps{
			Config:  config,
			Options: options,
			Cache:   handlerCache{service: service, handler: entry.name},
			DriveFS: files,
		}
		handler, err := entry.factory(handlerDeps)
		if err != nil {
			return nil, fmt.Errorf("create artifact handler %q: %w", entry.name, err)
		}
		if err := service.install(entry.name, handler); err != nil {
			return nil, fmt.Errorf("install artifact handler %q: %w", entry.name, err)
		}
	}
	service.stopCleaner = utils.TimeTick(service.clean, cleanInterval)
	return service, nil
}

func (s *Service) install(name string, handler Handler) error {
	if handler == nil {
		return errors.New("artifact handler is nil")
	}
	if _, exists := s.handlers[name]; exists {
		return fmt.Errorf("artifact handler %q is already registered", name)
	}
	if err := s.register(name, handler.Spec()); err != nil {
		return err
	}
	s.handlers[name] = handler
	return nil
}

func (s *Service) register(name string, spec Spec) error {
	for _, cache := range spec.Caches {
		if err := s.store.registerType(cacheBucket(name, cache.Name), cache.Policy); err != nil {
			return err
		}
	}
	if spec.Concurrency > 0 && s.runner != nil {
		group := taskGroup(name)
		if err := s.runner.RegisterGroup(group, spec.Concurrency); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) resolveHandler(name string) (Handler, error) {
	handler, ok := s.handlers[name]
	if !ok {
		return nil, apierr.NewNotFoundMessageError("unknown artifact handler")
	}
	return handler, nil
}

// FetchResult is the outcome of Service.Fetch. Artifact is set when a body is
// ready to stream. Task is set when generation is still running after the wait.
type FetchResult struct {
	Artifact *Artifact
	Task     *task.Task
}

// ArtifactInfo is the client-facing description of a ready artifact. Info is
// embedded so JSON serialization inlines the stored fields next to ref.
type ArtifactInfo struct {
	Info
	// Ref retrieves this cached artifact with GET ?ref=. It does not repeat
	// the request args and does not start generation.
	Ref string `json:"ref"`
}

// PrepareResult is the outcome of Service.Prepare. Info is set when the
// artifact is already stored. Task is set when generation is still running
// after the wait.
type PrepareResult struct {
	Info ArtifactInfo
	Task *task.Task
}

// Fetch serves a cached body when possible, otherwise generates and waits up
// to wait. A still-pending task is returned so HTTP GET can treat it as
// failure while generation continues in the background.
func (s *Service) Fetch(ctx context.Context, request Request, wait time.Duration) (FetchResult, error) {
	got, err := s.schedule(ctx, request, wait)
	if err != nil {
		return FetchResult{}, err
	}
	if got.task != nil {
		return FetchResult{Task: got.task}, nil
	}
	file, err := s.openResolved(request.Handler, got.spec, got.resolved)
	return FetchResult{Artifact: file}, err
}

// Prepare looks up a ready artifact or starts generation. It never opens the
// body. A ready cache hit or a task that finishes within wait returns Info.
func (s *Service) Prepare(ctx context.Context, request Request, wait time.Duration) (PrepareResult, error) {
	got, err := s.schedule(ctx, request, wait)
	if err != nil {
		return PrepareResult{}, err
	}
	if got.task != nil {
		return PrepareResult{Task: got.task}, nil
	}
	return PrepareResult{Info: got.info}, nil
}

// OpenCached returns a stored artifact by ref. A miss or expiry is not-found
// and does not start generation. The ref is cacheName:key for this handler.
// source supplies the real path that was stored with that fragment. Validity
// and expiry are decided by the store.
func (s *Service) OpenCached(handlerName string, source types.IEntry, ref string) (*Artifact, error) {
	handler, err := s.resolveHandler(handlerName)
	if err != nil {
		return nil, err
	}
	resolved, err := resolveRef(handler, source, ref)
	if err != nil {
		return nil, err
	}
	bucket, _, err := bindCache(handler.Spec(), handlerName, resolved)
	if err != nil {
		return nil, apierr.NewNotFoundMessageError("artifact not found")
	}
	return s.store.Open(bucket, resolved.fullKey)
}

type scheduled struct {
	spec     Spec
	resolved resolvedRequest
	info     ArtifactInfo
	task     *task.Task
}

func (s *Service) schedule(ctx context.Context, request Request, wait time.Duration) (scheduled, error) {
	handler, err := s.resolveHandler(request.Handler)
	if err != nil {
		return scheduled{}, err
	}
	resolved, err := resolveRequest(handler, request)
	if err != nil {
		return scheduled{}, err
	}
	spec := handler.Spec()
	info, err := s.lookupInfo(request.Handler, spec, resolved)
	if err == nil {
		return scheduled{spec: spec, resolved: resolved, info: info}, nil
	}
	if !errors.Is(err, errCacheMiss) {
		return scheduled{}, err
	}
	created, err := s.runner.ExecuteAndWait(ctx, func(taskCtx types.TaskCtx) (any, error) {
		return s.generateLocked(taskCtx, request, handler, resolved)
	}, wait, task.WithName(taskName(request)), task.WithGroup(taskGroup(request.Handler)))
	if err != nil {
		return scheduled{}, err
	}
	switch created.Status {
	case task.Done:
		info, ok := created.Result.(ArtifactInfo)
		if !ok {
			info, err = s.lookupInfo(request.Handler, spec, resolved)
			if err != nil {
				return scheduled{}, err
			}
		}
		return scheduled{spec: spec, resolved: resolved, info: info}, nil
	case task.Error:
		if created.Error != nil {
			return scheduled{}, created.Error
		}
		return scheduled{}, errors.New("artifact preview task failed")
	default:
		return scheduled{spec: spec, resolved: resolved, task: &created}, nil
	}
}

func (s *Service) open(request Request) (*Artifact, error) {
	handler, err := s.resolveHandler(request.Handler)
	if err != nil {
		return nil, err
	}
	resolved, err := resolveRequest(handler, request)
	if err != nil {
		return nil, err
	}
	return s.openResolved(request.Handler, handler.Spec(), resolved)
}

func (s *Service) generateLocked(ctx types.TaskCtx, request Request, handler Handler, resolved resolvedRequest) (ArtifactInfo, error) {
	spec := handler.Spec()
	bucket, _, err := bindCache(spec, request.Handler, resolved)
	if err != nil {
		return ArtifactInfo{}, err
	}
	if info, err := s.lookupInfo(request.Handler, spec, resolved); !errors.Is(err, errCacheMiss) {
		return info, err
	}
	unlock, err := s.store.Lock(bucket, resolved.fullKey)
	if err != nil {
		return ArtifactInfo{}, err
	}
	defer unlock()
	if info, err := s.lookupInfo(request.Handler, spec, resolved); !errors.Is(err, errCacheMiss) {
		return info, err
	}
	writer, err := s.store.Create(bucket, resolved.fullKey, resolved.fullFingerprint)
	if err != nil {
		return ArtifactInfo{}, err
	}
	defer writer.Abort()
	if err := handler.Produce(ctx, request, writer); err != nil {
		return s.finishProduce(request.Handler, spec, resolved, err)
	}
	if err := writer.Close(); err != nil {
		return s.finishProduce(request.Handler, spec, resolved, err)
	}
	return withRef(writer.info(), resolved.Cache, resolved.Key), nil
}

func (s *Service) finishProduce(handler string, spec Spec, resolved resolvedRequest, err error) (ArtifactInfo, error) {
	s.storeFailure(handler, spec, resolved, err)
	if cached, ok := errors.AsType[cacheableError](err); ok {
		return ArtifactInfo{}, cached.error
	}
	return ArtifactInfo{}, err
}

func (s *Service) lookupInfo(handler string, spec Spec, resolved resolvedRequest) (ArtifactInfo, error) {
	bucket, policy, err := bindCache(spec, handler, resolved)
	if err != nil {
		return ArtifactInfo{}, err
	}
	info, err := s.store.Lookup(bucket, resolved.fullKey, resolved.fullFingerprint, policy.TTL)
	if err != nil {
		return ArtifactInfo{}, err
	}
	return withRef(info, resolved.Cache, resolved.Key), nil
}

func (s *Service) openResolved(handler string, spec Spec, resolved resolvedRequest) (*Artifact, error) {
	bucket, _, err := bindCache(spec, handler, resolved)
	if err != nil {
		return nil, err
	}
	if _, err := s.lookupInfo(handler, spec, resolved); err != nil {
		if errors.Is(err, errCacheMiss) {
			return nil, apierr.NewNotFoundMessageError("artifact not found")
		}
		return nil, err
	}
	return s.store.Open(bucket, resolved.fullKey)
}

func (s *Service) storeFailure(handler string, spec Spec, resolved resolvedRequest, err error) {
	if err == nil || !IsCacheable(err) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	bucket, _, bindErr := bindCache(spec, handler, resolved)
	if bindErr != nil {
		artifactLog.Warnf("write artifact failure record handler=%s: %v", handler, bindErr)
		return
	}
	if writeErr := s.store.WriteFailure(bucket, resolved.fullKey, resolved.fullFingerprint); writeErr != nil {
		artifactLog.Warnf("write artifact failure record handler=%s: %v", handler, writeErr)
	}
}

type handlerCache struct {
	service *Service
	handler string
}

func (c handlerCache) Get(source types.IEntry, args string) (*Artifact, error) {
	return c.service.open(Request{
		Source:  source,
		Handler: c.handler,
		Args:    args,
	})
}

func resolveRequest(handler Handler, request Request) (resolvedRequest, error) {
	handlerResolved, err := handler.Resolve(request)
	if err != nil {
		return resolvedRequest{}, err
	}
	return bindSource(request.Source, handlerResolved), nil
}

func resolveRef(handler Handler, source types.IEntry, ref string) (resolvedRequest, error) {
	cache, key, ok := strings.Cut(ref, ":")
	if !ok {
		return resolvedRequest{}, apierr.NewNotFoundMessageError("artifact not found")
	}
	if _, _, err := resolveCache(handler.Spec(), cache); err != nil {
		return resolvedRequest{}, apierr.NewNotFoundMessageError("artifact not found")
	}
	return bindSource(source, ResolvedRequest{Cache: cache, Key: key}), nil
}

func withRef(info Info, cache, key string) ArtifactInfo {
	return ArtifactInfo{Info: info, Ref: cache + ":" + key}
}

func cacheBucket(handler, cache string) string {
	if cache == "" {
		return handler
	}
	return handler + "-" + cache
}

func resolveCache(spec Spec, name string) (string, Policy, error) {
	caches := spec.Caches
	if name == "" {
		if len(caches) != 1 {
			return "", Policy{}, apierr.NewNotFoundMessageError("unknown artifact cache")
		}
		return caches[0].Name, caches[0].Policy, nil
	}
	for _, cache := range caches {
		if cache.Name == name {
			return cache.Name, cache.Policy, nil
		}
	}
	return "", Policy{}, apierr.NewNotFoundMessageError("unknown artifact cache")
}

func bindCache(spec Spec, handler string, resolved resolvedRequest) (string, Policy, error) {
	cache, policy, err := resolveCache(spec, resolved.Cache)
	if err != nil {
		return "", Policy{}, err
	}
	return cacheBucket(handler, cache), policy, nil
}

func taskGroup(handler string) string {
	return "artifact/" + handler
}

// taskName is the label stored on the task and returned to clients. It is the
// source path, without args.
func taskName(request Request) string {
	return request.Source.Path()
}

// SysConfig exposes live handler configuration through the single artifact
// entry in /config.
func (s *Service) SysConfig() (string, types.M, error) {
	config := make(types.M, len(s.handlers))
	for name, handler := range s.handlers {
		values := handler.Spec().Config
		if values == nil {
			continue
		}
		clone := make(types.M, len(values))
		maps.Copy(clone, values)
		config[name] = clone
	}
	return "artifact", config, nil
}

func (s *Service) clean() {
	for name, handler := range s.handlers {
		for _, cache := range handler.Spec().Caches {
			bucket := cacheBucket(name, cache.Name)
			cleaned, err := s.store.Clean(bucket, cache.Policy.TTL, cache.Policy.MaxBytes)
			if err != nil {
				artifactLog.Warnf("artifact cleanup handler=%s cache=%s: %v", name, cache.Name, err)
				continue
			}
			if cleaned > 0 {
				artifactLog.Debugf("artifact cleanup handler=%s cache=%s cleaned=%d", name, cache.Name, cleaned)
			}
		}
	}
}

func (s *Service) Dispose() error {
	if s == nil {
		return nil
	}
	if s.stopCleaner != nil {
		s.stopCleaner()
		s.stopCleaner = nil
	}
	var err error
	for _, handler := range s.handlers {
		if disposable, ok := handler.(types.IDisposable); ok {
			err = errors.Join(err, disposable.Dispose())
		}
	}
	s.handlers = nil
	return err
}
