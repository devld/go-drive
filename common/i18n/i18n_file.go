package i18n

import (
	"fmt"
	"go-drive/common"
	"go-drive/common/utils"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
)

type FileMessageSource struct {
	defaultLang language.Tag
	msgMap      map[language.Tag]map[string]string
	matcher     language.Matcher
	languages   []language.Tag
}

// NewFileMessageSource creates a MessageSource read translated texts from file
func NewFileMessageSource(config common.Config) (*FileMessageSource, error) {
	langDir := config.LangDir

	msg := make(map[language.Tag]map[string]string)

	if exists, _ := utils.FileExists(langDir); exists {
		temp, e := readAllLang(langDir)
		if e != nil {
			return nil, e
		}
		msg = temp
	} else {
		log.Println("[i18n] no languages configuration found.")
	}

	lang := make([]language.Tag, 0, len(msg))
	for lt := range msg {
		lang = append(lang, lt)
	}
	sort.Slice(lang, func(i, j int) bool { return lang[i].String() < lang[j].String() })
	log.Printf("[i18n] %d languages loaded: %v", len(lang), lang)

	def, e := language.Parse(config.DefaultLang)
	if e != nil {
		def = language.AmericanEnglish
	}
	matcher := language.NewMatcher(lang)
	if len(lang) > 0 {
		_, index, confidence := matcher.Match(def)
		if confidence >= language.High {
			def = lang[index]
		}
	}
	log.Printf("[i18n] default language: %v", def)

	return &FileMessageSource{
		defaultLang: def,
		msgMap:      msg,
		matcher:     matcher,
		languages:   lang,
	}, nil
}

func (f *FileMessageSource) Translate(lang, key string, args ...string) string {
	return Translate(f.getMessage(key, lang), args...)
}

func (f *FileMessageSource) getMessage(key, lang string) string {
	tag, e := language.Parse(lang)
	if e != nil {
		tag = f.defaultLang
	}
	_, index, c := f.matcher.Match(tag)
	if c >= language.High && index < len(f.languages) {
		tag = f.languages[index]
	} else {
		tag = f.defaultLang
	}
	msg, ok := f.msgMap[tag][key]
	if !ok {
		msg, ok = f.msgMap[f.defaultLang][key]
		if !ok {
			msg = key
		}
	}
	return msg
}

func readAllLang(path string) (map[language.Tag]map[string]string, error) {
	files, e := os.ReadDir(path)
	if e != nil {
		return nil, e
	}
	r := make(map[language.Tag]map[string]string)
	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		lang := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		langTag, e := language.Parse(lang)
		if e != nil {
			log.Printf("[i18n] ignore unknown language tag for file '%s': %v", file.Name(), e)
			continue
		}

		bytes, e := os.ReadFile(filepath.Join(path, file.Name()))
		if e != nil {
			return nil, e
		}
		items := make(map[string]any)
		if e := yaml.Unmarshal(bytes, items); e != nil {
			return nil, fmt.Errorf("error parsing file '%s': %s", file.Name(), e.Error())
		}
		messages := utils.FlattenStringMap(items, ".")
		if _, exists := r[langTag]; exists {
			return nil, fmt.Errorf("duplicate language tag %q from file %q", langTag, file.Name())
		}
		r[langTag] = messages
	}
	return r, nil
}
