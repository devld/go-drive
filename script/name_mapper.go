package script

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type goFieldNameMapper struct{}

func (goFieldNameMapper) FieldName(_ reflect.Type, field reflect.StructField) string {
	return lowerCamelGoName(field.Name)
}

func (goFieldNameMapper) MethodName(t reflect.Type, method reflect.Method) string {
	if isJSClassHandleType(t) {
		return ""
	}
	switch method.Name {
	case "ConsoleString", "JsonEncodable", "MarshalJSON":
		return ""
	default:
		return lowerCamelGoName(method.Name)
	}
}

func lowerCamelGoName(name string) string {
	name = strings.ReplaceAll(name, "URL", "Url")
	name = strings.ReplaceAll(name, "JSON", "Json")
	name = strings.ReplaceAll(name, "OAuth", "Oauth")
	first, size := utf8.DecodeRuneInString(name)
	if size == 0 {
		return name
	}
	return string(unicode.ToLower(first)) + name[size:]
}

// jsObjectFieldNames is camelCase plus a differing json tag, if any.
func jsObjectFieldNames(field reflect.StructField) []string {
	camel := lowerCamelGoName(field.Name)
	names := []string{camel}
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return names
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" || name == camel {
		return names
	}
	return []string{camel, name}
}
