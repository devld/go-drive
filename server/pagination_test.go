package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetPagination(t *testing.T) {
	tests := []struct {
		query              string
		wantPage, wantSize int
		wantError          bool
	}{
		{"", 1, defaultPageSize, false},
		{"?page=3&pageSize=50", 3, 50, false},
		{"?page=0", 0, 0, true},
		{"?pageSize=101", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/"+tt.query, nil)
			page, size, e := getPagination(c)
			if (e != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError = %v", e, tt.wantError)
			}
			if page != tt.wantPage || size != tt.wantSize {
				t.Fatalf("pagination = %d/%d, want %d/%d", page, size, tt.wantPage, tt.wantSize)
			}
		})
	}
}
