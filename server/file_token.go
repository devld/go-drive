package server

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go-drive/common"
	"go-drive/common/types"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const sessionPrefix = "s_"

type FileToken struct {
	root        string
	validity    time.Duration
	autoRefresh bool
	stopCleaner func()
}

func init() {
	common.R().Register("tokenStore", func(c *common.ComponentRegistry) interface{} {
		dir, e := c.Get("config").(common.Config).GetDir("sessions", true)
		common.PanicIfError(e)
		ts, e := NewFileTokenStore(dir, 2*time.Hour, true, 6*time.Hour)
		common.PanicIfError(e)
		return ts
	}, 0)
}

// NewFileTokenStore creates a FileToken
func NewFileTokenStore(root string, validity time.Duration, autoRefresh bool,
	cleanupDuration time.Duration) (*FileToken, error) {
	if isDir, _ := common.IsDir(root); !isDir {
		return nil, errors.New("root not exists or is not a directory")
	}
	if cleanupDuration <= 0 {
		panic("invalid cleanupDuration")
	}
	ft := &FileToken{
		root:        root,
		autoRefresh: autoRefresh,
		validity:    validity,
	}
	ft.stopCleaner = common.TimeTick(ft.clean, cleanupDuration)
	return ft, nil
}

func (f *FileToken) Create(value types.Session) (types.Token, error) {
	token := uuid.New().String()
	return f.writeFile(token, &value, os.O_CREATE|os.O_WRONLY)
}

func (f *FileToken) Update(token string, value types.Session) (types.Token, error) {
	if _, e := f.readFile(token, false); e != nil {
		return types.Token{}, e
	}
	return f.writeFile(token, &value, os.O_TRUNC|os.O_WRONLY)
}

func (f *FileToken) Validate(token string) (types.Token, error) {
	t, e := f.readFile(token, true)
	if e != nil {
		return types.Token{}, e
	}
	if f.autoRefresh {
		_ = os.Chtimes(f.getSessionFile(token), time.Now(), time.Now())
	}
	return *t, nil
}

func (f *FileToken) Revoke(token string) error {
	_ = os.Remove(f.getSessionFile(token))
	return nil
}

func (f *FileToken) getSessionFile(token string) string {
	return filepath.Join(f.root, sessionPrefix+token)
}

func (f *FileToken) readFile(token string, read bool) (*types.Token, error) {
	filePath := f.getSessionFile(token)
	stat, e := os.Stat(filePath)
	if os.IsNotExist(e) || f.isExpired(stat.ModTime()) {
		return nil, common.NewUnauthorizedError("invalid token")
	}
	if !read {
		return nil, nil
	}
	s := types.Session{}
	file, e := os.Open(filePath)
	if e != nil {
		return nil, e
	}
	defer func() { _ = file.Close() }()
	e = gob.NewDecoder(file).Decode(&s)
	if e != nil {
		return nil, e
	}
	return &types.Token{
		Token:     token,
		Value:     s,
		ExpiredAt: stat.ModTime().Add(f.validity).Unix(),
	}, nil
}

func (f *FileToken) writeFile(token string, value *types.Session, flag int) (types.Token, error) {
	file, e := os.OpenFile(f.getSessionFile(token), flag, 0644)
	if e != nil {
		return types.Token{}, e
	}
	defer func() { _ = file.Close() }()
	e = gob.NewEncoder(file).Encode(value)
	if e != nil {
		return types.Token{}, e
	}
	return types.Token{
		Token:     token,
		Value:     *value,
		ExpiredAt: time.Now().Add(f.validity).Unix(),
	}, nil
}

func (f *FileToken) isExpired(modTime time.Time) bool {
	return modTime.Before(time.Now().Add(-f.validity))
}

func (f *FileToken) Dispose() error {
	f.stopCleaner()
	return nil
}

func (f *FileToken) forEachSession(fn func(string, os.FileInfo)) error {
	return filepath.Walk(f.root, func(path string, info os.FileInfo, e error) error {
		if e != nil || info.IsDir() || !strings.HasPrefix(filepath.Base(path), sessionPrefix) {
			return nil
		}
		fn(path, info)
		return nil
	})
}

func (f *FileToken) clean() {
	n := 0
	notBefore := time.Now().Add(-f.validity)
	e := f.forEachSession(func(path string, info os.FileInfo) {
		if info.ModTime().Before(notBefore) {
			if e := os.Remove(path); e != nil {
				log.Println("failed to delete file", e)
			}
			n++
		}
	})
	if n > 0 {
		log.Println(fmt.Sprintf("%d expired sessions cleaned", n))
	}
	if e != nil {
		log.Println("error when cleaning expired sessions", e)
	}
}

func (f *FileToken) Status() (string, types.SM, error) {
	total := 0
	active := 0
	if e := f.forEachSession(func(path string, info os.FileInfo) {
		total++
		if !f.isExpired(info.ModTime()) {
			active++
		}
	}); e != nil {
		return "", nil, e
	}

	return "Session", types.SM{
		"Total":  fmt.Sprintf("%d", total),
		"Active": fmt.Sprintf("%d", active),
	}, nil
}
