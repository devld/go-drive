package job

import (
	"testing"

	"go-drive/common/types"
)

func TestCronTriggerRegisterAndGetInfo(t *testing.T) {
	trigger := newCronTrigger(nil)
	t.Cleanup(func() {
		if e := trigger.Dispose(); e != nil {
			t.Errorf("dispose scheduler: %v", e)
		}
	})

	config := types.SM{"schedule": "0 0 1 1 *"}
	if e := trigger.Register(42, config); e != nil {
		t.Fatalf("register cron job: %v", e)
	}
	if e := trigger.Register(42, config); e == nil {
		t.Fatal("expected duplicate registration to fail")
	}

	info, e := trigger.GetInfo(42)
	if e != nil {
		t.Fatalf("get cron job info: %v", e)
	}
	if len(info) != 1 || info[0]["nextRun"] == "" {
		t.Fatalf("unexpected cron job info: %#v", info)
	}

	trigger.Clear()
	info, e = trigger.GetInfo(42)
	if e != nil {
		t.Fatalf("get cleared cron job info: %v", e)
	}
	if len(info) != 0 {
		t.Fatalf("expected cleared scheduler, got %#v", info)
	}
}
