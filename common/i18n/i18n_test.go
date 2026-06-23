package i18n

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type testMessageSource map[string]string

func (m testMessageSource) Translate(_ string, key string, args ...string) string {
	return Translate(m[key], args...)
}

func TestTranslate(t *testing.T) {
	tr := Translate("Hello{{2}}{{4}}}{{}}}{3}}{1}{{{ 1 }}}, {1c1}. How are you{2}{{2}}.", "JJ", "?")
	if tr != "Hello?{{4}}}{{}}}{3}}{1}{JJ}, {1c1}. How are you{2}?." {
		t.Error(tr)
	}
}

func TestTranslateTSupportsNestedMessage(t *testing.T) {
	source := testMessageSource{
		"outer": "outer: {{1}}",
		"inner": "inner: {{1}}",
	}
	message := T("outer", T("inner", "detail"))
	if got := TranslateT("en", source, message); got != "outer: inner: detail" {
		t.Fatalf("nested translation = %q", got)
	}
}

func TestTranslateTDepthLimit(t *testing.T) {
	source := testMessageSource{
		"outer": "outer: {{1}}",
		"inner": "inner: {{1}}",
		"deep":  "deep: {{1}}",
	}
	deep := T("deep", "detail")
	message := T("outer", T("inner", deep))
	if got := TranslateT("en", source, message); got != "outer: inner: "+deep {
		t.Fatalf("depth-limited translation = %q", got)
	}
}

func TestT(t *testing.T) {
	want := []string{"Hello, {{1}}. \"How are you{{2}}\"", "\"J\"J", "?", "你好"}
	token := T(want[0], want[1:]...)
	if !strings.HasPrefix(token, tokenPrefix) {
		t.Fatalf("token %q does not have prefix %q", token, tokenPrefix)
	}
	got, e := UnmarshalT(token)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded token = %#v, want %#v", got, want)
	}
}

func TestUnmarshalTRejectsUnversionedToken(t *testing.T) {
	for _, token := range []string{
		`"key"`,
		`"key","arg"`,
		"plain string",
		"",
	} {
		if _, e := UnmarshalT(token); e == nil {
			t.Errorf("expected unversioned token %q to be rejected", token)
		}
	}
}

func TestUnmarshalTRejectsInvalidVersionedToken(t *testing.T) {
	for _, token := range []string{
		tokenPrefix + `[]`,
		tokenPrefix + `["key"],`,
		tokenPrefix + `[1]`,
	} {
		if _, e := UnmarshalT(token); e == nil {
			t.Errorf("expected token %q to be rejected", token)
		}
	}
}

func TestTranslateVPreservesTypesAndTranslatesPointerFields(t *testing.T) {
	message := T("message")
	type payload struct {
		Message *string   `i18n:""`
		Items   [1]string `i18n:""`
	}
	input := &payload{Message: &message, Items: [1]string{message}}
	result, ok := TranslateV("en", testMessageSource{"message": "translated"}, input).(*payload)
	if !ok {
		t.Fatalf("result type is %T, want *payload", TranslateV("en", testMessageSource{}, input))
	}
	if *result.Message != "translated" || result.Items[0] != "translated" {
		t.Fatalf("unexpected translation: %#v", result)
	}

	mapResult := TranslateV("en", testMessageSource{"message": "translated"}, map[string]any{"value": message}).(map[string]any)
	if _, ok := mapResult["value"].(string); !ok {
		t.Fatalf("map value type is %T, want string", mapResult["value"])
	}
}

func TestTranslateVCollectionAndStructFieldBehavior(t *testing.T) {
	message := T("message")
	type payload struct {
		Plain  string
		Tagged string `i18n:""`
		Items  []string
	}

	result := TranslateV("en", testMessageSource{"message": "translated"}, payload{
		Plain:  message,
		Tagged: message,
		Items:  []string{message},
	}).(payload)
	if result.Plain != message {
		t.Fatalf("untagged string field was translated: %q", result.Plain)
	}
	if result.Tagged != "translated" {
		t.Fatalf("tagged string field was not translated: %q", result.Tagged)
	}
	if result.Items[0] != "translated" {
		t.Fatalf("collection string was not translated: %q", result.Items[0])
	}

	mapResult := TranslateV("en", testMessageSource{"key": "translated-key", "message": "translated-value"}, map[string]string{
		T("key"): message,
	}).(map[string]string)
	if mapResult["translated-key"] != "translated-value" {
		t.Fatalf("map keys and values were not translated: %#v", mapResult)
	}
}

func TestFileMessageSourceFallbackAndFileFiltering(t *testing.T) {
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "en.yaml"), []byte("hello: Hello\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "fr.yml"), []byte("hello: Bonjour\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "de"), []byte("not yaml"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(filepath.Join(dir, "zh-CN"), 0700); e != nil {
		t.Fatal(e)
	}

	source, e := NewFileMessageSource(os.DirFS(dir))
	if e != nil {
		t.Fatal(e)
	}
	if got := source.Translate("invalid language", "hello"); got != "Hello" {
		t.Fatalf("default translation = %q, want Hello", got)
	}
	if got := source.Translate("fr-FR", "hello"); got != "Bonjour" {
		t.Fatalf("matched translation = %q, want Bonjour", got)
	}
}
