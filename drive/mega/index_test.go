package mega

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"

	megaapi "go-drive/drive/mega/internal/gomega"
)

func TestDriveFileLifecycle(t *testing.T) {
	remote := newMemClient()
	drive := newDrive(remote, "", false)
	ctx := task.NewTaskContext(context.Background())

	dir, e := drive.MakeDir(ctx, "docs")
	if e != nil {
		t.Fatalf("MakeDir: %v", e)
	}
	if dir.Type() != types.TypeDir || dir.Path() != "docs" {
		t.Fatalf("directory = %+v", dir)
	}

	saved, e := drive.Save(ctx, "docs/note.txt", 5, false, bytes.NewReader([]byte("hello")))
	if e != nil {
		t.Fatalf("Save: %v", e)
	}
	if saved.Size() != 5 || saved.Name() != "note.txt" {
		t.Fatalf("saved = size %d name %s", saved.Size(), saved.Name())
	}
	if got := ctx.GetProgress(); got != 5 {
		t.Fatalf("progress = %d", got)
	}
	if body := readAll(t, saved, -1, -1); body != "hello" {
		t.Fatalf("body = %q", body)
	}

	if _, e = drive.Save(ctx, "docs/note.txt", 4, false, bytes.NewReader([]byte("nope"))); !err.IsNotAllowedError(e) {
		t.Fatalf("override=false error = %v", e)
	}
	replaced, e := drive.Save(ctx, "docs/note.txt", 7, true, bytes.NewReader([]byte("updated")))
	if e != nil {
		t.Fatalf("override Save: %v", e)
	}
	if body := readAll(t, replaced, -1, -1); body != "updated" {
		t.Fatalf("replaced body = %q", body)
	}
	entries, e := drive.List(ctx, "docs")
	if e != nil {
		t.Fatalf("List: %v", e)
	}
	if len(entries) != 1 || entries[0].Name() != "note.txt" {
		t.Fatalf("entries = %#v", names(entries))
	}

	if _, e = drive.Move(ctx, saved, "docs", true); !err.IsNotAllowedError(e) {
		t.Fatalf("move into parent error = %v", e)
	}
	moved, e := drive.Move(ctx, replaced, "docs/renamed.txt", false)
	if e != nil {
		t.Fatalf("Move: %v", e)
	}
	if moved.Path() != "docs/renamed.txt" {
		t.Fatalf("moved path = %s", moved.Path())
	}
	if _, e = drive.Copy(ctx, moved, "docs/copy.txt", false); !err.IsUnsupportedError(e) {
		t.Fatalf("Copy error = %v", e)
	}
	if e = drive.Delete(ctx, "docs"); e != nil {
		t.Fatalf("Delete: %v", e)
	}
	if _, e = drive.Get(ctx, "docs/renamed.txt"); !err.IsNotFoundError(e) {
		t.Fatalf("deleted get error = %v", e)
	}
	if e = drive.Delete(ctx, ""); e == nil {
		t.Fatal("Delete root succeeded")
	}
}

func TestDriveDuplicateNameUsesNewest(t *testing.T) {
	remote := newMemClient()
	remote.now = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	drive := newDrive(remote, "", false)
	ctx := context.Background()
	parent, e := drive.anchor()
	if e != nil {
		t.Fatalf("anchor: %v", e)
	}
	if _, e = remote.put(ctx, parent.id, "same.txt", 3, bytes.NewReader([]byte("old")), nil); e != nil {
		t.Fatalf("old put: %v", e)
	}
	remote.now = remote.now.Add(time.Minute)
	if _, e = remote.put(ctx, parent.id, "same.txt", 3, bytes.NewReader([]byte("new")), nil); e != nil {
		t.Fatalf("new put: %v", e)
	}
	if _, e = remote.put(ctx, parent.id, "a/bad", 1, bytes.NewReader([]byte("x")), nil); e != nil {
		t.Fatalf("invalid put: %v", e)
	}

	entries, e := drive.List(ctx, "")
	if e != nil {
		t.Fatalf("List: %v", e)
	}
	if len(entries) != 1 || entries[0].Name() != "same.txt" {
		t.Fatalf("entries = %#v", names(entries))
	}
	if body := readAll(t, entries[0], -1, -1); body != "new" {
		t.Fatalf("body = %q", body)
	}
}

func TestDriveRootPathAndRange(t *testing.T) {
	remote := newMemClient()
	ctx := task.NewTaskContext(context.Background())
	full := newDrive(remote, "", false)
	if _, e := full.MakeDir(ctx, "cloud"); e != nil {
		t.Fatalf("MakeDir cloud: %v", e)
	}
	if _, e := full.Save(ctx, "cloud/data.bin", 10, false, bytes.NewReader([]byte("abcdefghij"))); e != nil {
		t.Fatalf("Save: %v", e)
	}
	if _, e := full.Save(ctx, "outside.txt", 1, false, bytes.NewReader([]byte("z"))); e != nil {
		t.Fatalf("Save outside: %v", e)
	}

	rooted := newDrive(remote, "/cloud", false)
	entries, e := rooted.List(ctx, "")
	if e != nil {
		t.Fatalf("rooted List: %v", e)
	}
	if len(entries) != 1 || entries[0].Path() != "data.bin" {
		t.Fatalf("rooted entries = %#v", names(entries))
	}
	if _, e = rooted.Get(ctx, "outside.txt"); !err.IsNotFoundError(e) {
		t.Fatalf("outside error = %v", e)
	}
	entry, e := rooted.Get(ctx, "data.bin")
	if e != nil {
		t.Fatalf("Get: %v", e)
	}
	if body := readAll(t, entry, 2, 4); body != "cdef" {
		t.Fatalf("range = %q", body)
	}
	readFrom := len(remote.reads)
	if body := readAll(t, entry, 6, 3); body != "ghi" {
		t.Fatalf("tail range = %q", body)
	}
	if remote.reads[readFrom] != 2 {
		t.Fatalf("first chunk read = %d, want the chunk covering offset 6", remote.reads[readFrom])
	}
	if _, e = newDrive(remote, "missing", false).anchor(); !err.IsNotFoundError(e) {
		t.Fatalf("missing root error = %v", e)
	}
}

func TestReaderSeekReusesContentCache(t *testing.T) {
	dir := t.TempDir()
	pool, e := driveutil.NewCacheFilePool(driveutil.CacheFilePoolOptions{
		MaxEntries: 2,
		Dir:        dir,
		BlockSize:  contentBlockSize,
	})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = pool.Dispose() })
	remote := newMemClient()
	drive := newDrive(remote, "", false)
	drive.content = pool
	ctx := task.NewTaskContext(context.Background())
	if _, e = drive.Save(ctx, "data.bin", 10, false, bytes.NewReader([]byte("abcdefghij"))); e != nil {
		t.Fatalf("Save: %v", e)
	}
	entry, e := drive.Get(ctx, "data.bin")
	if e != nil {
		t.Fatalf("Get: %v", e)
	}
	readSeek := func(at int64, n int) string {
		t.Helper()
		reader, readErr := entry.GetReader(context.Background(), types.FullReaderRange())
		if readErr != nil {
			t.Fatalf("GetReader: %v", readErr)
		}
		defer reader.Close()
		seeker, ok := reader.(io.Seeker)
		if !ok {
			t.Fatal("reader cannot seek")
		}
		if _, readErr = seeker.Seek(at, io.SeekStart); readErr != nil {
			t.Fatalf("Seek: %v", readErr)
		}
		buf := make([]byte, n)
		if _, readErr = io.ReadFull(reader, buf); readErr != nil {
			t.Fatalf("Read: %v", readErr)
		}
		return string(buf)
	}
	if got := readSeek(2, 4); got != "cdef" {
		t.Fatalf("seek read = %q", got)
	}
	fetched := len(remote.reads)
	if fetched == 0 {
		t.Fatal("content was not fetched")
	}
	if got := readSeek(6, 3); got != "ghi" {
		t.Fatalf("second seek read = %q", got)
	}
	if got := readAll(t, entry, 6, 3); got != "ghi" {
		t.Fatalf("ranged read = %q", got)
	}
	if len(remote.reads) != fetched {
		t.Fatalf("cached reads fetched more chunks: %v", remote.reads)
	}
}

func TestRangedReadFillsCacheFromMega(t *testing.T) {
	dir := t.TempDir()
	pool, e := driveutil.NewCacheFilePool(driveutil.CacheFilePoolOptions{
		MaxEntries: 2,
		Dir:        dir,
		BlockSize:  6,
	})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = pool.Dispose() })
	remote := newMemClient()
	drive := newDrive(remote, "", false)
	drive.content = pool
	ctx := task.NewTaskContext(context.Background())
	body := []byte("abcdefghijklmnopqrst")
	if _, e = drive.Save(ctx, "data.bin", int64(len(body)), false, bytes.NewReader(body)); e != nil {
		t.Fatalf("Save: %v", e)
	}
	entry, e := drive.Get(ctx, "data.bin")
	if e != nil {
		t.Fatalf("Get: %v", e)
	}
	if got := readAll(t, entry, 8, 4); got != "ijkl" {
		t.Fatalf("ranged read = %q", got)
	}
	for _, index := range remote.reads {
		if index < 2 {
			t.Fatalf("fill read chunks before the requested block: %v", remote.reads)
		}
	}
	fetched := len(remote.reads)
	if got := readAll(t, entry, 10, 2); got != "kl" {
		t.Fatalf("cached ranged read = %q", got)
	}
	if len(remote.reads) != fetched {
		t.Fatalf("second range fetched more chunks: %v", remote.reads)
	}
}

func TestDriveListSeesNodesAddedOnTheClient(t *testing.T) {
	remote := newMemClient()
	drive := newDrive(remote, "", false)
	ctx := task.NewTaskContext(context.Background())
	parent, e := drive.anchor()
	if e != nil {
		t.Fatalf("anchor: %v", e)
	}
	if _, e = remote.put(ctx, parent.id, "late.txt", 1, bytes.NewReader([]byte("x")), nil); e != nil {
		t.Fatalf("put: %v", e)
	}
	entries, e := drive.List(ctx, "")
	if e != nil {
		t.Fatalf("List: %v", e)
	}
	if len(entries) != 1 || entries[0].Name() != "late.txt" {
		t.Fatalf("entries = %#v", names(entries))
	}
}

func TestRegisterDriveExposesMegaForm(t *testing.T) {
	driveRegistry := driveutil.NewDriveRegistry()
	RegisterDrive(driveRegistry)
	config := driveRegistry.GetDrive("mega")
	if config == nil || config.Factory.Create == nil || config.Factory.InitConfig == nil || config.Factory.Init == nil {
		t.Fatal("mega drive was not registered")
	}
	required := map[string]bool{}
	for _, item := range config.ConfigForm {
		required[item.Field] = item.Required
	}
	if !required["email"] || !required["password"] {
		t.Fatalf("form = %#v", config.ConfigForm)
	}
	if _, ok := required["mfa"]; ok {
		t.Fatal("mfa is stored in the drive config")
	}
	if required["https"] {
		t.Fatal("optional fields were marked required")
	}
	for _, item := range config.ConfigForm {
		if item.Field == "delete_mode" {
			if item.Type != "checkbox" || item.Required || item.DefaultValue != "" {
				t.Fatalf("delete_mode = %#v", item)
			}
		}
	}
}

func TestPermanentDeleteDefaultsToRubbishBin(t *testing.T) {
	if permanentDelete(nil) || permanentDelete(types.SM{}) || permanentDelete(types.SM{"delete_mode": "trash"}) {
		t.Fatal("unchecked delete mode destroys files")
	}
	if !permanentDelete(types.SM{"delete_mode": "1"}) || !permanentDelete(types.SM{"delete_mode": "permanent"}) {
		t.Fatal("checked delete mode does not destroy files")
	}
}

func TestDeleteModeMovesToTrashOrDestroys(t *testing.T) {
	ctx := task.NewTaskContext(context.Background())
	trashRemote := newMemClient()
	trashDrive := newDrive(trashRemote, "", false)
	if _, e := trashDrive.Save(ctx, "note.txt", 4, false, bytes.NewReader([]byte("keep"))); e != nil {
		t.Fatalf("Save: %v", e)
	}
	if _, e := trashDrive.Save(ctx, "note.txt", 3, true, bytes.NewReader([]byte("new"))); e != nil {
		t.Fatalf("override Save: %v", e)
	}
	if parentOf(trashRemote, "keep") != trashRemote.trashID {
		t.Fatalf("replaced file parent = %q", parentOf(trashRemote, "keep"))
	}
	if e := trashDrive.Delete(ctx, "note.txt"); e != nil {
		t.Fatalf("Delete: %v", e)
	}
	if _, e := trashDrive.Get(ctx, "note.txt"); !err.IsNotFoundError(e) {
		t.Fatalf("trashed get error = %v", e)
	}
	if parentOf(trashRemote, "new") != trashRemote.trashID {
		t.Fatalf("deleted file parent = %q", parentOf(trashRemote, "new"))
	}

	permanentRemote := newMemClient()
	permanentDrive := newDrive(permanentRemote, "", true)
	if _, e := permanentDrive.Save(ctx, "note.txt", 4, false, bytes.NewReader([]byte("gone"))); e != nil {
		t.Fatalf("Save: %v", e)
	}
	if e := permanentDrive.Delete(ctx, "note.txt"); e != nil {
		t.Fatalf("permanent Delete: %v", e)
	}
	if parentOf(permanentRemote, "gone") != "" {
		t.Fatal("permanently deleted file is still present")
	}
}

func parentOf(remote *memClient, content string) string {
	remote.mu.Lock()
	defer remote.mu.Unlock()
	for _, item := range remote.nodes {
		if string(item.content) == content {
			return item.parent
		}
	}
	return ""
}

func TestMapError(t *testing.T) {
	if !err.IsNotFoundError(mapError(megaapi.ENOENT)) {
		t.Fatal("ENOENT was not mapped to not found")
	}
	if !err.IsNotAllowedError(mapError(megaapi.EOVERQUOTA)) {
		t.Fatal("quota was not mapped to not allowed")
	}
}

func readAll(t *testing.T, entry types.IEntry, start, size int64) string {
	t.Helper()
	reader, e := entry.GetReader(context.Background(), types.ReaderRange{Start: start, Size: size})
	if e != nil {
		t.Fatalf("GetReader: %v", e)
	}
	defer reader.Close()
	body, e := io.ReadAll(reader)
	if e != nil {
		t.Fatalf("ReadAll: %v", e)
	}
	if e = reader.Close(); e != nil {
		t.Fatalf("Close: %v", e)
	}
	return string(body)
}

func names(entries []types.IEntry) []string {
	result := make([]string, len(entries))
	for i, entry := range entries {
		result[i] = entry.Path()
	}
	return result
}

type memClient struct {
	mu      sync.Mutex
	nodes   map[string]*memNode
	seq     int
	now     time.Time
	reads   []int
	rootID  string
	trashID string
}

type memNode struct {
	node
	parent  string
	content []byte
}

func newMemClient() *memClient {
	client := &memClient{nodes: map[string]*memNode{}, rootID: "root", trashID: "trash"}
	client.nodes[client.rootID] = &memNode{node: node{id: client.rootID, isDir: true}}
	client.nodes[client.trashID] = &memNode{node: node{id: client.trashID, name: "bin", isDir: true}}
	return client
}

func (m *memClient) close() error { return nil }

func (m *memClient) root() (node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nodes[m.rootID].node, nil
}

func (m *memClient) children(parentID string) ([]node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[parentID]; !ok {
		return nil, megaapi.ENOENT
	}
	var result []node
	for _, item := range m.nodes {
		if item.parent == parentID {
			result = append(result, item.node)
		}
	}
	return result, nil
}

func (m *memClient) mkdir(parentID, name string) (node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[parentID]; !ok {
		return node{}, megaapi.ENOENT
	}
	return m.addLocked(parentID, name, nil, true), nil
}

func (m *memClient) put(_ context.Context, parentID, name string, _ int64, reader io.Reader, progress func(int64)) (node, error) {
	body, e := io.ReadAll(reader)
	if e != nil {
		return node{}, e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[parentID]; !ok {
		return node{}, megaapi.ENOENT
	}
	created := m.addLocked(parentID, name, body, false)
	if progress != nil {
		progress(int64(len(body)))
	}
	return created, nil
}

func (m *memClient) open(id string) (downloader, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.nodes[id]
	if !ok || item.isDir {
		return nil, megaapi.ENOENT
	}
	return &memDownload{client: m, content: append([]byte(nil), item.content...)}, nil
}

func (m *memClient) move(id, parentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.nodes[id]
	if !ok {
		return megaapi.ENOENT
	}
	if _, ok = m.nodes[parentID]; !ok {
		return megaapi.ENOENT
	}
	item.parent = parentID
	return nil
}

func (m *memClient) rename(id, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.nodes[id]
	if !ok {
		return megaapi.ENOENT
	}
	item.name = name
	return nil
}

func (m *memClient) remove(id string, permanent bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.nodes[id]
	if !ok {
		return megaapi.ENOENT
	}
	if !permanent {
		item.parent = m.trashID
		return nil
	}
	var drop []string
	for _, candidate := range m.nodes {
		if candidate.id == id || m.hasAncestor(candidate.id, id) {
			drop = append(drop, candidate.id)
		}
	}
	for _, itemID := range drop {
		delete(m.nodes, itemID)
	}
	return nil
}

func (m *memClient) hasAncestor(id, ancestor string) bool {
	for id != "" {
		item := m.nodes[id]
		if item == nil {
			return false
		}
		if item.parent == ancestor {
			return true
		}
		id = item.parent
	}
	return false
}

func (m *memClient) addLocked(parentID, name string, content []byte, isDir bool) node {
	m.seq++
	created := node{
		id:      "n" + itoa(m.seq),
		name:    name,
		isDir:   isDir,
		size:    int64(len(content)),
		modTime: m.clock(),
	}
	m.nodes[created.id] = &memNode{node: created, parent: parentID, content: append([]byte(nil), content...)}
	return created
}

func (m *memClient) clock() time.Time {
	if m.now.IsZero() {
		return time.Now()
	}
	return m.now
}

type memDownload struct {
	client  *memClient
	content []byte
}

func (d *memDownload) chunkCount() int {
	if len(d.content) == 0 {
		return 1
	}
	return (len(d.content) + 2) / 3
}

func (d *memDownload) chunkAt(index int) (int64, int, error) {
	if index < 0 || index >= d.chunkCount() {
		return 0, 0, megaapi.EARGS
	}
	position := int64(index * 3)
	size := 3
	if int(position)+size > len(d.content) {
		size = len(d.content) - int(position)
	}
	return position, size, nil
}

func (d *memDownload) readChunk(index int) ([]byte, error) {
	position, size, e := d.chunkAt(index)
	if e != nil {
		return nil, e
	}
	d.client.mu.Lock()
	d.client.reads = append(d.client.reads, index)
	d.client.mu.Unlock()
	return append([]byte(nil), d.content[position:position+int64(size)]...), nil
}

func (d *memDownload) finish() error { return nil }

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
