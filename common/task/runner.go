package task

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	apierr "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/types"
	"go-drive/common/utils"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type taskRunner struct {
	mu         sync.Mutex
	running    int
	maxRun     int
	waiting    []*taskCtx
	maxWait    int
	groups     map[string]*taskGroup
	stopped    bool
	wg         sync.WaitGroup
	store      cmap.ConcurrentMap[string, *taskCtx]
	tickerStop func()
}

// taskGroup limits how many tasks of one exact group may run. Waiting tasks
// share taskRunner.waiting.
type taskGroup struct {
	limit   int
	running int
}

var cleanThreshold = 10 * time.Minute

var taskLog = logging.For("task")

func NewTaskRunner(config common.Config) Runner {
	tr := &taskRunner{
		maxRun:  config.MaxConcurrentTask,
		maxWait: config.MaxConcurrentTask,
		groups:  make(map[string]*taskGroup),
		store:   cmap.New[*taskCtx](),
	}
	tr.tickerStop = utils.TimeTick(tr.clean, 30*time.Second)
	return tr
}

var _ Runner = (*taskRunner)(nil)

func (t *taskRunner) RegisterGroup(group string, concurrency int) error {
	if concurrency <= 0 {
		return nil
	}
	if !IsValidGroup(group) {
		return fmt.Errorf("invalid task group %q", group)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.groups[group]; exists {
		return fmt.Errorf("task group %q is already registered", group)
	}
	if concurrency > t.maxRun {
		taskLog.Warnf("task group %s concurrency %d exceeds global task concurrency %d; using global limit",
			logging.Sanitize(group), concurrency, t.maxRun)
		concurrency = t.maxRun
	}
	// A limit that matches the global pool needs no separate cap.
	if concurrency == t.maxRun {
		t.groups[group] = &taskGroup{}
		return nil
	}
	t.groups[group] = &taskGroup{limit: concurrency}
	return nil
}

func (t *taskRunner) createTask(runnable Runnable, options ...Option) *taskCtx {
	task := &Task{
		ID:        uuid.New().String(),
		Status:    Pending,
		Progress:  Progress{Loaded: 0, Total: 0},
		CreatedAt: time.Now(),
	}

	for _, o := range options {
		o(task)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	w := &taskCtx{
		Context:  ctx,
		cancelFn: cancelFunc,
		runnable: runnable,
		task:     task,
	}

	t.store.Set(task.ID, w)
	taskLog.Debugf("task created id=%s group=%s name=%s", task.ID,
		logging.Sanitize(task.Group), logging.Sanitize(task.Name))
	return w
}

func (t *taskRunner) Execute(runnable Runnable, option ...Option) (Task, error) {
	w := t.createTask(runnable, option...)
	if e := t.submit(w); e != nil {
		w.cancelFn()
		t.store.Remove(w.task.ID)
		return w.snapshot(), e
	}
	return w.snapshot(), nil
}

// ExecuteAndWait waits for a detached task while the caller remains
// interested in its result. A non-positive timeout has no deadline. The waiter
// context and timeout never cancel the task; StopTask is the explicit
// cancellation mechanism. A canceled or timed-out wait returns the current
// snapshot with a nil error.
func (t *taskRunner) ExecuteAndWait(ctx context.Context, runnable Runnable, timeout time.Duration, option ...Option) (Task, error) {
	waitDone := ctx.Done()
	w := t.createTask(runnable, option...)
	done := make(chan struct{})
	var finished sync.Once
	w.done = func() { finished.Do(func() { close(done) }) }

	if e := t.submit(w); e != nil {
		w.cancelFn()
		t.store.Remove(w.task.ID)
		w.done()
		return w.snapshot(), e
	}
	var timerC <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timerC = timer.C
	}
	select {
	case <-timerC:
		taskLog.Debugf("task wait timed out id=%s group=%s name=%s timeout=%s; continuing in background",
			w.task.ID, logging.Sanitize(w.task.Group), logging.Sanitize(w.task.Name), timeout)
	case <-done:
	case <-waitDone:
		taskLog.Debugf("task wait canceled id=%s group=%s name=%s; continuing in background",
			w.task.ID, logging.Sanitize(w.task.Group), logging.Sanitize(w.task.Name))
	}
	// The waiter leaving does not fail the task. Timeout and a canceled
	// or deadline context only end the wait; generation keeps running.
	return w.snapshot(), nil
}

func (t *taskRunner) submit(w *taskCtx) error {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return unavailable()
	}
	if t.canStartLocked(w) {
		t.accountLocked(w)
		t.mu.Unlock()
		t.launch(w)
		return nil
	}
	if len(t.waiting) < t.maxWait {
		t.waiting = append(t.waiting, w)
		t.mu.Unlock()
		return nil
	}
	running, waiting := t.running, len(t.waiting)
	t.mu.Unlock()
	taskLog.Warnf("task queue full id=%s group=%s name=%s running=%d waiting=%d",
		w.task.ID, logging.Sanitize(w.task.Group), logging.Sanitize(w.task.Name), running, waiting)
	return unavailable()
}

func unavailable() error {
	return apierr.NewUnavailableError(i18n.T("error.task_queue_full"))
}

// canStartLocked reports whether w can take a global running slot. t.mu is held.
func (t *taskRunner) canStartLocked(w *taskCtx) bool {
	if w.canceled.Load() || t.running >= t.maxRun {
		return false
	}
	g := t.groups[w.task.Group]
	return g == nil || g.limit <= 0 || g.running < g.limit
}

func (t *taskRunner) accountLocked(w *taskCtx) {
	t.running++
	t.wg.Add(1)
	if g := t.groups[w.task.Group]; g != nil && g.limit > 0 {
		g.running++
	}
}

func (t *taskRunner) launch(tasks ...*taskCtx) {
	for _, w := range tasks {
		go func() {
			defer t.wg.Done()
			t.execute(w)
		}()
	}
}

func (t *taskRunner) release(w *taskCtx) {
	t.mu.Lock()
	if t.running > 0 {
		t.running--
	}
	if g := t.groups[w.task.Group]; g != nil && g.limit > 0 && g.running > 0 {
		g.running--
	}
	var ready, dropped []*taskCtx
	if !t.stopped {
		ready, dropped = t.collectLocked()
	}
	t.mu.Unlock()
	for _, item := range dropped {
		item.doneNotify()
	}
	t.launch(ready...)
}

// collectLocked pulls tasks that can start and drops canceled ones. t.mu is held.
// A task whose group is full stays in place so a later task can pass it.
func (t *taskRunner) collectLocked() (ready, dropped []*taskCtx) {
	kept := make([]*taskCtx, 0, len(t.waiting))
	for _, w := range t.waiting {
		if w.canceled.Load() {
			dropped = append(dropped, w)
			continue
		}
		if !t.canStartLocked(w) {
			kept = append(kept, w)
			continue
		}
		t.accountLocked(w)
		ready = append(ready, w)
	}
	t.waiting = kept
	return ready, dropped
}

func (t *taskRunner) dequeue(w *taskCtx) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, item := range t.waiting {
		if item != w {
			continue
		}
		t.waiting = append(t.waiting[:i], t.waiting[i+1:]...)
		return true
	}
	return false
}

func (t *taskRunner) GetTasks(group string) ([]Task, error) {
	tasks := make([]Task, 0)
	for _, w := range t.store.Items() {
		task := w.snapshot()
		if group == "" || task.Group == group || strings.HasPrefix(task.Group, group+"/") {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (t *taskRunner) GetTask(id string) (Task, error) {
	w, ok := t.store.Get(id)
	if !ok {
		return Task{}, ErrorNotFound
	}
	return w.snapshot(), nil
}

func (t *taskRunner) StopTask(id string) (Task, error) {
	w, ok := t.store.Get(id)
	if !ok {
		return Task{}, ErrorNotFound
	}
	if task := w.snapshot(); task.Finished() {
		return task, nil
	}
	w.cancel()
	if t.dequeue(w) {
		w.doneNotify()
	}
	taskLog.Debugf("task canceled id=%s group=%s name=%s", id,
		logging.Sanitize(w.task.Group), logging.Sanitize(w.task.Name))
	return w.snapshot(), nil
}

func (t *taskRunner) RemoveTask(id string) error {
	w, ok := t.store.Get(id)
	if !ok {
		return ErrorNotFound
	}
	w.cancel()
	if t.dequeue(w) {
		w.doneNotify()
	}
	t.store.Remove(w.task.ID)
	taskLog.Debugf("task removed id=%s group=%s name=%s", id,
		logging.Sanitize(w.task.Group), logging.Sanitize(w.task.Name))
	return nil
}

func (t *taskRunner) Dispose() error {
	t.mu.Lock()
	t.stopped = true
	waiting := t.waiting
	t.waiting = nil
	t.mu.Unlock()
	for _, w := range waiting {
		w.cancel()
		w.doneNotify()
	}
	t.store.IterCb(func(key string, v *taskCtx) { v.cancel() })
	t.tickerStop()
	t.wg.Wait()
	return nil
}

func (t *taskRunner) clean() {
	ids := make([]string, 0)
	t.store.IterCb(func(key string, t *taskCtx) {
		task := t.snapshot()
		if task.Finished() && (time.Now().Unix()-task.UpdatedAt.Unix() > int64(cleanThreshold.Seconds())) {
			ids = append(ids, task.ID)
		}
	})
	for _, id := range ids {
		t.store.Remove(id)
	}
	if len(ids) > 0 {
		logging.For("task").Debugf("%d tasks cleaned", len(ids))
	}
}

func (t *taskRunner) Status() (string, types.SM, error) {
	total := 0
	pending := 0
	running := 0
	done := 0
	err := 0
	canceled := 0

	t.store.IterCb(func(key string, v *taskCtx) {
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

type taskCtx struct {
	context.Context

	cancelFn func()
	runnable Runnable
	task     *Task
	mux      sync.RWMutex
	canceled atomic.Bool
	done     func()
}

func (w *taskCtx) TaskID() string {
	return w.task.ID
}

func (w *taskCtx) Progress(loaded int64, abs bool) {
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

func (w *taskCtx) Total(total int64, abs bool) {
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

func (w *taskCtx) GetProgress() int64 {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.task.Progress.Loaded
}

func (w *taskCtx) GetTotal() int64 {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.task.Progress.Total
}

func (w *taskCtx) snapshot() Task {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return *w.task
}

func (w *taskCtx) cancel() {
	w.canceled.Store(true)
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.task.Finished() {
		return
	}
	w.cancelFn()
	w.task.Status = Canceled
}

func (w *taskCtx) doneNotify() {
	if w.done != nil {
		w.done()
	}
}

func (t *taskRunner) execute(w *taskCtx) {
	defer w.doneNotify()
	defer t.release(w)
	// Status is recorded before this runs, including from the panic defer below.
	defer w.cancelFn()
	defer func() {
		if recovered := recover(); recovered != nil {
			finishTask(w, nil, fmt.Errorf("task panicked: %v\n%s", recovered, debug.Stack()))
		}
	}()
	w.mux.Lock()
	if w.Err() != nil || w.task.Finished() {
		w.mux.Unlock()
		return
	}
	w.task.Status = Running
	w.task.UpdatedAt = time.Now()
	w.mux.Unlock()
	taskLog.Debugf("task started id=%s group=%s name=%s", w.task.ID,
		logging.Sanitize(w.task.Group), logging.Sanitize(w.task.Name))

	r, e := w.runnable(w)
	finishTask(w, r, e)
}

func finishTask(w *taskCtx, result any, taskErr error) {
	w.mux.Lock()
	if w.Err() != nil {
		w.task.Status = Canceled
		w.task.UpdatedAt = time.Now()
		status := w.task.Status
		duration := w.task.UpdatedAt.Sub(w.task.CreatedAt)
		id, group, name := w.task.ID, w.task.Group, w.task.Name
		w.mux.Unlock()
		taskLog.Debugf("task finished id=%s group=%s name=%s status=%s duration=%s",
			id, logging.Sanitize(group), logging.Sanitize(name), status, duration)
		return
	}
	status := ""
	if taskErr != nil {
		if errors.Is(taskErr, context.Canceled) {
			w.task.Status = Canceled
		} else {
			w.task.Status = Error
			w.task.Error = jsonError{taskErr}
		}
	} else {
		w.task.Status = Done
		w.task.Result = result
	}
	w.task.UpdatedAt = time.Now()
	status = w.task.Status
	duration := w.task.UpdatedAt.Sub(w.task.CreatedAt)
	id, group, name := w.task.ID, w.task.Group, w.task.Name
	w.mux.Unlock()
	if taskErr != nil && status != Canceled {
		taskLog.Errorf("task failed id=%s group=%s name=%s duration=%s: %s",
			id, logging.Sanitize(group), logging.Sanitize(name), duration, taskErr)
	}
	taskLog.Debugf("task finished id=%s group=%s name=%s status=%s duration=%s",
		id, logging.Sanitize(group), logging.Sanitize(name), status, duration)
}
