package i18n

import (
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/utils"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

type FileMessageSource struct {
	defaultLang language.Tag
	msgMap      map[language.Tag]map[string]string
	matcher     language.Matcher
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
	log.Printf("[i18n] %d languages loaded: %v", len(lang), lang)

	def, e := language.Parse(config.DefaultLang)
	if e != nil {
		def = language.AmericanEnglish
	}
	log.Printf("[i18n] default language: %v", def)

	return &FileMessageSource{
		defaultLang: def,
		msgMap:      msg,
		matcher:     language.NewMatcher(lang),
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
	matched, _, c := f.matcher.Match(tag)
	if c >= language.High {
		tag = matched
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
	files, e := ioutil.ReadDir(path)
	if e != nil {
		return nil, e
	}
	r := make(map[language.Tag]map[string]string)
	for _, file := range files {
		lang := file.Name()
		if strings.HasSuffix(lang, ".yml") {
			lang = lang[:len(lang)-4]
		}

		langTag, e := language.Parse(lang)
		if e != nil {
			log.Printf("[i18n] ignore unknown language tag for file '%s': %v", file.Name(), e)
			continue
		}

		bytes, e := ioutil.ReadFile(filepath.Join(path, file.Name()))
		if e != nil {
			return nil, e
		}
		items := make(map[string]interface{})
		if e := yaml.Unmarshal(bytes, items); e != nil {
			return nil, errors.New(fmt.Sprintf("error parsing file '%s': %s", file.Name(), e.Error()))
		}
		messages := utils.FlattenStringMap(items, ".")
		r[langTag] = messages
	}
	return r, nil
}
