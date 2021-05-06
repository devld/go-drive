package task

import (
	"context"
	"errors"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map"
	"go-drive/common"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"
	"sync"
	"time"
)

type TunnyRunner struct {
	pool       *tunny.Pool
	store      cmap.ConcurrentMap
	tickerStop func()
}

var cleanThreshold = 1 * time.Minute

func NewTunnyRunner(config common.Config, ch *registry.ComponentsHolder) *TunnyRunner {
	tr := &TunnyRunner{
		pool:  tunny.NewFunc(config.MaxConcurrentTask, executor),
		store: cmap.New(),
	}
	tr.tickerStop = utils.TimeTick(tr.clean, 30*time.Second)
	ch.Add("taskRunner", tr)
	return tr
}

func (t *TunnyRunner) createTask(runnable Runnable) *tunnyTaskCtx {
	task := &Task{
		Id:        uuid.New().String(),
		Status:    Pending,
		Progress:  Progress{Loaded: 0, Total: 0},
		CreatedAt: time.Now(),
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	w := &tunnyTaskCtx{
		Context:  ctx,
		cancelFn: cancelFunc,
		runnable: runnable,
		task:     task,
		mux:      &sync.RWMutex{},
	}

	t.store.Set(task.Id, w)
	return w
}

func (t *TunnyRunner) Execute(runnable Runnable) (Task, error) {
	w := t.createTask(runnable)
	go t.pool.Process(w)
	return *w.task, nil
}

func (t *TunnyRunner) ExecuteAndWait(runnable Runnable, timeout time.Duration) (Task, error) {
	w := t.createTask(runnable)

	timer := time.NewTimer(timeout)
	done := make(chan int)
	defer timer.Stop()

	go func() {
		t.pool.Process(w)
		done <- 0
	}()
	select {
	case <-timer.C:
	case <-done:
	}

	return *w.task, nil
}

func (t *TunnyRunner) GetTask(id string) (Task, error) {
	w, ok := t.store.Get(id)
	if !ok {
		return Task{}, ErrorNotFound
	}
	return *w.(*tunnyTaskCtx).task, nil
}

func (t *TunnyRunner) StopTask(id string) (Task, error) {
	temp, ok := t.store.Get(id)
	if !ok {
		return Task{}, ErrorNotFound
	}
	w := temp.(*tunnyTaskCtx)
	w.cancel()
	return *w.task, nil
}

func (t *TunnyRunner) RemoveTask(id string) error {
	temp, ok := t.store.Get(id)
	if !ok {
		return ErrorNotFound
	}
	w := temp.(*tunnyTaskCtx)
	w.cancel()
	t.store.Remove(w.task.Id)
	return nil
}

func (t *TunnyRunner) Dispose() error {
	t.store.IterCb(func(key string, v interface{}) {
		v.(*tunnyTaskCtx).cancel()
	})
	t.pool.Close()
	t.tickerStop()
	return nil
}

func (t *TunnyRunner) clean() {
	ids := make([]string, 0)
	t.store.IterCb(func(key string, v interface{}) {
		t := v.(*tunnyTaskCtx)
		if t.task.Finished() && (time.Now().Unix()-t.task.UpdatedAt.Unix() > int64(cleanThreshold.Seconds())) {
			ids = append(ids, t.task.Id)
		}
	})
	for _, id := range ids {
		t.store.Remove(id)
	}
	if len(ids) > 0 {
		log.Printf("%d tasks cleaned", len(ids))
	}
}

func (t *TunnyRunner) Status() (string, types.SM, error) {
	total := 0
	pending := 0
	running := 0
	done := 0
	err := 0
	canceled := 0

	t.store.IterCb(func(key string, v interface{}) {
		switch v.(*tunnyTaskCtx).task.Status {
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

type tunnyTaskCtx struct {
	context.Context

	cancelFn func()
	runnable Runnable
	task     *Task
	mux      *sync.RWMutex
}

func (w *tunnyTaskCtx) Progress(loaded int64, abs bool) {
	if w.Err() != nil {
		return
	}
	w.mux.Lock()
	defer w.mux.Unlock()
	if abs {
		w.task.Progress.Loaded = loaded
	} else {
		w.task.Progress.Loaded += loaded
	}
	w.task.UpdatedAt = time.Now()
}

func (w *tunnyTaskCtx) Total(total int64, abs bool) {
	if w.Err() != nil {
		return
	}
	w.mux.Lock()
	defer w.mux.Unlock()
	if abs {
		w.task.Progress.Total = total
	} else {
		w.task.Progress.Total += total
	}
	w.task.UpdatedAt = time.Now()
}

func (w *tunnyTaskCtx) cancel() {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.Err() != nil {
		return
	}
	w.cancelFn()
	w.task.Status = Canceled
}

func executor(arg interface{}) interface{} {
	w := arg.(*tunnyTaskCtx)
	if w.Err() != nil {
		return nil
	}
	w.task.Status = Running
	w.task.UpdatedAt = time.Now()
	r, e := w.runnable(w)
	if e != nil {
		if errors.Is(e, context.Canceled) {
			w.task.Status = Canceled
		} else {
			log.Printf("error when executing task: %s", e.Error())
			w.task.Status = Error
			w.task.Error = types.M{"message": e.Error()}
		}
	} else {
		w.task.Status = Done
		w.task.Result = r
	}
	w.task.UpdatedAt = time.Now()
	return nil
}
