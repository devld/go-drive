package script

import (
	"reflect"
	"testing"
)

type mapperFixture struct {
	ContentURL string `js:"ignored"`
	OAuth      string
}

func (mapperFixture) GetURL()       {}
func (mapperFixture) JSON()         {}
func (mapperFixture) UnixMilli()    {}
func (mapperFixture) HttpResponse() {}
func (mapperFixture) OAuthLoad()    {}

func TestJSObjectFieldNamesUsesJSONTag(t *testing.T) {
	typ := reflect.TypeOf(struct {
		OAuth string `json:"oauth"`
		Value string `json:"value"`
		Skip  string `json:"-"`
	}{})
	oauth, _ := typ.FieldByName("OAuth")
	if got := jsObjectFieldNames(oauth); len(got) != 1 || got[0] != "oauth" {
		t.Fatalf("OAuth names = %#v", got)
	}
	value, _ := typ.FieldByName("Value")
	if got := jsObjectFieldNames(value); len(got) != 1 || got[0] != "value" {
		t.Fatalf("Value names = %#v", got)
	}
	skip, _ := typ.FieldByName("Skip")
	if got := jsObjectFieldNames(skip); len(got) != 1 || got[0] != "skip" {
		t.Fatalf("Skip names = %#v", got)
	}
}

func TestGoFieldNameMapper(t *testing.T) {
	typ := reflect.TypeOf(mapperFixture{})
	mapper := goFieldNameMapper{}
	field, _ := typ.FieldByName("ContentURL")
	if got := mapper.FieldName(typ, field); got != "contentUrl" {
		t.Fatalf("ContentURL = %q", got)
	}
	oauth, _ := typ.FieldByName("OAuth")
	if got := mapper.FieldName(typ, oauth); got != "oauth" {
		t.Fatalf("OAuth = %q", got)
	}
	wants := map[string]string{
		"GetURL":       "getUrl",
		"JSON":         "json",
		"UnixMilli":    "unixMilli",
		"HttpResponse": "httpResponse",
		"OAuthLoad":    "oauthLoad",
	}
	for name, want := range wants {
		method, _ := typ.MethodByName(name)
		if got := mapper.MethodName(typ, method); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestHostHandleHidesGoMethods(t *testing.T) {
	mapper := goFieldNameMapper{}
	bytesType := reflect.TypeOf(jsObjBytes{})
	lenMethod, ok := bytesType.MethodByName("Len")
	if !ok {
		t.Fatal("Bytes.Len missing")
	}
	if got := mapper.MethodName(bytesType, lenMethod); got != "" {
		t.Fatalf("JSClass handle method should be hidden, got %q", got)
	}
	driveType := reflect.TypeOf(jsObjDrive{})
	get, ok := driveType.MethodByName("Get")
	if !ok {
		t.Fatal("Drive.Get missing")
	}
	if got := mapper.MethodName(driveType, get); got != "" {
		t.Fatalf("JSClass handle method should be hidden, got %q", got)
	}
}
