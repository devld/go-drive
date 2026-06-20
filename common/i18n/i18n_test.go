package i18n

import (
	"fmt"
	"go-drive/common"
	"os"
	"path/filepath"
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

func TestT(t *testing.T) {
	fmt.Println(T("Hello, {{1}}. \"How are you{{2}}\"", "\"J\"J", "?"))
}

func TestUnmarshalT(t *testing.T) {
	// "Hello, {1}. ""How are you{2}""","""J""J","?"
	s := "\"Hello, {{1}}. \"\"How are you{{2}}\"\"\",\"\"\"J\"\"J\",\"?\""
	r, e := UnmarshalT(s)
	if e != nil {
		t.Error(e)
	}
	if len(r) != 3 {
		t.Errorf("unexpected result: length: %d, %v", len(r), r)
	}
	for _, ss := range r {
		fmt.Print("==> ")
		fmt.Println(ss)
	}

	s += ","
	r, e = UnmarshalT(s)
	if e != nil {
		t.Error(e)
	}
	if len(r) != 3 {
		t.Errorf("unexpected result: length: %d, %v", len(r), r)
	}
	for _, ss := range r {
		fmt.Print("==> ")
		fmt.Println(ss)
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

	source, e := NewFileMessageSource(common.Config{LangDir: dir, DefaultLang: "en-US"})
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
