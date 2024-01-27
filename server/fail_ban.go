package server

import (
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func NewFailBanGroup(clearInterval time.Duration) *FailBanGroup {
	f := &FailBanGroup{cache: make(map[string]*utils.KVCache[failBanRecord])}
	f.clearInterval = clearInterval
	return f
}

type FailBanGroup struct {
	cache         map[string]*utils.KVCache[failBanRecord]
	clearInterval time.Duration
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
	m := utils.NewKVCache[failBanRecord](f.clearInterval)
	f.cache[path] = m

	return func(c *gin.Context) {
		key := keyFn(c)

		record, exists := m.Get(key)
		if exists && record.n >= maxFailure {
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
			if !ok {
				record = failBanRecord{}
			}
			record.n++
			m.Set(key, record, duration)
		}
	}
}

func (f *FailBanGroup) Dispose() error {
	for _, m := range f.cache {
		m.Dispose()
	}
	return nil
}

type failBanRecord struct {
	n uint32
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
