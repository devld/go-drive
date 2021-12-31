package utils

import (
	"net/http"
	"time"
)

const (
	SignatureQueryKey = "_k"
)

func GetSignPayload(req *http.Request, path string) string {
	return req.Host + "." + path
}

func SignPathRequest(signer *Signer, req *http.Request, path string, notAfter time.Time) string {
	return signer.Sign(GetSignPayload(req, path), notAfter)
}

func CheckSignature(signer *Signer, req *http.Request, path string) bool {
	return signer.Validate(GetSignPayload(req, path), req.URL.Query().Get(SignatureQueryKey))
}
