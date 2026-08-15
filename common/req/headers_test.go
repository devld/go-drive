package req

import (
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"testing"
)

func TestRequestHeadersFormSchema(t *testing.T) {
	form := RequestHeadersForm("Request Headers", "Request header description")
	if form.Field != RequestHeadersField || form.Type != "form" {
		t.Fatalf("unexpected top-level form: %#v", form)
	}
	if form.Label != "Request Headers" || form.Description != "Request header description" {
		t.Fatalf("unexpected top-level text: %#v", form)
	}
	if form.Forms == nil || len(form.Forms.Forms) != 1 {
		t.Fatalf("unexpected nested forms: %#v", form.Forms)
	}
	item := form.Forms.Forms[0]
	if item.Key != requestHeaderItemKey || item.Name != "" || len(item.Form) != 2 {
		t.Fatalf("unexpected header item form: %#v", item)
	}
	if item.Form[0].Field != requestHeaderNameField || !item.Form[0].Required {
		t.Fatalf("unexpected header name field: %#v", item.Form[0])
	}
	if item.Form[1].Field != requestHeaderValueField {
		t.Fatalf("unexpected header value field: %#v", item.Form[1])
	}
	translations := map[string]string{
		form.Forms.AddText: "request_headers.add",
		item.Form[0].Label: "request_headers.name",
		item.Form[1].Label: "request_headers.value",
	}
	for token, want := range translations {
		parts, e := i18n.UnmarshalT(token)
		if e != nil || len(parts) == 0 || parts[0] != want {
			t.Fatalf("translation token %q: parts=%#v error=%v, want key %q", token, parts, e, want)
		}
	}
}

func TestParseRequestHeaders(t *testing.T) {
	headers, e := ParseRequestHeaders(`[
		{"$key":"header","name":" User-Agent ","value":"rclone/v1.68.0"},
		{"$key":"header","name":"X-Data-Space","value":"capsule"}
	]`)
	if e != nil {
		t.Fatalf("ParseRequestHeaders: %v", e)
	}
	if got := headers.Get("User-Agent"); got != "rclone/v1.68.0" {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := headers.Get("X-Data-Space"); got != "capsule" {
		t.Fatalf("X-Data-Space = %q", got)
	}
}

func TestParseRequestHeadersRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		value string
		key   string
	}{
		{name: "invalid json", value: `{`, key: "request_headers.invalid_format"},
		{name: "empty name", value: `[{"name":"","value":"x"}]`, key: "request_headers.invalid_name"},
		{name: "invalid name", value: `[{"name":"Bad Header","value":"x"}]`, key: "request_headers.invalid_name"},
		{name: "invalid value", value: `[{"name":"X-Test","value":"a\nb"}]`, key: "request_headers.invalid_value"},
		{name: "unsupported", value: `[{"name":"Content-Length","value":"10"}]`, key: "request_headers.unsupported"},
		{name: "duplicate", value: `[{"name":"X-Test","value":"a"},{"name":"x-test","value":"b"}]`, key: "request_headers.duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, e := ParseRequestHeaders(tt.value)
			if e == nil {
				t.Fatal("expected an error")
			}
			if _, ok := e.(err.BadRequestError); !ok {
				t.Fatalf("error type = %T, want errors.BadRequestError", e)
			}
			parts, parseError := i18n.UnmarshalT(e.Error())
			if parseError != nil || len(parts) == 0 || parts[0] != tt.key {
				t.Fatalf("error token: parts=%#v error=%v, want key %q", parts, parseError, tt.key)
			}
		})
	}
}
