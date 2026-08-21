package script

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	err "go-drive/common/errors"
	s "go-drive/script"
)

const (
	defaultIntervalTimeout = 30 * time.Second
	minInterval            = time.Millisecond
)

type driveInterval struct {
	name        string
	interval    time.Duration
	timeout     time.Duration
	immediately bool
	running     atomic.Bool
}

func (sd *ScriptDrive) prepareIntervals(raw *s.Value) error {
	if raw == nil || raw.IsNil() {
		return nil
	}
	specs := raw.Array()
	if specs == nil {
		return err.NewNotAllowedMessageError("intervals must be an array")
	}
	if len(specs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(specs))
	intervals := make([]*driveInterval, 0, len(specs))
	for _, spec := range specs {
		name := ""
		if nv := spec.Get("Name"); nv != nil && !nv.IsNil() {
			name = nv.String()
		}
		if name == "" {
			return err.NewNotAllowedMessageError("interval name is required")
		}
		if _, ok := seen[name]; ok {
			return err.NewNotAllowedMessageError("duplicate interval name: " + name)
		}
		seen[name] = struct{}{}
		interval, ok := s.DurationFrom(spec.Get("Interval"))
		if !ok || interval < minInterval {
			return err.NewNotAllowedMessageError("interval must be a duration >= 1ms: " + name)
		}
		timeout := defaultIntervalTimeout
		if tv := spec.Get("Timeout"); tv != nil && !tv.IsNil() {
			timeout, ok = s.DurationFrom(tv)
			if !ok {
				return err.NewNotAllowedMessageError("interval timeout must be a duration >= 1ms: " + name)
			}
			if timeout == 0 {
				timeout = defaultIntervalTimeout
			} else if timeout < minInterval {
				return err.NewNotAllowedMessageError("interval timeout must be a duration >= 1ms: " + name)
			}
		}
		immediately := false
		if iv := spec.Get("Immediately"); iv != nil && !iv.IsNil() {
			immediately = iv.Bool()
		}
		intervals = append(intervals, &driveInterval{
			name:        name,
			interval:    interval,
			timeout:     timeout,
			immediately: immediately,
		})
	}
	sd.intervals = intervals
	return nil
}

func (sd *ScriptDrive) startIntervals() error {
	if len(sd.intervals) == 0 {
		return nil
	}
	if !sd.has.onInterval {
		return err.NewNotAllowedMessageError("onInterval is required when intervals are declared")
	}
	sd.intervalCtx, sd.intervalCancel = context.WithCancel(context.Background())
	for _, interval := range sd.intervals {
		sd.intervalWG.Add(1)
		go sd.runInterval(interval)
	}
	return nil
}

func (sd *ScriptDrive) runInterval(job *driveInterval) {
	defer sd.intervalWG.Done()
	delay := job.interval
	if job.immediately {
		delay = 0
	}
	clock := time.NewTimer(delay)
	defer clock.Stop()
	for {
		select {
		case <-sd.intervalCtx.Done():
			return
		case <-clock.C:
			next := sd.fireInterval(job)
			clock.Reset(next)
		}
	}
}

func (sd *ScriptDrive) fireInterval(job *driveInterval) time.Duration {
	if sd.intervalCtx.Err() != nil {
		return job.interval
	}
	if !job.running.CompareAndSwap(false, true) {
		return job.interval
	}
	defer job.running.Store(false)

	ctx, cancel := context.WithTimeout(sd.intervalCtx, job.timeout)
	defer cancel()

	next := job.interval
	_, e := sd.withVM(ctx, func(vm *s.VM) (*s.Value, error) {
		v, e := sd.call(ctx, vm, "onInterval", s.NewContext(vm, ctx), job.name)
		if e != nil {
			return nil, e
		}
		if v == nil || v.IsNil() {
			return nil, nil
		}
		d := s.RequireDuration(v, "onInterval")
		if d < minInterval {
			vm.ThrowTypeError("onInterval must return a duration >= 1ms")
		}
		next = d
		return nil, nil
	})
	if e != nil {
		if sd.intervalCtx.Err() != nil || errors.Is(e, s.ErrVMPoolClosed) {
			return job.interval
		}
		log.Printf("script drive interval %q: %v", job.name, e)
		return job.interval
	}
	return next
}
