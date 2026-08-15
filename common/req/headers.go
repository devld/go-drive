package req

import (
	"encoding/json"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/http/httpguts"
)

const RequestHeadersField = "request_headers"

const (
	requestHeaderItemKey    = "header"
	requestHeaderNameField  = "name"
	requestHeaderValueField = "value"
)

type requestHeaderConfig struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

var unsupportedRequestHeaders = map[string]struct{}{
	"connection":          {},
	"content-length":      {},
	"host":                {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"proxy-connection":    {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

// RequestHeadersForm returns the form schema consumed by ParseRequestHeaders.
func RequestHeadersForm(label, description string) types.FormItem {
	return types.FormItem{
		Field: RequestHeadersField, Label: label, Type: "form",
		Description: description,
		Forms: &types.FormItemForms{
			AddText: i18n.T("request_headers.add"),
			Forms: []types.FormItemForm{{
				Key: requestHeaderItemKey,
				Form: []types.FormItem{
					{Field: requestHeaderNameField, Label: i18n.T("request_headers.name"), Type: "text", Required: true},
					{Field: requestHeaderValueField, Label: i18n.T("request_headers.value"), Type: "text"},
				},
			}},
		},
	}
}

// ParseRequestHeaders parses the repeatable request-header form value.
func ParseRequestHeaders(value string) (http.Header, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	items := []requestHeaderConfig{}
	if e := json.Unmarshal([]byte(value), &items); e != nil {
		return nil, err.NewBadRequestError(i18n.T("request_headers.invalid_format", e.Error()))
	}

	headers := make(http.Header, len(items))
	for i, item := range items {
		name := strings.TrimSpace(item.Name)
		if !httpguts.ValidHeaderFieldName(name) {
			return nil, err.NewBadRequestError(i18n.T("request_headers.invalid_name", strconv.Itoa(i+1), name))
		}
		if _, unsupported := unsupportedRequestHeaders[strings.ToLower(name)]; unsupported {
			return nil, err.NewBadRequestError(i18n.T("request_headers.unsupported", name))
		}
		if !httpguts.ValidHeaderFieldValue(item.Value) {
			return nil, err.NewBadRequestError(i18n.T("request_headers.invalid_value", name))
		}

		name = http.CanonicalHeaderKey(name)
		if _, exists := headers[name]; exists {
			return nil, err.NewBadRequestError(i18n.T("request_headers.duplicate", name))
		}
		headers.Set(name, item.Value)
	}
	return headers, nil
}
