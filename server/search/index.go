package search

import (
	"encoding/json"
	"errors"
	"go-drive/common"
	"go-drive/common/event"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	"golang.org/x/net/context"
	"log"
	"path"
	"path/filepath"
	"strconv"
)

type IndexFilter = func(entry types.IEntry) bool

const (
	searchIndexName = "files.index"
	filterOptionKey = "search.filter"
	initialPageSize = 10
	pageSizeStep    = 10
	maxPageSize     = 100
)

type Service struct {
	es    *EntrySearcher
	drive *drive.RootDrive

	runner  task.Runner
	options *storage.OptionsDAO

	bus event.Bus
}

func NewService(ch *registry.ComponentsHolder, config common.Config, od *storage.OptionsDAO,
	rootDrive *drive.RootDrive, runner task.Runner, bus event.Bus) (*Service, error) {

	indexPath, _ := config.GetDir(searchIndexName, false)
	searcher, e := NewEntrySearcher(indexPath)
	if e != nil {
		return nil, e
	}

	s := &Service{
		es:      searcher,
		drive:   rootDrive,
		runner:  runner,
		options: od,
		bus:     bus,
	}
	ch.Add("searchService", s)

	s.bus.Subscribe(event.EntryUpdated, s.onUpdated)
	s.bus.Subscribe(event.EntryDeleted, s.onDeleted)

	return s, nil
}

func (s *Service) Search(ctx context.Context, path string, query string,
	next int, perms utils.PermMap) (types.EntrySearchResult, error) {
	if next < 0 {
		next = 0
	}
	from := next
	size := initialPageSize
	var more bool
	var items []types.EntrySearchResultItem
	var e error
	for {
		if e := ctx.Err(); e != nil {
			return types.EntrySearchResult{}, e
		}
		items, more, e = s.search(path, query, from, size, perms)
		if e != nil {
			return types.EntrySearchResult{}, e
		}
		if len(items) > 0 || !more {
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
		Items: items,
		Next:  nextNext,
	}, nil
}

func (s *Service) TriggerIndexAll(path string, ignoreError bool) (task.Task, error) {
	return s.runner.Execute(func(ctx types.TaskCtx) (interface{}, error) {
		e := s.indexAll(ctx, path, ignoreError)
		if e != nil {
			log.Printf("Error indexing %s: %s", path, e)
		}
		return nil, e
	}, task.WithNameGroup(path, "search/index"))
}

func (s *Service) search(path, query string, from, size int,
	perms utils.PermMap) ([]types.EntrySearchResultItem, bool, error) {
	r, e := s.es.Search(path, query, from, size)
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
	_ = s.es.DeleteDir(ctx, path)
	ctx.Total(0, true)
	ctx.Progress(0, true)
	filters := s.loadFilters()
	return s.walk(ctx, s.drive.Get(), path, ignoreError, func(entry types.IEntry) error {
		if isEntryExcluded(entry, filters) {
			return skip
		}
		return s.es.Index(s.mapEntry(entry))
	})
}

func (s *Service) Index(entry types.IEntry) error {
	return s.es.Index(s.mapEntry(entry))
}

func isEntryExcluded(entry types.IEntry, filter *searchFilter) bool {
	if filter == nil {
		// exclude all if filter is not loaded
		return true
	}
	include := false
	if filter.Includes != nil && len(filter.Includes) > 0 {
		for _, p := range filter.Includes {
			matched, e := path.Match(p, entry.Path())
			if e == nil && matched {
				include = true
				break
			}
		}
	} else {
		include = true
	}
	if !include {
		return true
	}
	if filter.Excludes != nil && len(filter.Excludes) > 0 {
		for _, p := range filter.Excludes {
			matched, e := path.Match(p, entry.Path())
			if e == nil && matched {
				return true
			}
		}
	}
	return !include
}

func (s *Service) loadFilters() *searchFilter {
	v, e := s.options.Get(filterOptionKey)
	if e != nil {
		log.Printf("[SearchService ] failed to get filter options, skip all entries: %s", e.Error())
		return nil
	}
	if v == "" {
		return &searchFilter{}
	}
	f := &searchFilter{}
	e = json.Unmarshal([]byte(v), f)
	if e != nil {
		log.Printf("[SearchService ] failed to parse filter options, skip all entries: %s", e.Error())
		return nil
	}
	return f
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

var skip = errors.New("skip this dir")

func (s *Service) walk(ctx types.TaskCtx, d types.IDrive, rootPath string,
	ignoreError bool, visit func(entry types.IEntry) error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	entry, e := d.Get(ctx, rootPath)
	if e != nil {
		if ignoreError {
			return nil
		}
		return e
	}
	if e := visit(entry); e != nil {
		if e == skip {
			return nil
		}
		if ignoreError {
			return nil
		}
		return e
	}
	ctx.Total(1, false)
	ctx.Progress(1, false)

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

func (s *Service) onUpdated(_ types.DriveListenerContext, path string, includeDescendants bool) {
	if includeDescendants {
		_, _ = s.TriggerIndexAll(path, true)
	} else {
		_, _ = s.runner.Execute(func(ctx types.TaskCtx) (interface{}, error) {
			entry, e := s.drive.Get().Get(ctx, path)
			if e != nil {
				return nil, e
			}
			e = s.Index(entry)
			if e != nil {
				log.Printf("Error indexing %s: %s", path, e)
			}
			return nil, e
		})
	}
}

func (s *Service) onDeleted(_ types.DriveListenerContext, path string) {
	_, _ = s.runner.Execute(func(ctx types.TaskCtx) (interface{}, error) {
		e := s.es.DeleteDir(ctx, path)
		if e != nil {
			log.Printf("Error deleting index %s: %s", utils.LogSanitize(path), e)
		}
		return nil, e
	}, task.WithNameGroup(path, "search/delete"))
}

func (s *Service) Status() (string, types.SM, error) {
	stats, e := s.es.Stats()
	if e != nil {
		return "", nil, e
	}
	return "Search", types.SM{
		"Total":      strconv.FormatUint(stats.Total, 10),
		"Searches":   strconv.FormatUint(stats.Searches, 10),
		"SearchTime": strconv.FormatUint(stats.SearchTime, 10),
	}, nil
}

func (s *Service) Dispose() error {
	s.bus.Unsubscribe(event.EntryUpdated, s.onUpdated)
	s.bus.Unsubscribe(event.EntryDeleted, s.onDeleted)
	return s.es.Dispose()
}

type searchFilter struct {
	Includes []string
	Excludes []string
}
