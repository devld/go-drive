package search

import (
	"errors"
	"go-drive/common"
	"go-drive/common/types"
	"strings"
)

var searcherRegistry = make(map[string]SearcherFactory)

func RegisterSearcher(name string, factory SearcherFactory) {
	if _, ok := searcherRegistry[name]; ok {
		panic("Searcher already registered: " + name)
	}
	searcherRegistry[name] = factory
}

func GetSearcher(name string) (SearcherFactory, error) {
	factory, ok := searcherRegistry[name]
	if !ok {
		ss := make([]string, 0, len(searcherRegistry))
		for k := range searcherRegistry {
			ss = append(ss, k)
		}
		return nil, errors.New("Searcher not found: " + name + ", available searchers: " + strings.Join(ss, ", "))
	}
	return factory, nil
}

type SearcherFactory func(config common.Config, searcherConfig types.SM) (Searcher, error)

type Searcher interface {
	Search(path string, query string, from, size int) ([]types.EntrySearchResultItem, error)
	// Index add or update an entry to the index
	Index(ctx types.TaskCtx, entries []types.EntrySearchItem) error
	// Delete remove an entry from the index
	Delete(path string) error
	// DeleteDir remove all entries in the dir from the index
	DeleteDir(ctx types.TaskCtx, dirPath string) error

	// Examples returns a list of example search queries
	Examples() []string
	Stats() (types.SM, error)
	Dispose() error
}
