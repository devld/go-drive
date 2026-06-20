package task

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type PondRunner struct {
	pool       pond.Pool
	store      cmap.ConcurrentMap[string, *pondTaskCtx]
	tickerStop func()
}

var cleanThreshold = 10 * time.Minute

func NewPondRunner(config common.Config, ch *registry.ComponentsHolder) *PondRunner {
	tr := &PondRunner{
		pool:  pond.NewPool(config.MaxConcurrentTask),
		store: cmap.New[*pondTaskCtx](),
	}
	tr.tickerStop = utils.TimeTick(tr.clean, 30*time.Second)
	ch.Add(registry.KeyTaskRunner, tr)
	return tr
}

func (t *PondRunner) createTask(runnable Runnable, options ...Option) *pondTaskCtx {
	task := &Task{
		Id:        uuid.New().String(),
		Status:    Pending,
		Progress:  Progress{Loaded: 0, Total: 0},
		CreatedAt: time.Now(),
	}

	for _, o := range options {
		o(task)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	w := &pondTaskCtx{
		Context:  ctx,
		cancelFn: cancelFunc,
		runnable: runnable,
		task:     task,
	}

	t.store.Set(task.Id, w)
	return w
}

func (t *PondRunner) Execute(runnable Runnable, option ...Option) (Task, error) {
	w := t.createTask(runnable, option...)
	if e := t.pool.Go(func() { execute(w) }); e != nil {
		t.store.Remove(w.task.Id)
		return w.snapshot(), e
	}
	return w.snapshot(), nil
}

func (t *PondRunner) ExecuteAndWait(runnable Runnable, timeout time.Duration, option ...Option) (Task, error) {
	w := t.createTask(runnable, option...)

	timer := time.NewTimer(timeout)
	done := make(chan struct{})
	defer timer.Stop()

	if e := t.pool.Go(func() {
		execute(w)
		close(done)
	}); e != nil {
		t.store.Remove(w.task.Id)
		return w.snapshot(), e
	}
	select {
	case <-timer.C:
		// Timeout only limits how long the caller waits. The task deliberately
		// remains queued/running and continues in the background.
	case <-done:
	}

	return w.snapshot(), nil
}

func (t *PondRunner) GetTasks(group string) ([]Task, error) {
	tasks := make([]Task, 0)
	for _, w := range t.store.Items() {
		task := w.snapshot()
		if group == "" || task.Group == group || strings.HasPrefix(task.Group, group+"/") {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (t *PondRunner) GetTask(id string) (Task, error) {
	w, ok := t.store.Get(id)
	if !ok {
		return Task{}, ErrorNotFound
	}
	return w.snapshot(), nil
}

func (t *PondRunner) StopTask(id string) (Task, error) {
	w, ok := t.store.Get(id)
	if !ok {
		return Task{}, ErrorNotFound
	}
	if task := w.snapshot(); task.Finished() {
		return task, nil
	}
	w.cancel()
	return w.snapshot(), nil
}

func (t *PondRunner) RemoveTask(id string) error {
	w, ok := t.store.Get(id)
	if !ok {
		return ErrorNotFound
	}
	w.cancel()
	t.store.Remove(w.task.Id)
	return nil
}

func (t *PondRunner) Dispose() error {
	t.store.IterCb(func(key string, v *pondTaskCtx) { v.cancel() })
	t.tickerStop()
	t.pool.StopAndWait()
	return nil
}

func (t *PondRunner) clean() {
	ids := make([]string, 0)
	t.store.IterCb(func(key string, t *pondTaskCtx) {
		task := t.snapshot()
		if task.Finished() && (time.Now().Unix()-task.UpdatedAt.Unix() > int64(cleanThreshold.Seconds())) {
			ids = append(ids, task.Id)
		}
	})
	for _, id := range ids {
		t.store.Remove(id)
	}
	if len(ids) > 0 {
		log.Printf("%d tasks cleaned", len(ids))
	}
}

func (t *PondRunner) Status() (string, types.SM, error) {
	total := 0
	pending := 0
	running := 0
	done := 0
	err := 0
	canceled := 0

	t.store.IterCb(func(key string, v *pondTaskCtx) {
		switch v.snapshot().Status {
		case Pending:
			pending++
		case Running:
			running++
		case Done:
			done++
		case Error:
			err++
		case Canceled:
			canceled++
		}
		total++
	})
	return "Task", types.SM{
		i18n.T("stat.task.total"):    fmt.Sprintf("%d", total),
		i18n.T("stat.task.pending"):  fmt.Sprintf("%d", pending),
		i18n.T("stat.task.running"):  fmt.Sprintf("%d", running),
		i18n.T("stat.task.done"):     fmt.Sprintf("%d", done),
		i18n.T("stat.task.error"):    fmt.Sprintf("%d", err),
		i18n.T("stat.task.canceled"): fmt.Sprintf("%d", canceled),
	}, nil
}

type pondTaskCtx struct {
	context.Context

	cancelFn func()
	runnable Runnable
	task     *Task
	mux      sync.RWMutex
}

func (w *pondTaskCtx) Progress(loaded int64, abs bool) {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.Err() != nil || w.task.Finished() {
		return
	}
	if abs {
		w.task.Progress.Loaded = loaded
	} else {
		w.task.Progress.Loaded += loaded
	}
	w.task.UpdatedAt = time.Now()
}

func (w *pondTaskCtx) Total(total int64, abs bool) {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.Err() != nil || w.task.Finished() {
		return
	}
	if abs {
		w.task.Progress.Total = total
	} else {
		w.task.Progress.Total += total
	}
	w.task.UpdatedAt = time.Now()
}

func (w *pondTaskCtx) snapshot() Task {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return *w.task
}

func (w *pondTaskCtx) cancel() {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.task.Finished() {
		return
	}
	w.cancelFn()
	w.task.Status = Canceled
}

func execute(w *pondTaskCtx) {
	w.mux.Lock()
	if w.Err() != nil || w.task.Finished() {
		w.mux.Unlock()
		return
	}
	w.task.Status = Running
	w.task.UpdatedAt = time.Now()
	w.mux.Unlock()

	defer func() {
		if recovered := recover(); recovered != nil {
			finishTask(w, nil, fmt.Errorf("task panicked: %v\n%s", recovered, debug.Stack()))
		}
	}()
	r, e := w.runnable(w)
	finishTask(w, r, e)
}

func finishTask(w *pondTaskCtx, result any, taskErr error) {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.Err() != nil {
		w.task.Status = Canceled
		w.task.UpdatedAt = time.Now()
		return
	}
	if taskErr != nil {
		if errors.Is(taskErr, context.Canceled) {
			w.task.Status = Canceled
		} else {
			log.Printf("error when executing task: %s", taskErr.Error())
			w.task.Status = Error
			w.task.Error = types.M{"message": taskErr.Error()}
		}
	} else {
		w.task.Status = Done
		w.task.Result = result
	}
	w.task.UpdatedAt = time.Now()
}
