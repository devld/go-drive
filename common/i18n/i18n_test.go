package i18n

import (
	"fmt"
	"testing"
)

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
