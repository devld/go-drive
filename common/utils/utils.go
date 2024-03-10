package utils

import (
	cryptoRand "crypto/rand"
	"encoding/base64"
	"fmt"
	"go-drive/common/types"
	"log"
	"math"
	"math/rand"
	"net/url"
	"os"
	path2 "path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var IsDebugOn bool

func init() {
	debugOn, _ := os.LookupEnv("GO_DRIVE_DEBUG")
	IsDebugOn = debugOn != ""
	if IsDebugOn {
		log.Println("debug mode is on")
	}
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

var unsafePathCharsRegexp = regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]+`)

func CleanPath(path string) string {
	path = unsafePathCharsRegexp.ReplaceAllLiteralString(strings.TrimSpace(path), "")
	path = path2.Clean(path)
	path = strings.TrimPrefix(path, "/")
	for strings.HasPrefix(path, "../") {
		path = path[3:]
	}
	if path == "." {
		path = ""
	}
	return path
}

func PathExt(name string) string {
	ext := path2.Ext(name)
	if ext != "" {
		ext = ext[1:]
	}
	return strings.ToLower(ext)
}

func PathName(name string) string {
	name = PathBase(name)
	i := strings.LastIndex(name, ".")
	if i == -1 {
		return name
	}
	return name[:i]
}

func PathBase(path string) string {
	base := path2.Base(path)
	if base == "/" || base == "." {
		base = ""
	}
	return base
}

func IsPathParent(path, parent string) bool {
	path = CleanPath(path)
	parent = CleanPath(parent)
	if IsRootPath(parent) {
		return !IsRootPath(path)
	}
	return strings.HasPrefix(path, parent+"/")
}

func PathParent(path string) string {
	path = CleanPath(path)
	parent := path2.Dir(path)
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

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func RandSecret(n int) []byte {
	b := make([]byte, n)
	_, e := cryptoRand.Read(b)
	if e != nil {
		panic(e)
	}
	return b
}

var lineEndRegexp = regexp.MustCompile("\r?\n")

func SplitLines(s string) []string {
	return lineEndRegexp.Split(s, -1)
}

func Millisecond(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func Time(millisecond int64) time.Time {
	return time.Unix(0, millisecond*int64(time.Millisecond))
}

func ToInt64(s string, def int64) int64 {
	return types.SV(s).Int64(def)
}

func ToUInt64(s string, def uint64) uint64 {
	return types.SV(s).Uint64(def)
}

func ToInt(s string, def int) int {
	return types.SV(s).Int(def)
}

func ToUInt(s string, def uint) uint {
	return types.SV(s).Uint(def)
}

func ToBool(s string) bool {
	return types.SV(s).Bool()
}

func BoolString(b bool) string {
	if b {
		return "1"
	}
	return ""
}

func FlattenStringMap(m map[string]interface{}, separator string) map[string]string {
	r := make(map[string]string)
	for k, v := range m {
		flattenStringMap(k, v, separator, r)
	}
	return r
}

func flattenStringMap(prefix string, val interface{}, separator string, result map[string]string) {
	m, isMap := val.(map[string]interface{})
	if isMap {
		for k, v := range m {
			flattenStringMap(prefix+separator+k, v, separator, result)
		}
		return
	}
	a, isArr := val.([]interface{})
	if isArr {
		result[prefix+separator+"size"] = strconv.Itoa(len(a))
		for i, v := range a {
			flattenStringMap(prefix+separator+strconv.Itoa(i), v, separator, result)
		}
		return
	}
	result[prefix] = fmt.Sprintf("%v", val)
}

func ArrayMap[TF any, TT any](a []TF, convert func(*TF) TT) []TT {
	r := make([]TT, len(a))
	for i := range a {
		r[i] = convert(&a[i])
	}
	return r
}

func ArrayMapWithError[TF any, TT any](a []TF, convert func(*TF) (TT, error)) ([]TT, error) {
	r := make([]TT, len(a))
	for i := range a {
		t, e := convert(&a[i])
		if e != nil {
			return nil, e
		}
		r[i] = t
	}
	return r, nil
}

func ArrayFind[T any](a []T, matches func(T, int) bool) (ret T, ok bool) {
	for i := range a {
		if matches(a[i], i) {
			ret = a[i]
			ok = true
			return
		}
	}
	return
}

func ArrayKeyBy[KT comparable, T any](a []T, keyFn func(T, int) KT) map[KT]T {
	m := make(map[KT]T, len(a))
	for i := range a {
		m[keyFn(a[i], i)] = a[i]
	}
	return m
}

func MapCopy[K comparable, V any](m map[K]V, dest map[K]V) map[K]V {
	if dest == nil {
		dest = make(map[K]V, len(m))
	}
	for k, v := range m {
		dest[k] = v
	}
	return dest
}

func MapKeys[K comparable, V any](m map[K]V) []K {
	values := make([]K, 0, len(m))
	for k := range m {
		values = append(values, k)
	}
	return values
}

func MapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func TimeTick(fn func(), d time.Duration) func() {
	ticker := time.NewTicker(d)
	stopped := make(chan bool)
	alreadyClosed := false
	go func() {
	out:
		for {
			select {
			case <-stopped:
				break out
			case <-ticker.C:
				fn()
			}
		}
	}()
	return func() {
		if alreadyClosed {
			return
		}
		alreadyClosed = true
		ticker.Stop()
		close(stopped)
	}
}

var bytesSizes = []string{"B", "K", "M", "G", "T"}

func FormatBytes(bytes uint64, decimals int) string {
	if bytes == 0 {
		return "0 B"
	}
	if decimals < 0 {
		decimals = 0
	}
	i := math.Floor(math.Log(float64(bytes)) / math.Log(1024))
	if int(i) >= len(bytesSizes) {
		i = float64(len(bytesSizes) - 1)
	}
	return fmt.Sprintf("%.*f %s", decimals, float64(bytes)/math.Pow(1024, i), bytesSizes[int(i)])
}

func BuildURL(pattern string, variables ...string) string {
	if len(variables) == 0 {
		return pattern
	}
	seg := strings.SplitN(pattern, "{}", len(variables)+1)
	i := 0
	j := 0
	pattern = ""
	for j < len(seg) {
		val := "{}"
		if i < len(variables) {
			val = strings.ReplaceAll(url.PathEscape(variables[i]), "%2F", "/")
		}
		pattern += seg[j]
		if j < len(seg)-1 {
			pattern += val
		}
		i++
		j++
	}
	return pattern
}

func URLEncodePath(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), "%2F", "/")
}

func Base64URLEncode(v []byte) string {
	s := base64.URLEncoding.EncodeToString(v)
	return strings.TrimRight(s, "=")
}

func Base64URLDecode(s string) ([]byte, error) {
	if len(s)%4 != 0 {
		s += strings.Repeat("=", 4-len(s)%4)
	}
	return base64.URLEncoding.DecodeString(s)
}

func LogSanitize(s string) string {
	s = strings.Replace(s, "\n", "", -1)
	s = strings.Replace(s, "\r", "", -1)
	return s
}
