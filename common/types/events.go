package types

import "net/http"

type DriveListenerContext struct {
	Request *http.Request
	Session Session
}
