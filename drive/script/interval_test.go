package script

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestScriptDriveIntervalRunsAndStops(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return {
        $n: 0,
        intervals: [{ name: "tick", interval: ms(20), immediately: true }]
      };
    }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; },
    onInterval: function (ctx, name) {
      if (name !== "tick") throw new Error("unexpected interval " + name);
      var n = this.$n;
      n += 1;
      this.$n = n;
    }
  }
);
`, nil, nil)

	waitSharedInt(t, d, "$n", 1, time.Second)
	if e := d.Dispose(); e != nil {
		t.Fatal(e)
	}
	n := sharedInt(t, d, "$n")
	time.Sleep(80 * time.Millisecond)
	if got := sharedInt(t, d, "$n"); got != n {
		t.Fatalf("interval ran after Dispose: %d -> %d", n, got)
	}
}

func TestScriptDriveIntervalSkipsOverlap(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return {
        $n: 0,
        intervals: [{ name: "slow", interval: "15ms", timeout: "1s", immediately: true }]
      };
    }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; },
    onInterval: function () {
      var n = this.$n;
      n += 1;
      this.$n = n;
      sleep(ms(150));
    }
  }
);
`, nil, nil)

	waitSharedInt(t, d, "$n", 1, time.Second)
	time.Sleep(50 * time.Millisecond)
	if n := sharedInt(t, d, "$n"); n != 1 {
		t.Fatalf("overlapping ticks were not skipped, n=%d", n)
	}
}

func TestScriptDriveIntervalReschedulesFromReturnValue(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return {
        $n: 0,
        intervals: [{ name: "dyn", interval: "10ms", immediately: true }]
      };
    }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; },
    onInterval: function () {
      var n = this.$n;
      n += 1;
      this.$n = n;
      return "200ms";
    }
  }
);
`, nil, nil)

	waitSharedInt(t, d, "$n", 1, time.Second)
	time.Sleep(80 * time.Millisecond)
	if n := sharedInt(t, d, "$n"); n != 1 {
		t.Fatalf("returned delay was ignored, n=%d", n)
	}
	waitSharedInt(t, d, "$n", 2, time.Second)
}

func TestScriptDriveIntervalsRequireOnInterval(t *testing.T) {
	e := newTestScriptDriveExpectError(t, `
defineDrive(
  {
    createInstance: function () {
      return { intervals: [{ name: "t", interval: "1s" }] };
    }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`)
	if e == nil || !strings.Contains(e.Error(), "onInterval") {
		t.Fatalf("expected onInterval error, got %v", e)
	}
}

func TestScriptDriveIntervalRejectsDuplicateNames(t *testing.T) {
	e := newTestScriptDriveExpectError(t, `
defineDrive(
  {
    createInstance: function () {
      return {
        intervals: [
          { name: "t", interval: "1s" },
          { name: "t", interval: "2s" }
        ]
      };
    }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; },
    onInterval: function () {}
  }
);
`)
	if e == nil || !strings.Contains(e.Error(), "duplicate") {
		t.Fatalf("expected duplicate name error, got %v", e)
	}
}

func TestScriptDriveIntervalRejectsInvalidDuration(t *testing.T) {
	e := newTestScriptDriveExpectError(t, `
defineDrive(
  {
    createInstance: function () {
      return { intervals: [{ name: "t", interval: "nope" }] };
    }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; },
    onInterval: function () {}
  }
);
`)
	if e == nil || !strings.Contains(strings.ToLower(e.Error()), "duration") {
		t.Fatalf("expected duration error, got %v", e)
	}
}

func sharedInt(t *testing.T, d *ScriptDrive, key string) int {
	t.Helper()
	d.mu.RLock()
	encoded, ok := d.data[key]
	d.mu.RUnlock()
	if !ok {
		return 0
	}
	var n float64
	if e := json.Unmarshal(encoded, &n); e != nil {
		t.Fatal(e)
	}
	return int(n)
}

func waitSharedInt(t *testing.T, d *ScriptDrive, key string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last int
	for time.Now().Before(deadline) {
		last = sharedInt(t, d, key)
		if last >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s >= %d (last %d)", key, want, last)
}
