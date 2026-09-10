package server

import (
	err "go-drive/common/errors"
	"go-drive/common/utils"

	"github.com/gin-gonic/gin"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func newPageResult[T any](items []T, total int64, page, pageSize int) PageResult[T] {
	return PageResult[T]{Items: items, Total: total, Page: page, PageSize: pageSize}
}

func getPagination(c *gin.Context) (page, pageSize int, e error) {
	page = utils.ToInt(c.Query("page"), 1)
	pageSize = utils.ToInt(c.Query("pageSize"), defaultPageSize)
	if page < 1 || pageSize < 1 || pageSize > maxPageSize {
		return 0, 0, err.NewBadRequestError("invalid pagination")
	}
	return page, pageSize, nil
}
