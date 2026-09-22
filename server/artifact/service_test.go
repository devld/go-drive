package artifact

import (
	"context"
	"errors"
	"go-drive/common"
	apierr "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"testing"
	"time"
)

type registryTestProcessor struct{}

type registryTestRunner struct {
	task.Runner
	group       string
	concurrency int
	calls       int
	groups      map[string]int
}

func (r *registryTestRunner) RegisterGroup(group string, concurrency int) error {
	if r.groups == nil {
		r.groups = make(map[string]int)
	}
	if _, exists := r.groups[group]; exists {
		return errors.New("task group already registered")
	}
	r.groups[group] = concurrency
	r.group = group
	r.concurrency = concurrency
	r.calls++
	return nil
}

func (registryTestProcessor) Resolve(Request) (ResolvedRequest, error) {
	return ResolvedRequest{}, nil
}

func (registryTestProcessor) Produce(types.TaskCtx, Request, Writer) error {
	return nil
}

type registryTestHandler struct {
	registryTestProcessor
	registrations []Spec
}

func (h registryTestHandler) Spec() Spec {
	if len(h.registrations) == 0 {
		return Spec{Caches: []CacheSpec{{}}}
	}
	spec := h.registrations[0]
	if len(spec.Caches) == 0 {
		spec.Caches = []CacheSpec{{}}
	}
	return spec
}

func registerTestHandler(t *testing.T, name string, handler Handler) {
	t.Helper()
	snapshot := append([]handlerFactoryEntry(nil), handlerFactories...)
	t.Cleanup(func() { handlerFactories = snapshot })
	RegisterHandler(name, func(HandlerContext) (Handler, error) {
		return handler, nil
	})
}

func newTestService(t *testing.T, runner task.Runner) *Service {
	t.Helper()
	svc, err := NewService(common.Config{TempDir: t.TempDir()}, runner)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Dispose() })
	return svc
}

func generateRequest(svc *Service, request Request) (Info, error) {
	handler, err := svc.resolveHandler(request.Handler)
	if err != nil {
		return Info{}, err
	}
	resolved, err := resolveRequest(handler, request)
	if err != nil {
		return Info{}, err
	}
	return svc.generateLocked(task.DummyContext(), request, handler, resolved)
}

func TestServiceDisposeReleasesHandlers(t *testing.T) {
	handler := &disposableTestHandler{}
	svc := newTestService(t, nil)
	if err := svc.install("temp", handler); err != nil {
		t.Fatal(err)
	}
	if err := svc.Dispose(); err != nil {
		t.Fatal(err)
	}
	if handler.disposed != 1 {
		t.Fatalf("Dispose() calls = %d, want 1", handler.disposed)
	}
	if err := svc.Dispose(); err != nil {
		t.Fatal(err)
	}
	if handler.disposed != 1 {
		t.Fatalf("second Dispose() calls = %d, want 1", handler.disposed)
	}
}

type disposableTestHandler struct {
	registryTestHandler
	disposed int
}

func (h *disposableTestHandler) Dispose() error {
	h.disposed++
	return nil
}

func TestServiceRegistersGenericArtifactType(t *testing.T) {
	runner := &registryTestRunner{}
	registerTestHandler(t, "custom", registryTestHandler{
		registrations: []Spec{{
			Caches:      []CacheSpec{{Policy: Policy{MaxBytes: 32}}},
			Concurrency: 2,
			Config:      types.M{"extensions": "foo,bar"},
		}},
	})
	svc, err := NewService(common.Config{TempDir: t.TempDir()}, runner)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Dispose() })
	resolved, err := svc.resolveHandler("custom")
	if err != nil || resolved == nil {
		t.Fatalf("resolve() = %#v, %v", resolved, err)
	}
	if runner.group != "artifact/custom" || runner.concurrency != 2 {
		t.Fatalf("registered task group = %q/%d", runner.group, runner.concurrency)
	}
	if runner.calls != 1 {
		t.Fatalf("RegisterGroup() calls = %d, want 1", runner.calls)
	}
	name, config, err := svc.SysConfig()
	if err != nil || name != "artifact" {
		t.Fatalf("SysConfig() = %q %#v, %v", name, config, err)
	}
	customConfig, ok := config["custom"].(types.M)
	if !ok || customConfig["extensions"] != "foo,bar" {
		t.Fatalf("custom artifact config = %#v", config["custom"])
	}
	writer, err := svc.store.Create("custom", "source", "fingerprint")
	if err != nil {
		t.Fatalf("registered store type is not usable: %v", err)
	}
	if err := writer.WriteMeta(Meta{}); err != nil {
		writer.Abort()
		t.Fatalf("registered store type is not usable: %v", err)
	}
	if _, err := io.WriteString(writer, "artifact"); err != nil {
		writer.Abort()
		t.Fatalf("registered store type is not usable: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("registered store type is not usable: %v", err)
	}
}

func TestServiceSysConfigUsesRegisterHandlerName(t *testing.T) {
	handler := &registryTestHandler{
		registrations: []Spec{{Config: types.M{"maxSize": int64(1)}}},
	}
	registerTestHandler(t, "live", handler)
	svc := newTestService(t, nil)
	handler.registrations[0].Config = types.M{"maxSize": int64(2)}
	_, config, err := svc.SysConfig()
	if err != nil {
		t.Fatal(err)
	}
	live, ok := config["live"].(types.M)
	if !ok || live["maxSize"] != int64(2) {
		t.Fatalf("live handler config = %#v", config["live"])
	}
}

func TestServiceRegistersHandlerTaskGroup(t *testing.T) {
	runner := &registryTestRunner{}
	svc := newTestService(t, runner)
	if err := svc.install("shared", registryTestHandler{registrations: []Spec{
		{Concurrency: 2},
	}}); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || runner.group != "artifact/shared" || runner.concurrency != 2 {
		t.Fatalf("RegisterGroup() = %q/%d calls=%d", runner.group, runner.concurrency, runner.calls)
	}
}

func TestServiceSkipsRegisterGroupWhenConcurrencyZero(t *testing.T) {
	runner := &registryTestRunner{}
	svc := newTestService(t, runner)
	if err := svc.install("plain", registryTestHandler{registrations: []Spec{{}}}); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 0 {
		t.Fatalf("RegisterGroup() calls = %d, want 0", runner.calls)
	}
}

func TestServiceInstallsHandler(t *testing.T) {
	handler := registryTestHandler{
		registrations: []Spec{{Config: types.M{"extensions": "zip"}}},
	}
	registerTestHandler(t, "shared", handler)
	svc := newTestService(t, nil)
	if _, err := svc.resolveHandler("shared"); err != nil {
		t.Fatal("handler was not registered")
	}
	name, config, err := svc.SysConfig()
	if err != nil || name != "artifact" {
		t.Fatalf("SysConfig() = %q %#v, %v", name, config, err)
	}
	shared, ok := config["shared"].(types.M)
	if !ok || shared["extensions"] != "zip" {
		t.Fatalf("shared handler config = %#v", config["shared"])
	}
}

func TestRegisterHandlerPanicsOnDuplicate(t *testing.T) {
	snapshot := append([]handlerFactoryEntry(nil), handlerFactories...)
	t.Cleanup(func() { handlerFactories = snapshot })

	RegisterHandler("test-handler", func(HandlerContext) (Handler, error) {
		return registryTestHandler{}, nil
	})
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate RegisterHandler did not panic")
		}
	}()
	RegisterHandler("test-handler", func(HandlerContext) (Handler, error) {
		return registryTestHandler{}, nil
	})
}

func TestServiceSysConfigOmitsNilHandlerConfig(t *testing.T) {
	svc := newTestService(t, nil)
	if err := svc.install("extra", registryTestHandler{
		registrations: []Spec{{}},
	}); err != nil {
		t.Fatal(err)
	}
	_, config, err := svc.SysConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := config["extra"]; ok {
		t.Fatalf("unnamed extra handler was published: %#v", config)
	}
}

func TestServiceRejectsDuplicateHandlers(t *testing.T) {
	svc := newTestService(t, nil)
	handler := registryTestHandler{registrations: []Spec{{}}}
	if err := svc.install("preview", handler); err != nil {
		t.Fatal(err)
	}
	if err := svc.install("preview", handler); err == nil {
		t.Fatal("duplicate Register() unexpectedly succeeded")
	}
}

func TestServiceResolveUnknownHandler(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.resolveHandler("missing"); !apierr.IsNotFoundError(err) {
		t.Fatalf("resolve(missing) = %v, want not found", err)
	}
}

func TestServiceInvalidatesWhenSourceChanges(t *testing.T) {
	handler := &countingProduceHandler{}
	svc := newTestService(t, nil)
	if err := svc.install("preview", handler); err != nil {
		t.Fatal(err)
	}
	entry := &identityTestEntry{path: "a.png", real: "drive/a.png", size: 1, modTime: 1}
	request := Request{Handler: "preview", Source: entry}
	if _, err := generateRequest(svc, request); err != nil {
		t.Fatal(err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after first generate = %d, want 1", handler.calls)
	}
	if _, err := generateRequest(svc, request); err != nil {
		t.Fatal(err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after cache hit = %d, want 1", handler.calls)
	}
	entry.size++
	if _, err := generateRequest(svc, request); err != nil {
		t.Fatal(err)
	}
	if handler.calls != 2 {
		t.Fatalf("handler calls after source update = %d, want 2", handler.calls)
	}
}

type countingProduceHandler struct {
	calls int
}

func (h *countingProduceHandler) Resolve(Request) (ResolvedRequest, error) {
	return ResolvedRequest{Fingerprint: "v1"}, nil
}

func (h *countingProduceHandler) Produce(_ types.TaskCtx, _ Request, out Writer) error {
	h.calls++
	if err := out.WriteMeta(Meta{Name: "blob", MimeType: "text/plain"}); err != nil {
		return err
	}
	_, err := out.Write([]byte("ok"))
	return err
}

func (h *countingProduceHandler) Spec() Spec {
	return Spec{Caches: []CacheSpec{{}}}
}

func TestServiceCachesCacheableProduceFailure(t *testing.T) {
	handler := &cacheableFailureHandler{}
	svc := newTestService(t, nil)
	if err := svc.install("preview", handler); err != nil {
		t.Fatal(err)
	}
	request := Request{Handler: "preview"}
	if _, err := generateRequest(svc, request); err == nil || err.Error() != "boom" {
		t.Fatalf("first generate() = %v, want boom", err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after first attempt = %d, want 1", handler.calls)
	}
	if _, err := generateRequest(svc, request); !apierr.IsNotFoundError(err) {
		t.Fatalf("cached generate() = %v, want not found", err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after cached failure = %d, want 1", handler.calls)
	}
}

func TestHandlerCacheReadsOnlyOwnArtifacts(t *testing.T) {
	snapshot := append([]handlerFactoryEntry(nil), handlerFactories...)
	t.Cleanup(func() { handlerFactories = snapshot })

	var owner, peer *scopedCacheHandler
	RegisterHandler("owner", func(ctx HandlerContext) (Handler, error) {
		owner = &scopedCacheHandler{cache: ctx.Cache, body: "mine"}
		return owner, nil
	})
	RegisterHandler("peer", func(ctx HandlerContext) (Handler, error) {
		peer = &scopedCacheHandler{cache: ctx.Cache, body: "theirs"}
		return peer, nil
	})
	svc := newTestService(t, nil)
	if _, err := generateRequest(svc, Request{Handler: "owner"}); err != nil {
		t.Fatal(err)
	}

	got, err := owner.cache.Get(nil, "")
	if err != nil {
		t.Fatalf("owner cache Get() = %v", err)
	}
	defer func() { _ = got.Body.Close() }()
	body, err := io.ReadAll(got.Body)
	if err != nil || string(body) != "mine" {
		t.Fatalf("owner cache body = %q %v", body, err)
	}

	if _, err := peer.cache.Get(nil, ""); !apierr.IsNotFoundError(err) {
		t.Fatalf("peer cache Get() = %v, want not found", err)
	}
}

type scopedCacheHandler struct {
	cache Cache
	body  string
}

func (h *scopedCacheHandler) Resolve(Request) (ResolvedRequest, error) {
	return ResolvedRequest{Key: "shared", Fingerprint: "fp"}, nil
}

func (h *scopedCacheHandler) Produce(_ types.TaskCtx, _ Request, out Writer) error {
	if err := out.WriteMeta(Meta{Name: "blob", MimeType: "text/plain"}); err != nil {
		return err
	}
	_, err := out.Write([]byte(h.body))
	return err
}

func (h *scopedCacheHandler) Spec() Spec { return Spec{Caches: []CacheSpec{{}}} }

func TestServiceResolveSelectsStore(t *testing.T) {
	handler := &splitStoreHandler{}
	svc := newTestService(t, nil)
	if err := svc.install("split", handler); err != nil {
		t.Fatal(err)
	}
	if _, err := generateRequest(svc, Request{Handler: "split", Args: "index"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.store.Lookup("split", "k", "fp", 0); err == nil {
		t.Fatal("unnamed handler bucket unexpectedly exists")
	}
	index, err := svc.open(Request{Handler: "split", Args: "index"})
	if err != nil {
		t.Fatal(err)
	}
	indexBody, err := io.ReadAll(index.Body)
	_ = index.Body.Close()
	if err != nil || string(indexBody) != "index-body" {
		t.Fatalf("index body = %q %v", indexBody, err)
	}
	if _, err := svc.open(Request{Handler: "split", Args: "content"}); !apierr.IsNotFoundError(err) {
		t.Fatalf("content open() before generate = %v, want not found", err)
	}
	info, err := generateRequest(svc, Request{Handler: "split", Args: "content"})
	if err != nil {
		t.Fatalf("content generate() = %#v %v", info, err)
	}
	if info.Ref == "" {
		t.Fatal("content Info.Ref is empty")
	}
	if _, err := generateRequest(svc, Request{Handler: "split", Args: "missing"}); !apierr.IsNotFoundError(err) {
		t.Fatalf("unknown store generate() = %v, want not found", err)
	}
}

type splitStoreHandler struct{}

func (splitStoreHandler) Resolve(request Request) (ResolvedRequest, error) {
	return ResolvedRequest{Key: "k", Fingerprint: "fp", Cache: request.Args}, nil
}

func (splitStoreHandler) Produce(_ types.TaskCtx, request Request, out Writer) error {
	if err := out.WriteMeta(Meta{Name: request.Args, MimeType: "text/plain"}); err != nil {
		return err
	}
	_, err := out.Write([]byte(request.Args + "-body"))
	return err
}

func (splitStoreHandler) Spec() Spec {
	return Spec{Caches: []CacheSpec{{Name: "index"}, {Name: "content"}}}
}

type cacheableFailureHandler struct {
	calls int
}

func (h *cacheableFailureHandler) Resolve(Request) (ResolvedRequest, error) {
	return ResolvedRequest{Key: "source", Fingerprint: "fp"}, nil
}

func (h *cacheableFailureHandler) Produce(types.TaskCtx, Request, Writer) error {
	h.calls++
	return Cacheable(errors.New("boom"))
}

func (h *cacheableFailureHandler) Spec() Spec {
	return Spec{Caches: []CacheSpec{{}}}
}

func TestPrepareResultCarriesRef(t *testing.T) {
	handler := &countingProduceHandler{}
	svc := newTestService(t, syncRunner{})
	if err := svc.install("preview", handler); err != nil {
		t.Fatal(err)
	}
	entry := &identityTestEntry{path: "demo.zip", real: "drive/demo.zip", size: 4, modTime: 1}
	prepare := func(args string) Info {
		t.Helper()
		result, err := svc.Prepare(context.Background(), Request{
			Handler: "preview",
			Source:  entry,
			Args:    args,
		}, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if result.Task != nil {
			t.Fatalf("args %q returned a task: %#v", args, result.Task)
		}
		if result.Info.Ref == "" {
			t.Fatal("Info.Ref is empty")
		}
		return result.Info
	}
	first := prepare("index")
	second := prepare("content:docs/info.md")
	if first.Ref != second.Ref {
		t.Fatalf("ref changed without a new cache slot: %q %q", first.Ref, second.Ref)
	}
	if handler.calls != 1 {
		t.Fatalf("produce calls = %d, want 1", handler.calls)
	}
	opened, err := svc.OpenCached(entry, "preview", first.Ref)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(opened.Body)
	_ = opened.Body.Close()
	if err != nil || string(body) != "ok" {
		t.Fatalf("cached body = %q %v", body, err)
	}
	if _, err := svc.OpenCached(entry, "preview", "missing"); !apierr.IsNotFoundError(err) {
		t.Fatalf("missing ref = %v", err)
	}
	other := &identityTestEntry{path: "other.zip", real: "drive/other.zip", size: 4, modTime: 1}
	if _, err := svc.OpenCached(other, "preview", first.Ref); !apierr.IsNotFoundError(err) {
		t.Fatalf("ref for another path = %v", err)
	}
}

type syncRunner struct{}

func (syncRunner) Execute(runnable task.Runnable, options ...task.Option) (task.Task, error) {
	return syncRunner{}.ExecuteAndWait(context.Background(), runnable, 0, options...)
}

func (syncRunner) ExecuteAndWait(ctx context.Context, runnable task.Runnable, _ time.Duration, _ ...task.Option) (task.Task, error) {
	result, err := runnable(task.NewContextWrapper(ctx))
	if err != nil {
		return task.Task{Status: task.Error, Error: err}, nil
	}
	return task.Task{Status: task.Done, Result: result}, nil
}

func (syncRunner) RegisterGroup(string, int) error { return nil }
func (syncRunner) GetTask(string) (task.Task, error) {
	return task.Task{}, task.ErrorNotFound
}
func (syncRunner) GetTasks(string) ([]task.Task, error) { return nil, nil }
func (syncRunner) StopTask(string) (task.Task, error) {
	return task.Task{}, task.ErrorNotFound
}
func (syncRunner) RemoveTask(string) error { return nil }
func (syncRunner) Dispose() error          { return nil }
