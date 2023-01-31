package search

import (
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	sqliteDBFilename = "index.db"
)

var (
	multipleAsterisksPattern = regexp.MustCompile(`\*{2,}`)
	spacesPattern            = regexp.MustCompile(`\s+`)
)

func init() {
	RegisterSearcher("sqlite", NewSQLiteSearcher)
}

type SQLiteSearcher struct {
	db *gorm.DB
}

func NewSQLiteSearcher(config common.Config, searcherConfig types.SM) (Searcher, error) {
	dbName := searcherConfig["name"]
	if dbName == "" {
		dbName = "files.sqlite.index"
	}
	dbDir, e := config.GetDir(dbName, true)
	if e != nil {
		return nil, e
	}

	dbConfig := &gorm.Config{}
	if utils.IsDebugOn {
		dbConfig.Logger = logger.New(
			log.New(os.Stdout, "\n[SQLiteSearcher] ", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	}
	dbFile := filepath.Join(dbDir, sqliteDBFilename)
	db, e := gorm.Open(sqlite.Open(dbFile), dbConfig)
	if e != nil {
		return nil, e
	}
	if e := db.AutoMigrate(&entry{}); e != nil {
		return nil, e
	}

	searcher := &SQLiteSearcher{db: db}
	return searcher, nil
}

func (s *SQLiteSearcher) Index(ctx types.TaskCtx, entries []types.EntrySearchItem) error {
	data := utils.ArrayMap(entries, func(t *types.EntrySearchItem) entry {
		return entry{
			Path: t.Path, Name: t.Name, Ext: &t.Ext,
			Type: t.Type, Size: t.Size, ModTime: t.ModTime,
		}
	})
	for i := 0; i < len(data); i += 500 {
		end := int(math.Min(float64(i+500), float64(len(data))))
		s.db.Where("path in ?",
			utils.ArrayMap(data[i:end], func(t *entry) string { return t.Path })).Delete(&entry{})
	}
	return s.db.CreateInBatches(data, 150).Error
}

func (s *SQLiteSearcher) Search(path string, query string, from int, size int) ([]types.EntrySearchResultItem, error) {
	tx := s.db.Model(&entry{}).Offset(from).Limit(size)
	if path != "" {
		tx = tx.Where("path LIKE (? || '%')", path+"/")
	}

	tx = s.buildQuery(tx, query)

	var entries []entry
	if e := tx.Find(&entries).Error; e != nil {
		return nil, e
	}
	return utils.ArrayMap(entries, func(t *entry) types.EntrySearchResultItem {
		return types.EntrySearchResultItem{
			Entry: types.EntrySearchItem{
				Path: t.Path, Name: t.Name, Ext: *t.Ext,
				Type: t.Type, Size: t.Size, ModTime: t.ModTime,
			},
		}
	}), nil
}

func (s *SQLiteSearcher) Delete(ctx types.TaskCtx, dirPath string) error {
	ctx.Total(1, false)
	for {
		var entries []entry
		if e := s.db.Where("path LIKE (? || '%')", dirPath+"/").Limit(999).Find(&entries).Error; e != nil {
			return e
		}
		if len(entries) == 0 {
			break
		}
		ctx.Total(int64(len(entries)), false)
		if e := s.db.Where(
			"path IN ("+strings.TrimSuffix(strings.Repeat("?, ", len(entries)), ", ")+")",
			utils.ArrayMap(entries, func(t *entry) interface{} { return t.Path })...,
		).Delete(&entry{}).Error; e != nil {
			return e
		}
		ctx.Progress(int64(len(entries)), false)
	}
	if e := s.db.Delete(&entry{}, "path = ?", dirPath).Error; e != nil {
		return e
	}
	ctx.Progress(1, false)
	return nil
}

func (s *SQLiteSearcher) Stats() (types.SM, error) {
	var count int64
	if e := s.db.Model(&entry{}).Count(&count).Error; e != nil {
		return nil, e
	}
	return types.SM{
		"Total": strconv.FormatInt(count, 10),
	}, nil
}

var sizeQueryPattern = regexp.MustCompile(`^(=|[><]=?)([0-9]+[bkmgtBKMGT]?)$`)

func (s *SQLiteSearcher) buildQuery(tx *gorm.DB, query string) *gorm.DB {
	query = strings.TrimSpace(query)
	// replace multiple asterisks
	query = multipleAsterisksPattern.ReplaceAllString(query, "*")

	// split query by space
	queries := spacesPattern.Split(query, -1)

	where := make([]string, 0, len(queries))
	values := make([]interface{}, 0, len(queries))

	for _, query := range queries {
		var where_ string
		var values_ []interface{}
		if strings.HasPrefix(query, "path:") {
			where_, values_ = buildWildcardQuery(query[5:], "path")
		} else if strings.HasPrefix(query, "in:") {
			duration := types.SV(query[3:]).Duration(-1)
			if duration <= 0 {
				continue
			}
			modTime := time.Now().Add(-duration)
			where_ = "mod_time > ?"
			values_ = []interface{}{modTime}
		} else if m := sizeQueryPattern.FindStringSubmatch(query); m != nil {
			size := types.SV(m[2]).DataSize(-1)
			if size < 0 {
				continue
			}
			where_ = "size >= 0 AND size " + m[1] + " ?"
			values_ = []interface{}{size}
		} else if strings.HasPrefix(query, "type:") {
			where_ = "type = ?"
			values_ = []interface{}{query[5:]}
		} else {
			where_, values_ = buildWildcardQuery(query, "name")
		}
		where = append(where, where_)
		values = append(values, values_...)
	}

	if len(where) > 0 {
		tx = tx.Where("("+strings.Join(where, " AND ")+")", values...)
	}
	return tx
}

func (*SQLiteSearcher) Examples() []string {
	return []string{"hello.txt", "path:a*dir", "*.mp3", "*.mp3 in:1h", ">10m", "type:dir"}
}

func (s *SQLiteSearcher) Dispose() error {
	db, e := s.db.DB()
	if e != nil {
		return e
	}
	return db.Close()
}

func buildWildcardQuery(query, column string) (string, []interface{}) {
	query = strings.Trim(query, "*")
	values := utils.ArrayMap(strings.Split(query, "*"), func(t *string) interface{} { return t })
	where := column + " LIKE ('%' || " +
		strings.TrimSuffix(strings.Repeat("? || '%' || ", len(values)), " || '%' || ") + " || '%' )"
	return where, values
}

type entry struct {
	Path    string          `gorm:"column:path;primaryKey;not null;type:string;size:4096"`
	Name    string          `gorm:"column:name;not null;type:string;size:255"`
	Ext     *string         `gorm:"column:ext;not null;type:string;size:255"`
	Type    types.EntryType `gorm:"column:type;not null;type:string;size:16"`
	Size    int64           `gorm:"column:size;not null;type:int"`
	ModTime time.Time       `gorm:"column:mod_time;not null;type:time;index"`
}

func (entry) TableName() string {
	return "entries"
}

var _ Searcher = (*SQLiteSearcher)(nil)
