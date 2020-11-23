package task

import (
	"context"
	"errors"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map"
	"go-drive/common"
	"go-drive/common/types"
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

func NewTunnyRunner(config common.Config, ch *common.ComponentsHolder) *TunnyRunner {
	tr := &TunnyRunner{
		pool:  tunny.NewFunc(config.MaxConcurrentTask, executor),
		store: cmap.New(),
	}
	tr.tickerStop = common.TimeTick(tr.clean, 30*time.Second)
	ch.Add("taskRunner", tr)
	return tr
}

func (t *TunnyRunner) createTask(runnable Runnable) *wrapper {
	task := &Task{
		Id:        uuid.New().String(),
		Status:    Pending,
		Progress:  Progress{Loaded: 0, Total: 0},
		CreatedAt: time.Now(),
	}

	w := &wrapper{
		done:     make(chan struct{}),
		runnable: runnable,
		task:     task,
		mux:      &sync.Mutex{},
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
	return *w.(*wrapper).task, nil
}

func (t *TunnyRunner) StopTask(id string) (Task, error) {
	temp, ok := t.store.Get(id)
	if !ok {
		return Task{}, ErrorNotFound
	}
	w := temp.(*wrapper)
	w.cancel()
	return *w.task, nil
}

func (t *TunnyRunner) RemoveTask(id string) error {
	temp, ok := t.store.Get(id)
	if !ok {
		return ErrorNotFound
	}
	w := temp.(*wrapper)
	w.cancel()
	t.store.Remove(w.task.Id)
	return nil
}

func (t *TunnyRunner) Dispose() error {
	t.store.IterCb(func(key string, v interface{}) {
		v.(*wrapper).cancel()
	})
	t.pool.Close()
	t.tickerStop()
	return nil
}

func (t *TunnyRunner) clean() {
	ids := make([]string, 0)
	t.store.IterCb(func(key string, v interface{}) {
		t := v.(*wrapper)
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
		switch v.(*wrapper).task.Status {
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
		"Total":    fmt.Sprintf("%d", total),
		"Pending":  fmt.Sprintf("%d", pending),
		"Running":  fmt.Sprintf("%d", running),
		"Done":     fmt.Sprintf("%d", done),
		"Error":    fmt.Sprintf("%d", err),
		"Canceled": fmt.Sprintf("%d", canceled),
	}, nil
}

type wrapper struct {
	runnable Runnable
	task     *Task
	canceled bool
	mux      *sync.Mutex
	done     chan struct{}
}

func (w *wrapper) Progress(loaded int64, abs bool) {
	if w.canceled {
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

func (w *wrapper) Total(total int64, abs bool) {
	if w.canceled {
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

func (w *wrapper) Canceled() bool {
	return w.canceled
}

func (w *wrapper) Deadline() (deadline time.Time, ok bool) {
	return
}

func (w *wrapper) Done() <-chan struct{} {
	return w.done
}

func (w *wrapper) Err() error {
	if w.canceled {
		return context.Canceled
	}
	return nil
}

func (w *wrapper) Value(interface{}) interface{} {
	return nil
}

func (w *wrapper) cancel() {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.canceled {
		return
	}
	close(w.done)
	w.canceled = true
	w.task.Status = Canceled
}

func executor(arg interface{}) interface{} {
	w := arg.(*wrapper)
	if w.Canceled() {
		return nil
	}
	w.task.Status = Running
	w.task.UpdatedAt = time.Now()
	r, e := w.runnable(w)
	if e != nil {
		if e == ErrorCanceled || errors.Is(e, context.Canceled) {
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
