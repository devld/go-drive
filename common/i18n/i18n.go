package i18n

import (
	"errors"
	"go-drive/common/utils"
	"reflect"
	"strconv"
	"strings"
)

type MessageSource interface {
	Translate(lang, key string, args ...string) string
}

func TranslateV(lang string, ms MessageSource, v interface{}) interface{} {
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
	arr, e := UnmarshalT(t)
	if e != nil || len(arr) == 0 {
		return t
	}
	// translate args
	for i := 1; i < len(arr); i++ {
		arr[i] = TranslateT(lang, ms, arr[i])
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

// T encodes pattern and args into a single string
func T(pattern string, args ...string) string {
	sb := strings.Builder{}
	sb.WriteRune('"')
	sb.WriteString(strings.ReplaceAll(pattern, "\"", "\"\""))
	sb.WriteRune('"')
	if len(args) > 0 {
		sb.WriteRune(',')
		for i, s := range args {
			sb.WriteRune('"')
			sb.WriteString(strings.ReplaceAll(s, "\"", "\"\""))
			sb.WriteRune('"')
			if i < len(args)-1 {
				sb.WriteRune(',')
			}
		}
	}
	return sb.String()
}

const (
	stateIdle      = 0
	stateString    = 1
	stateStringEnd = 2
)

func UnmarshalT(s string) ([]string, error) {
	state := stateIdle
	sb := strings.Builder{}
	result := make([]string, 0)
	for _, ch := range s {
		switch ch {
		case '"':
			switch state {
			case stateIdle:
				state = stateString
			case stateString:
				state = stateStringEnd
			case stateStringEnd:
				state = stateString
				sb.WriteRune('"')
			default:
				return nil, errors.New("unexpected char '\"'")
			}
		case ',':
			switch state {
			case stateString:
				sb.WriteRune(',')
			case stateStringEnd:
				state = stateIdle
				result = append(result, sb.String())
				sb.Reset()
			default:
				return nil, errors.New("unexpected char ','")
			}
		default:
			switch state {
			case stateString:
				sb.WriteRune(ch)
			default:
				return nil, errors.New("unexpected char '" + string(ch) + "'")
			}
		}
	}
	if state != stateIdle && state != stateStringEnd {
		return nil, errors.New("unexpected end of string")
	}
	if state == stateStringEnd {
		result = append(result, sb.String())
	}
	return result, nil
}
