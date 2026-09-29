package server

import (
	"context"
	"encoding/json"
	"errors"
	"go-drive/common"
	apierr "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type responseTestMessages struct{}

func (responseTestMessages) Translate(lang, key string, args ...string) string {
	if key == "job_failed" && lang == "zh-CN" {
		return "任务失败"
	}
	if key == "move_across_not_supported" && lang == "zh-CN" {
		return "不支持跨 Drive 移动文件"
	}
	return key
}

func TestTaskResponseErrorTranslated(t *testing.T) {
	runner := task.NewTaskRunner(common.Config{MaxConcurrentTask: 1})
	t.Cleanup(func() { _ = runner.Dispose() })
	failed, e := runner.ExecuteAndWait(context.Background(), func(types.TaskCtx) (any, error) {
		return nil, errors.New(i18n.T("move_across_not_supported"))
	}, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if failed.Status != task.Error {
		t.Fatalf("task status = %q", failed.Status)
	}

	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodGet, "/tasks/test", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN,en-US;q=0.8")
	SetMessageSource(c, responseTestMessages{})
	writeJSON(c, http.StatusOK, failed)
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if e := json.Unmarshal(response.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	if body.Error.Message != "不支持跨 Drive 移动文件" {
		t.Fatalf("response error = %q", body.Error.Message)
	}
}

func TestJobExecutionErrorTranslatedForResponse(t *testing.T) {
	stored := types.JobExecution{ErrorMsg: i18n.T("job_failed")}
	view := jobExecutionView{JobExecution: stored}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/jobs/test", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	SetMessageSource(c, responseTestMessages{})
	localized := GetTranslator(c).TranslateV(view).(jobExecutionView)
	if localized.ErrorMsg != "任务失败" {
		t.Fatalf("response error = %q", localized.ErrorMsg)
	}
	if stored.ErrorMsg != i18n.T("job_failed") {
		t.Fatalf("stored error changed: %q", stored.ErrorMsg)
	}
}

func TestTranslatorCapturesRequestLanguage(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/jobs/test", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN,en-US;q=0.8")
	SetMessageSource(c, responseTestMessages{})
	translator := GetTranslator(c)
	c.Request.Header.Set("Accept-Language", "en-US")
	if got := translator.TranslateT(i18n.T("job_failed")); got != "任务失败" {
		t.Fatalf("translated text = %q", got)
	}
}

func TestFileBucketErrorUsesRequestTranslation(t *testing.T) {
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodGet, "/f/test/file", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN,en-US;q=0.8")
	SetMessageSource(c, responseTestMessages{})

	fr := &fileBucketRoute{}
	fr.abortWithError(c, apierr.NewNotAllowedMessageError(i18n.T("move_across_not_supported")))
	if response.Code != http.StatusForbidden || response.Body.String() != "不支持跨 Drive 移动文件" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}
