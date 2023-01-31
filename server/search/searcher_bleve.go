package search

import (
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"regexp"
	"strconv"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
)

func init() {
	RegisterSearcher("bleve", NewBleveSearcher)
}

type BleveSearcher struct {
	index bleve.Index
}

func NewBleveSearcher(config common.Config, searcherConfig types.SM) (Searcher, error) {
	indexName := searcherConfig["index"]
	if indexName == "" {
		indexName = "files.index"
	}
	indexPath, e := config.GetDir(indexName, false)
	if e != nil {
		return nil, e
	}

	var index bleve.Index
	if exists, _ := utils.FileExists(indexPath); exists {
		index, e = bleve.Open(indexPath)
	} else {
		index, e = bleve.New(indexPath, createMapping())
	}
	if e != nil {
		return nil, e
	}
	return &BleveSearcher{index: index}, nil
}

func createMapping() mapping.IndexMapping {
	m := bleve.NewIndexMapping()
	entryMapping := bleve.NewDocumentMapping()

	entryMapping.AddFieldMappingsAt("path", bleve.NewTextFieldMapping(), bleve.NewKeywordFieldMapping())
	entryMapping.AddFieldMappingsAt("name", bleve.NewTextFieldMapping(), bleve.NewKeywordFieldMapping())
	entryMapping.AddFieldMappingsAt("ext", bleve.NewKeywordFieldMapping())
	entryMapping.AddFieldMappingsAt("type", bleve.NewKeywordFieldMapping())
	entryMapping.AddFieldMappingsAt("size", bleve.NewNumericFieldMapping())
	entryMapping.AddFieldMappingsAt("modTime", bleve.NewDateTimeFieldMapping())

	m.DefaultMapping = entryMapping
	return m
}

var sizePattern = regexp.MustCompile("(size:)([>=<]*)(([0-9]+)([bkmgtBKMGT])?)")

func (s *BleveSearcher) processQuery(query string) string {
	query = sizePattern.ReplaceAllStringFunc(query, func(s string) string {
		g := sizePattern.FindStringSubmatch(s)
		size := types.SV(g[3]).DataSize(-1)
		if size < 0 {
			return ""
		}
		return g[1] + g[2] + strconv.FormatInt(size, 10)
	})
	return query
}

func (s *BleveSearcher) Search(path string, query string, from, size int) ([]types.EntrySearchResultItem, error) {
	if !utils.IsRootPath(path) {
		path += "/"
	}

	pq := bleve.NewPrefixQuery(path)
	pq.SetField("path")

	bqn := bleve.NewBooleanQuery()
	// exclude this query from highlights
	bqn.AddMustNot(pq)

	qq := bleve.NewQueryStringQuery(s.processQuery(query))

	bq := bleve.NewBooleanQuery()
	bq.AddMustNot(bqn)
	bq.AddMust(qq)

	sr := bleve.NewSearchRequestOptions(bq, size, from, false)

	sr.Fields = []string{"*"}

	sr.Highlight = bleve.NewHighlight()
	sr.Highlight.AddField("path")
	sr.Highlight.AddField("name")

	result, e := s.index.Search(sr)
	if e != nil {
		if e.Error() == "syntax error" {
			return nil, err.NewBadRequestError(i18n.T("search.invalid_query"))
		}
		return nil, e
	}

	return mapSearchResultItem(result.Hits), nil
}

func (s *BleveSearcher) Index(ctx types.TaskCtx, entries []types.EntrySearchItem) error {
	for _, entry := range entries {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := s.index.Index(entry.Path, entry); e != nil {
			return e
		}
	}
	return nil
}

// Delete remove all entries in the dir(or single file) from the index
func (s *BleveSearcher) Delete(ctx types.TaskCtx, dirPath string) error {
	ctx.Total(1, false)
	total := uint64(0)
	dirIndexPath := dirPath
	if !utils.IsRootPath(dirIndexPath) {
		dirIndexPath += "/"
	}
	ps := bleve.NewPrefixQuery(dirIndexPath)
	for {
		if e := ctx.Err(); e != nil {
			return e
		}
		req := bleve.NewSearchRequestOptions(ps, 1000, 0, false)
		r, e := s.index.Search(req)
		if e != nil {
			return e
		}
		if total == 0 {
			total = r.Total
			ctx.Total(int64(total), false)
		}
		if total == 0 {
			break
		}
		if len(r.Hits) == 0 {
			break
		}
		for _, hit := range r.Hits {
			if e := ctx.Err(); e != nil {
				return e
			}
			e := s.index.Delete(hit.ID)
			if e != nil {
				return e
			}
			ctx.Progress(1, false)
		}
	}
	e := s.index.Delete(dirPath)
	if e != nil {
		return e
	}
	ctx.Progress(1, false)
	return nil
}

func (s *BleveSearcher) Stats() (types.SM, error) {
	docs, e := s.index.DocCount()
	if e != nil {
		return nil, e
	}
	stats := s.index.StatsMap()
	return types.SM{
		"Total":      strconv.FormatUint(docs, 10),
		"Searches":   strconv.FormatUint(stats["searches"].(uint64), 10),
		"SearchTime": strconv.FormatUint(stats["search_time"].(uint64), 10),
	}, nil
}

func (s *BleveSearcher) Examples() []string {
	return []string{"*.txt", "name", "type:dir", "size:>10m", "modTime:>\"1998-04-23\""}
}

func (s *BleveSearcher) Dispose() error {
	return s.index.Close()
}

func mapSearchResultItem(hits search.DocumentMatchCollection) []types.EntrySearchResultItem {
	items := make([]types.EntrySearchResultItem, 0, len(hits))
	for _, hit := range hits {
		modTime, _ := time.Parse(time.RFC3339, hit.Fields["modTime"].(string))
		esi := types.EntrySearchItem{
			Path:    hit.Fields["path"].([]interface{})[0].(string),
			Name:    hit.Fields["name"].([]interface{})[0].(string),
			Ext:     hit.Fields["ext"].(string),
			Type:    types.EntryType(hit.Fields["type"].(string)),
			Size:    int64(hit.Fields["size"].(float64)),
			ModTime: modTime,
		}
		highlights := make(map[string][]string, len(hit.Locations))
		for k, v := range hit.Locations {
			segments := make([]string, 0, len(v))
			for seg := range v {
				segments = append(segments, seg)
			}
			highlights[k] = segments
		}
		item := types.EntrySearchResultItem{
			Entry:      esi,
			Highlights: highlights,
		}

		items = append(items, item)
	}
	return items
}
