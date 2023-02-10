package server

import (
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	cmap "github.com/orcaman/concurrent-map/v2"
)

func NewFailBanGroup(cleanInterval time.Duration) *FailBanGroup {
	f := &FailBanGroup{cache: make(map[string]cmap.ConcurrentMap[string, failBanRecord])}
	f.timerStop = utils.TimeTick(f.evict, cleanInterval)
	return f
}

type FailBanGroup struct {
	cache     map[string]cmap.ConcurrentMap[string, failBanRecord]
	timerStop func()
}

func (f *FailBanGroup) LimiterByIP(path string, duration time.Duration, maxFailure uint32) gin.HandlerFunc {
	return f.Limiter(path, duration, maxFailure, func(ctx *gin.Context) string {
		ip := ctx.ClientIP()
		if utils.IsDebugOn {
			ctx.Header("X-ClientIP", ip)
		}
		return ip
	})
}

func (f *FailBanGroup) Limiter(path string, duration time.Duration, maxFailure uint32, keyFn func(ctx *gin.Context) string) gin.HandlerFunc {
	m := cmap.New[failBanRecord]()
	f.cache[path] = m

	return func(c *gin.Context) {
		key := keyFn(c)

		record, exists := m.Get(key)
		if exists && !record.isExpired() && record.n >= maxFailure {
			_ = c.Error(errFailBan)
			c.Abort()
			return
		}

		c.Next()

		status := c.Writer.Status()
		ok := len(c.Errors) == 0 && status >= 100 && status < 300

		if ok {
			m.Remove(key)
		} else {
			record, ok = m.Get(key)
			if !ok || record.isExpired() {
				record = failBanRecord{e: time.Now().Add(duration)}
			}
			record.n++
			m.Set(key, record)
		}
	}
}

func (f *FailBanGroup) evict() {
	for _, m := range f.cache {
		evictKeys := make([]string, 0)
		m.IterCb(func(key string, v failBanRecord) {
			if v.isExpired() {
				evictKeys = append(evictKeys, key)
			}
		})
		for _, k := range evictKeys {
			m.Remove(k)
		}
	}
}

func (f *FailBanGroup) Dispose() error {
	f.timerStop()
	return nil
}

type failBanRecord struct {
	n uint32
	e time.Time
}

func (f failBanRecord) isExpired() bool {
	return f.e.Before(time.Now())
}

var errFailBan err.Error = failBanError{}

type failBanError struct{}

func (failBanError) Code() int {
	return http.StatusTooManyRequests
}

func (f failBanError) Error() string {
	return i18n.T("error.fail_ban_message")
}

func (failBanError) Name() string {
	return "FAIL_BAN"
}
