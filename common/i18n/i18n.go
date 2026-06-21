package i18n

import (
	"encoding/json"
	"errors"
	"go-drive/common/utils"
	"reflect"
	"strconv"
	"strings"
)

type MessageSource interface {
	Translate(lang, key string, args ...string) string
}

func TranslateV(lang string, ms MessageSource, v any) any {
	return utils.VisitValueTree(v, func(v reflect.Value, sf *reflect.StructField) {
		if v.Kind() != reflect.String {
			return
		}
		if sf != nil {
			if _, ok := sf.Tag.Lookup("i18n"); !ok {
				return
			}
		}
		v.SetString(TranslateT(lang, ms, v.String()))
	})
}

func TranslateT(lang string, ms MessageSource, t string) string {
	return translateT(lang, ms, t, 0)
}

const maxTranslationDepth = 2

func translateT(lang string, ms MessageSource, t string, depth int) string {
	if depth >= maxTranslationDepth {
		return t
	}
	arr, e := UnmarshalT(t)
	if e != nil || len(arr) == 0 {
		return t
	}
	// translate args
	for i := 1; i < len(arr); i++ {
		arr[i] = translateT(lang, ms, arr[i], depth+1)
	}
	return ms.Translate(lang, arr[0], arr[1:]...)
}

const (
	stateText = 0
	stateVar  = 1
)

func Translate(pattern string, args ...string) string {
	sb := strings.Builder{}
	currentVar := strings.Builder{}
	state := stateText
	for _, ch := range pattern {
		switch ch {
		case '{':
			switch state {
			case stateText:
				currentVar.WriteRune('{')
				state = stateVar
			case stateVar:
				currentVar.WriteRune('{')
				v := currentVar.String()
				if len(v) > 2 {
					sb.WriteString(v[0 : len(v)-2])
					currentVar.Reset()
					currentVar.WriteString(v[len(v)-2:])
				}
			}
		case '}':
			switch state {
			case stateText:
				sb.WriteRune(ch)
			case stateVar:
				currentVar.WriteRune('}')
				v := currentVar.String()
				if !strings.HasPrefix(v, "{{") {
					state = stateText
					sb.WriteString(v)
					currentVar.Reset()
					continue
				}
				if strings.HasSuffix(v, "}}") {
					sb.WriteString(expandVar(currentVar.String(), args))
					currentVar.Reset()
					state = stateText
				}
			}
		default:
			switch state {
			case stateText:
				sb.WriteRune(ch)
			case stateVar:
				currentVar.WriteRune(ch)
			}
		}
	}
	if currentVar.Len() > 0 {
		sb.WriteString(currentVar.String())
	}
	return sb.String()
}

// expandVar expands pattern like '{{ 1 }}' to vars at index
func expandVar(s string, vars []string) string {
	if !strings.HasPrefix(s, "{{") || !strings.HasSuffix(s, "}}") {
		return s
	}
	indexStr := strings.TrimSpace(s[2 : len(s)-2])
	index, e := strconv.Atoi(indexStr)
	if e != nil {
		return s
	}
	index-- // starts from 1
	if index < 0 || index >= len(vars) {
		return s
	}
	return vars[index]
}

const tokenPrefix = "@go-drive/i18n:v1:"

// T encodes pattern and args into a versioned translation token.
func T(pattern string, args ...string) string {
	items := make([]string, 1, len(args)+1)
	items[0] = pattern
	items = append(items, args...)
	data, _ := json.Marshal(items) // []string is always JSON-marshalable.
	return tokenPrefix + string(data)
}

func TPrefix(prefix string) func(string, ...string) string {
	return func(pattern string, args ...string) string {
		return T(prefix+pattern, args...)
	}
}

func UnmarshalT(s string) ([]string, error) {
	if !strings.HasPrefix(s, tokenPrefix) {
		return nil, errors.New("invalid translation token prefix")
	}
	var result []string
	if e := json.Unmarshal([]byte(strings.TrimPrefix(s, tokenPrefix)), &result); e != nil {
		return nil, e
	}
	if len(result) == 0 {
		return nil, errors.New("translation token contains no message key")
	}
	return result, nil
}
