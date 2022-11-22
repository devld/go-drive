package search

import (
	"errors"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/event"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	"log"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"golang.org/x/net/context"
)

type IndexFilter = func(entry types.IEntry) bool

const (
	indexBatchSize  = 1000
	filterOptionKey = "search.filter"
	minimumPageSize = 10
	initialPageSize = 10
	pageSizeStep    = 10
	maxPageSize     = 100
)

type Service struct {
	s     Searcher
	drive *drive.RootDrive

	runner  task.Runner
	options *storage.OptionsDAO

	bus event.Bus
}

func NewService(ch *registry.ComponentsHolder, config common.Config, od *storage.OptionsDAO,
	rootDrive *drive.RootDrive, runner task.Runner, bus event.Bus) (*Service, error) {

	var s *Service = nil

	sConfig := config.Search
	if sConfig.Enabled {
		sf, e := GetSearcher(sConfig.Type)
		if e != nil {
			return nil, e
		}
		searcher, e := sf(config, sConfig.Config)
		if e != nil {
			return nil, e
		}

		s = &Service{
			s:       searcher,
			drive:   rootDrive,
			runner:  runner,
			options: od,
			bus:     bus,
		}
		s.bus.Subscribe(event.EntryUpdated, s.onUpdated)
		s.bus.Subscribe(event.EntryDeleted, s.onDeleted)
	} else {
		s = &Service{}
	}

	ch.Add("searchService", s)
	return s, nil
}

func (s *Service) checkEnabled() error {
	if s.s == nil {
		return err.NewNotAllowedMessageError("search is not enabled")
	}
	return nil
}

func (s *Service) Search(ctx context.Context, path string, query string,
	next int, perms utils.PermMap) (types.EntrySearchResult, error) {
	if e := s.checkEnabled(); e != nil {
		return types.EntrySearchResult{}, e
	}
	if next < 0 {
		next = 0
	}
	from := next
	size := initialPageSize
	var more bool
	result := make([]types.EntrySearchResultItem, 0, size)
	for {
		if e := ctx.Err(); e != nil {
			return types.EntrySearchResult{}, e
		}
		items, hasMore, e := s.search(path, query, from, size, perms)
		if e != nil {
			return types.EntrySearchResult{}, e
		}
		more = hasMore
		result = append(result, items...)
		if len(result) >= minimumPageSize || !hasMore {
			break
		}
		if size >= maxPageSize {
			break
		}
		from += size
		size += pageSizeStep
	}
	nextNext := -1
	if more {
		nextNext = from + size
	}
	return types.EntrySearchResult{
		Items: result,
		Next:  nextNext,
	}, nil
}

func (s *Service) TriggerIndexAll(path string, ignoreError bool) (task.Task, error) {
	if e := s.checkEnabled(); e != nil {
		return task.Task{}, e
	}
	return s.runner.Execute(func(ctx types.TaskCtx) (interface{}, error) {
		e := s.indexAll(ctx, path, ignoreError)
		if e != nil {
			log.Printf("Error indexing %s: %s", utils.LogSanitize(path), e)
		}
		return nil, e
	}, task.WithNameGroup(path, "search/index"))
}

func (s *Service) search(path, query string, from, size int,
	perms utils.PermMap) ([]types.EntrySearchResultItem, bool, error) {
	r, e := s.s.Search(path, query, from, size)
	if e != nil {
		return nil, false, e
	}
	if len(r) == 0 {
		// no more results
		return r, false, nil
	}
	items := make([]types.EntrySearchResultItem, 0, len(r))
	for _, item := range r {
		p := perms.ResolvePath(item.Entry.Path)
		if p.Readable() {
			items = append(items, item)
		}
	}
	return items, len(r) >= size, nil
}

func (s *Service) indexAll(ctx types.TaskCtx, path string, ignoreError bool) error {
	_ = s.s.DeleteDir(ctx, path)
	ctx.Total(0, true)
	ctx.Progress(0, true)
	filters, e := s.loadFilters()
	if e != nil {
		log.Println("[SearchService ] failed to get filter options", e.Error())
		return e
	}
	if filters == nil {
		return errors.New("no filters found")
	}
	items := make([]types.EntrySearchItem, 0, indexBatchSize)

	doIndex := func(ctx types.TaskCtx) error {
		if len(items) > 0 {
			e := s.s.Index(ctx, items)
			items = items[:0]
			return e
		}
		return nil
	}

	e = s.walk(ctx, s.drive.Get(), path, ignoreError, func(entry types.IEntry) error {
		if utils.IsRootPath(entry.Path()) {
			return nil
		}
		if isEntryExcluded(entry, filters) {
			return errSkip
		}
		items = append(items, s.mapEntry(entry))
		if len(items) >= indexBatchSize {
			return doIndex(ctx)
		}
		return nil
	})
	if e != nil {
		return e
	}
	return doIndex(ctx)
}

func (s *Service) Index(ctx types.TaskCtx, entry types.IEntry) error {
	if e := s.checkEnabled(); e != nil {
		return e
	}
	return s.s.Index(ctx, []types.EntrySearchItem{s.mapEntry(entry)})
}

func isEntryExcluded(entry types.IEntry, filters []string) bool {
	if len(filters) == 0 {
		return false
	}
	path := strings.ToLower(entry.Path())
	hasIncludes := false
	for _, f := range filters {
		t := f[0]
		p := f[1:]
		matched, e := doublestar.Match(p, path)
		if e != nil {
			log.Println("Warning: invalid filter pattern: ", f, e)
		}
		if t == '+' {
			hasIncludes = true
		}
		if matched {
			if t == '-' {
				return true
			}
			if t == '+' {
				return false
			}
		}
	}
	// if there are including patterns, but none of them matched, then it should be excluded
	return hasIncludes
}

func (s *Service) loadFilters() ([]string, error) {
	v, e := s.options.Get(filterOptionKey)
	if e != nil {
		return nil, e
	}

	filters := make([]string, 0)
	for _, f := range utils.SplitLines(v) {
		if f != "" && (f[0] == '+' || f[0] == '-') {
			filters = append(filters, strings.ToLower(f))
		}
	}
	return filters, nil
}

func (s *Service) mapEntry(e types.IEntry) types.EntrySearchItem {
	name := filepath.Base(e.Path())
	return types.EntrySearchItem{
		Path:    e.Path(),
		Name:    name,
		Ext:     utils.PathExt(name),
		Type:    e.Type(),
		Size:    e.Size(),
		ModTime: utils.Time(e.ModTime()),
	}
}

var errSkip = errors.New("skip")

func (s *Service) walk(ctx types.TaskCtx, d types.IDrive, rootPath string,
	ignoreError bool, visit func(entry types.IEntry) error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	entry, e := d.Get(ctx, rootPath)
	if e != nil {
		if ignoreError {
			log.Printf("failed to index %s: %s", utils.LogSanitize(rootPath), e)
			return nil
		}
		return e
	}
	if e = visit(entry); e != nil {
		if e == errSkip {
			return nil
		}
		log.Printf("failed to index %s: %s", utils.LogSanitize(rootPath), e)
		if !ignoreError {
			return e
		}
	}
	ctx.Total(1, false)
	if e == nil {
		ctx.Progress(1, false)
	}
	if entry.Type() == types.TypeDir {
		entries, e := d.List(ctx, rootPath)
		if e != nil {
			if ignoreError {
				log.Printf("failed to index %s: %s", utils.LogSanitize(rootPath), e)
				return nil
			}
			return e
		}
		for _, entry := range entries {
			e = s.walk(ctx, d, entry.Path(), ignoreError, visit)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

func (s *Service) onUpdated(dc types.DriveListenerContext, path string, includeDescendants bool) {
	if s.checkEnabled() != nil {
		return
	}
	if includeDescendants {
		_, _ = s.TriggerIndexAll(path, true)
	} else {
		_, _ = s.runner.Execute(func(ctx types.TaskCtx) (interface{}, error) {
			entry, e := dc.Drive.Get(ctx, path)
			if e != nil {
				return nil, e
			}
			e = s.Index(ctx, entry)
			if e != nil {
				log.Printf("Error indexing %s: %s", path, e)
			}
			return nil, e
		})
	}
}

func (s *Service) onDeleted(dc types.DriveListenerContext, path string) {
	if s.checkEnabled() != nil {
		return
	}
	_, _ = s.runner.Execute(func(ctx types.TaskCtx) (interface{}, error) {
		e := s.s.DeleteDir(ctx, path)
		if e != nil {
			log.Printf("Error deleting index %s: %s", utils.LogSanitize(path), e)
		}
		return nil, e
	}, task.WithNameGroup(path, "search/delete"))
}

func (s *Service) Status() (string, types.SM, error) {
	if s.checkEnabled() != nil {
		return "Search", types.SM{}, nil
	}
	stats, e := s.s.Stats()
	if e != nil {
		return "", nil, e
	}
	return "Search", stats, nil
}

func (s *Service) Dispose() error {
	if s.checkEnabled() != nil {
		return nil
	}
	s.bus.Unsubscribe(event.EntryUpdated, s.onUpdated)
	s.bus.Unsubscribe(event.EntryDeleted, s.onDeleted)
	return s.s.Dispose()
}

func (s *Service) SysConfig() (string, types.M, error) {
	var examples []string
	if s.s != nil {
		examples = s.s.Examples()
	}
	return "search", types.M{
		"enabled":  s.checkEnabled() == nil,
		"examples": examples,
	}, nil
}
