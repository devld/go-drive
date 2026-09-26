package mega

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-drive/common/req"
	"go-drive/common/types"
)

func TestAPIRequestsUseDefaultUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[-9]`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("X_MEGA_API_URL", server.URL)
	t.Setenv("X_MEGA_USER_AGENT", "go-mega")

	api := prepareAPI(types.SM{})
	t.Cleanup(func() { api.Close() })
	_ = api.Login("user@example.com", "secret")
	if got != req.DefaultUserAgent {
		t.Fatalf("User-Agent = %q", got)
	}
}
