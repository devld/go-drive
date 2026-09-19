package artifact

import (
	"errors"
	apierr "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"testing"
)

type registryTestProcessor struct{}

type registryTestRunner struct {
	task.Runner
	group       string
	concurrency int
	calls       int
}

func (r *registryTestRunner) RegisterGroup(group string, concurrency int) error {
	r.group = group
	r.concurrency = concurrency
	r.calls++
	return nil
}

func (registryTestProcessor) Identity(ArtifactRequest) (Identity, error) {
	return Identity{}, nil
}

func (registryTestProcessor) Produce(types.TaskCtx, ArtifactRequest, Writer) error {
	return nil
}

type registryTestHandler struct {
	registryTestProcessor
	registrations []Registration
	config        types.M
}

func (h registryTestHandler) Registrations() []Registration {
	return h.registrations
}

func (h registryTestHandler) Config() types.M {
	return h.config
}

func registerTestHandler(t *testing.T, name string, handler Handler) {
	t.Helper()
	snapshot := append([]handlerFactoryEntry(nil), handlerFactories...)
	t.Cleanup(func() { handlerFactories = snapshot })
	RegisterHandler(name, func(HandlerContext) (Handler, error) {
		return handler, nil
	})
}

func newTestService(t *testing.T, runner task.Runner, handlers ...Handler) *Service {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(store, runner, HandlerContext{}, handlers...)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestServiceRegistersGenericArtifactType(t *testing.T) {
	runner := &registryTestRunner{}
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	registerTestHandler(t, "custom", registryTestHandler{
		registrations: []Registration{{
			Type:        "custom-preview",
			Policy:      Policy{MaxBytes: 32},
			TaskGroup:   "custom",
			Concurrency: 2,
		}},
		config: types.M{"extensions": "foo,bar"},
	})
	svc, err := NewService(store, runner, HandlerContext{})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := svc.resolve("custom-preview")
	if err != nil || resolved.Type != "custom-preview" || resolved.TaskGroup != "custom" || resolved.handler == nil {
		t.Fatalf("resolve() = %#v, %v", resolved, err)
	}
	if runner.group != "custom" || runner.concurrency != 2 {
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
	writer, err := store.Create("custom-preview", "source", "fingerprint")
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
		registrations: []Registration{{Type: "live"}},
		config:        types.M{"maxSize": int64(1)},
	}
	registerTestHandler(t, "live", handler)
	svc := newTestService(t, nil)
	handler.config = types.M{"maxSize": int64(2)}
	_, config, err := svc.SysConfig()
	if err != nil {
		t.Fatal(err)
	}
	live, ok := config["live"].(types.M)
	if !ok || live["maxSize"] != int64(2) {
		t.Fatalf("live handler config = %#v", config["live"])
	}
}

func TestServiceSharesTaskGroupAcrossArtifactTypes(t *testing.T) {
	runner := &registryTestRunner{}
	_ = newTestService(t, runner, registryTestHandler{registrations: []Registration{
		{Type: "first", TaskGroup: "shared", Concurrency: 2},
		{Type: "second", TaskGroup: "shared", Concurrency: 2},
	}})
	if runner.calls != 1 {
		t.Fatalf("RegisterGroup() calls = %d, want 1", runner.calls)
	}
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewService(store, runner, HandlerContext{},
		registryTestHandler{registrations: []Registration{
			{Type: "first", TaskGroup: "shared", Concurrency: 2},
		}},
		registryTestHandler{registrations: []Registration{
			{Type: "conflict", TaskGroup: "shared", Concurrency: 1},
		}},
	)
	if err == nil {
		t.Fatal("conflicting task-group concurrency unexpectedly succeeded")
	}
}

func TestServiceInstallsMultipleTypesFromOneHandler(t *testing.T) {
	handler := registryTestHandler{
		registrations: []Registration{
			{Type: "index"},
			{Type: "content"},
		},
		config: types.M{"extensions": "zip"},
	}
	registerTestHandler(t, "shared", handler)
	svc := newTestService(t, nil)
	if _, err := svc.resolve("index"); err != nil {
		t.Fatal("index artifact type was not registered")
	}
	if _, err := svc.resolve("content"); err != nil {
		t.Fatal("content artifact type was not registered")
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

func TestRegisterHandlerOverwritesDuplicates(t *testing.T) {
	snapshot := append([]handlerFactoryEntry(nil), handlerFactories...)
	t.Cleanup(func() { handlerFactories = snapshot })

	RegisterHandler("test-handler", func(HandlerContext) (Handler, error) {
		return registryTestHandler{registrations: []Registration{{Type: "first"}}}, nil
	})
	RegisterHandler("test-handler", func(HandlerContext) (Handler, error) {
		return registryTestHandler{registrations: []Registration{{Type: "second"}}}, nil
	})
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(store, nil, HandlerContext{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.resolve("second"); err != nil {
		t.Fatal("overwritten factory was not installed")
	}
	if _, err := svc.resolve("first"); err == nil {
		t.Fatal("replaced factory type was still installed")
	}
}

func TestServiceSysConfigOmitsUnnamedHandlers(t *testing.T) {
	svc := newTestService(t, nil, registryTestHandler{
		registrations: []Registration{{Type: "extra"}},
		config:        types.M{"maxSize": int64(1)},
	})
	_, config, err := svc.SysConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := config["extra"]; ok {
		t.Fatalf("unnamed extra handler was published: %#v", config)
	}
}

func TestServiceRejectsDuplicateTypes(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := registryTestHandler{registrations: []Registration{{Type: "preview"}}}
	if _, err := NewService(store, nil, HandlerContext{}, handler, handler); err == nil {
		t.Fatal("duplicate Register() unexpectedly succeeded")
	}
}

func TestServiceResolveUnknownAndNil(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.resolve("missing"); !apierr.IsNotFoundError(err) {
		t.Fatalf("resolve(missing) = %v, want bad request", err)
	}
	var nilService *Service
	if _, err := nilService.resolve("thumbnail"); !apierr.IsNotFoundError(err) {
		t.Fatalf("nil resolve() = %v, want not found", err)
	}
}

func TestServiceCachesCacheableProduceFailure(t *testing.T) {
	handler := &cacheableFailureHandler{}
	svc := newTestService(t, nil, handler)
	request := ArtifactRequest{Type: "preview"}
	if _, err := svc.Generate(task.DummyContext(), request); err == nil || err.Error() != "boom" {
		t.Fatalf("first Generate() = %v, want boom", err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after first attempt = %d, want 1", handler.calls)
	}
	if _, err := svc.Generate(task.DummyContext(), request); !apierr.IsNotFoundError(err) {
		t.Fatalf("cached Generate() = %v, want not found", err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after cached failure = %d, want 1", handler.calls)
	}
}

type cacheableFailureHandler struct {
	calls int
}

func (h *cacheableFailureHandler) Identity(ArtifactRequest) (Identity, error) {
	return Identity{Key: "source", Fingerprint: "fp"}, nil
}

func (h *cacheableFailureHandler) Produce(types.TaskCtx, ArtifactRequest, Writer) error {
	h.calls++
	return Cacheable(errors.New("boom"))
}

func (h *cacheableFailureHandler) Registrations() []Registration {
	return []Registration{{Type: "preview"}}
}

func (h *cacheableFailureHandler) Config() types.M {
	return nil
}
