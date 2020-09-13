package task

import (
	"github.com/Jeffail/tunny"
	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map"
	"log"
	"time"
)

type TunnyRunner struct {
	pool       *tunny.Pool
	store      cmap.ConcurrentMap
	cleanTimer *time.Ticker
	dispose    chan bool
}

var cleanThreshold = 1 * time.Minute

func NewTunnyRunner(workers int) *TunnyRunner {
	tr := &TunnyRunner{
		pool:       tunny.NewFunc(workers, executor),
		store:      cmap.New(),
		cleanTimer: time.NewTicker(30 * time.Second),
		dispose:    make(chan bool),
	}
	go func() {
		for {
			select {
			case <-tr.dispose:
				return
			case <-tr.cleanTimer.C:
				tr.clean()
			}
		}
	}()
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
		runnable: runnable,
		task:     task,
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
	w.canceled = true
	return *w.task, nil
}

func (t *TunnyRunner) RemoveTask(id string) error {
	temp, ok := t.store.Get(id)
	if !ok {
		return ErrorNotFound
	}
	w := temp.(*wrapper)
	w.canceled = true
	t.store.Remove(w.task.Id)
	return nil
}

func (t *TunnyRunner) Dispose() error {
	t.store.IterCb(func(key string, v interface{}) {
		v.(*wrapper).canceled = true
	})
	t.pool.Close()
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

type wrapper struct {
	runnable Runnable
	task     *Task
	canceled bool
}

func (w *wrapper) Progress(loaded int64) {
	w.task.Progress.Loaded = loaded
	w.task.UpdatedAt = time.Now()
}

func (w *wrapper) Total(total int64) {
	w.task.Progress.Total = total
	w.task.UpdatedAt = time.Now()
}

func (w *wrapper) Canceled() bool {
	return w.canceled
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
		if e == ErrorCanceled {
			w.task.Status = Canceled
		} else {
			log.Printf("error when executing task: %s", e.Error())
			w.task.Status = Error
			w.task.Error = e.Error()
		}
	} else {
		w.task.Status = Done
		w.task.Result = r
	}
	w.task.UpdatedAt = time.Now()
	return nil
}
