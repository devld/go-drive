package common

import (
	"math/rand"
	"net/http"
	"os"
	fsPath "path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func IsDebugOn() bool {
	_, exists := os.LookupEnv("DEBUG")
	return exists
}

func FileExists(path string) (bool, error) {
	_, e := os.Stat(path)
	if os.IsNotExist(e) {
		return false, nil
	}
	return e == nil, e
}

func IsDir(path string) (bool, error) {
	stat, e := os.Stat(path)
	if e != nil {
		return false, e
	}
	return stat.IsDir(), nil
}

func IsRootPath(path string) bool {
	return path == ""
}

func CleanPath(path string) string {
	path = fsPath.Clean(path)
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	for strings.HasPrefix(path, "../") {
		path = path[3:]
	}
	if path == "." {
		path = ""
	}
	return path
}

func PathBase(path string) string {
	base := fsPath.Base(path)
	if base == "/" || base == "." {
		base = ""
	}
	return base
}

func PathParent(path string) string {
	path = CleanPath(path)
	parent := fsPath.Dir(path)
	if parent == "/" || parent == "." {
		parent = ""
	}
	return parent
}

func PathParentTree(path string) []string {
	path = CleanPath(path)
	r := make([]string, 0, PathDepth(path))
	for path != "" {
		r = append(r, path)
		path = PathParent(path)
	}
	r = append(r, "")
	return r
}

var slashPattern = regexp.MustCompile("/")

func PathDepth(path string) int {
	path = CleanPath(path)
	if path == "" {
		return 0
	}
	return len(slashPattern.FindAll([]byte(path), -1)) + 1
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func PanicIfError(e error) {
	if e != nil {
		panic(e)
	}
}

func RequireNotNil(v interface{}, msg string) {
	if v == nil {
		panic(msg)
	}
}

func RegSplit(text string, reg *regexp.Regexp) []string {
	indexes := reg.FindAllStringIndex(text, -1)
	lastStart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[lastStart:element[0]]
		lastStart = element[1]
	}
	result[len(indexes)] = text[lastStart:]
	return result
}

func GetRealIP(r *http.Request) string {
	clientIP := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return clientIP
	}
	ips := RegSplit(forwarded, regexp.MustCompile(",\\s*"))
	return ips[0]
}

func Millisecond(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func Time(millisecond int64) time.Time {
	return time.Unix(0, millisecond*int64(time.Millisecond))
}

func ToInt64(s string, def int64) int64 {
	v, e := strconv.ParseInt(s, 10, 64)
	if e != nil {
		return def
	}
	return v
}

func ToInt(s string, def int) int {
	v, e := strconv.Atoi(s)
	if e != nil {
		return def
	}
	return v
}

func CopyMap(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	if m != nil {
		for k, v := range m {
			newMap[k] = v
		}
	}
	return newMap
}
