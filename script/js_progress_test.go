package script

import (
	"context"
	"strings"
	"testing"

	"go-drive/common/task"
)

func TestProgressReporterCapabilities(t *testing.T) {
	vm := newScriptTestVM(t)
	ctx := task.NewTaskContext(context.Background())
	p := NewProgressReporter(ctx, true, true)
	mustDefineGlobal(t, vm, "progress", p)

	got, e := vm.Run(context.Background(), `
		const loaded = progress.derive({loaded: true});
		const total = progress.derive({total: true});
		loaded.addLoaded(3);
		total.addTotal(7);
		progress.addLoaded(2);
		progress.addTotal(4);
		progress.extra = true;
		loaded.addLoaded = function () {};
		progress instanceof ProgressReporter &&
		loaded instanceof ProgressReporter &&
		typeof progress.extra === "undefined" &&
			typeof loaded.addLoaded === "function";
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if !got.Bool() {
		t.Fatal("ProgressReporter instances must reject JavaScript mutation")
	}
	if ctx.GetProgress() != 5 || ctx.GetTotal() != 11 {
		t.Fatalf("progress = %d/%d, want 5/11", ctx.GetProgress(), ctx.GetTotal())
	}
}

func TestProgressReporterRejectsInvalidUse(t *testing.T) {
	vm := newScriptTestVM(t)
	ctx := task.NewTaskContext(context.Background())
	p := NewProgressReporter(ctx, true, false)
	mustDefineGlobal(t, vm, "progress", p)

	for _, source := range []string{
		`new ProgressReporter()`,
		`progress.addTotal(1)`,
		`progress.derive({loaded: true, total: true})`,
		`progress.derive({loaded: 1})`,
		`progress.addLoaded(-1)`,
		`progress.addLoaded(1.5)`,
		`progress.addLoaded(Number.MAX_SAFE_INTEGER + 1)`,
		`progress.addLoaded(1, 2)`,
	} {
		if _, e := vm.Run(context.Background(), source, ""); e == nil {
			t.Fatalf("%s did not fail", source)
		}
	}

	if e := vm.Dispose(); e != nil {
		t.Fatal(e)
	}
	if e := p.addLoaded(1); e == nil || !strings.Contains(e.Error(), "no longer active") {
		t.Fatalf("reporter after VM dispose error = %v", e)
	}
}

func TestReaderWithProgressIsExplicitAndCannotNest(t *testing.T) {
	vm := newScriptTestVM(t)
	ctx := task.NewTaskContext(context.Background())
	p := NewProgressReporter(ctx, true, false)
	mustDefineGlobal(t, vm, "progress", p)

	if _, e := vm.Run(context.Background(), `
		const plain = new TempFile();
		plain.write(Bytes.fromString("plain"));
		plain.seekTo(0, SEEK_START);
		plain.readAsString();
		plain.close();

		const source = new TempFile();
		source.write(Bytes.fromString("hello"));
		source.seekTo(0, SEEK_START);
			const reported = source.withProgress(progress);
			if (reported.readAsString() !== "hello") throw new Error("incorrect content");
			let rejected = false;
			try { reported.withProgress(progress); } catch (e) { rejected = true; }
			if (!rejected) throw new Error("nested progress reader accepted");
			rejected = false;
			try { reported.limitReader(1).withProgress(progress); } catch (e) { rejected = true; }
			if (!rejected) throw new Error("limited nested progress reader accepted");
		source.close();
	`, ""); e != nil {
		t.Fatal(e)
	}
	if ctx.GetProgress() != 5 || ctx.GetTotal() != 0 {
		t.Fatalf("progress = %d/%d, want 5/0", ctx.GetProgress(), ctx.GetTotal())
	}
}

func TestTempFileCopyFromUsesReaderProgressOnly(t *testing.T) {
	vm := newScriptTestVM(t)
	ctx := task.NewTaskContext(context.Background())
	p := NewProgressReporter(ctx, true, false)
	mustDefineGlobal(t, vm, "progress", p)

	if _, e := vm.Run(context.Background(), `
		const source = new TempFile();
		source.write(Bytes.fromString("abc"));
		source.seekTo(0, SEEK_START);
		const plain = new TempFile();
		plain.copyFrom(source);

		source.seekTo(0, SEEK_START);
		const reported = new TempFile();
		reported.copyFrom(source.withProgress(progress));
		source.close();
		plain.close();
		reported.close();
	`, ""); e != nil {
		t.Fatal(e)
	}
	if ctx.GetProgress() != 3 {
		t.Fatalf("copyFrom progress = %d, want 3", ctx.GetProgress())
	}
}

func TestBuildEntriesTreeUsesExplicitTotalReporter(t *testing.T) {
	vm := newScriptTestVM(t)
	entry := sampleInspectEntry()
	entry.ClassHost = NewClassHost(vm)
	mustDefineGlobal(t, vm, "entry", entry)
	ctx := task.NewTaskContext(context.Background())
	p := NewProgressReporter(ctx, false, true)
	mustDefineGlobal(t, vm, "progress", p)

	if _, e := vm.Run(context.Background(), `
		buildEntriesTree(entry, false);
		buildEntriesTree(entry, false, progress);
	`, ""); e != nil {
		t.Fatal(e)
	}
	if ctx.GetProgress() != 0 || ctx.GetTotal() != 1 {
		t.Fatalf("tree progress = %d/%d, want 0/1", ctx.GetProgress(), ctx.GetTotal())
	}
}
