// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package localization

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// jsonUnmarshal is an alias so tests can substitute it.
var jsonUnmarshal = json.Unmarshal

type Language string

const (
	English Language = "en"
	Spanish Language = "es"
	Chinese Language = "zh"
)

var supported = map[Language]bool{
	English: true,
	Spanish: true,
	Chinese: true,
}

type Localizer struct {
	mu          sync.RWMutex
	lang        Language
	messages    map[Language]map[string]string
	defaultLang Language
}

func New() *Localizer {
	lang := detectLanguage()
	return &Localizer{
		lang:        lang,
		messages:    make(map[Language]map[string]string),
		defaultLang: English,
	}
}

func detectLanguage() Language {
	envLang := os.Getenv("GLASSBOX_LANG")
	if envLang == "" {
		return English
	}

	lang := Language(strings.ToLower(strings.TrimSpace(envLang)))
	if supported[lang] {
		return lang
	}

	return English
}

func (l *Localizer) SetLanguage(lang Language) error {
	if !supported[lang] {
		return fmt.Errorf("unsupported language: %s", lang)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.lang = lang
	return nil
}

func (l *Localizer) GetLanguage() Language {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lang
}

func (l *Localizer) RegisterMessages(lang Language, messages map[string]string) error {
	if !supported[lang] {
		return fmt.Errorf("unsupported language: %s", lang)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.messages[lang] == nil {
		l.messages[lang] = make(map[string]string)
	}

	for key, msg := range messages {
		l.messages[lang][key] = msg
	}

	return nil
}

func (l *Localizer) Get(key string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if msg, ok := l.messages[l.lang][key]; ok {
		return msg
	}

	if msg, ok := l.messages[l.defaultLang][key]; ok {
		return msg
	}

	return key
}

func (l *Localizer) GetForLang(lang Language, key string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if msg, ok := l.messages[lang][key]; ok {
		return msg
	}

	if msg, ok := l.messages[l.defaultLang][key]; ok {
		return msg
	}

	return key
}

func (l *Localizer) Translate(key string, args ...interface{}) string {
	template := l.Get(key)
	if len(args) > 0 {
		return fmt.Sprintf(template, args...)
	}
	return template
}

func (l *Localizer) TranslateForLang(lang Language, key string, args ...interface{}) string {
	template := l.GetForLang(lang, key)
	if len(args) > 0 {
		return fmt.Sprintf(template, args...)
	}
	return template
}

// LoadFromFile reads a JSON translation bundle from the given file path and
// registers the messages for the specified language. The file must contain a
// flat JSON object mapping message keys to translated strings, e.g.:
//
//	{"cli.debug.short": "Debug a failed transaction", ...}
func (l *Localizer) LoadFromFile(lang Language, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("localization: failed to read translation file %s: %w", path, err)
	}

	var messages map[string]string
	if err := jsonUnmarshal(data, &messages); err != nil {
		return fmt.Errorf("localization: failed to parse translation file %s: %w", path, err)
	}

	return l.RegisterMessages(lang, messages)
}

// SupportedLanguages returns the list of language codes recognised by the localizer.
func SupportedLanguages() []Language {
	langs := make([]Language, 0, len(supported))
	for l := range supported {
		langs = append(langs, l)
	}
	return langs
}

var globalLocalizer = New()

func Get(key string) string {
	return globalLocalizer.Get(key)
}

func Translate(key string, args ...interface{}) string {
	return globalLocalizer.Translate(key, args...)
}

func SetLanguage(lang Language) error {
	return globalLocalizer.SetLanguage(lang)
}

func GetLanguage() Language {
	return globalLocalizer.GetLanguage()
}

func RegisterMessages(lang Language, messages map[string]string) error {
	return globalLocalizer.RegisterMessages(lang, messages)
}

// LoadFromFile loads a JSON translation bundle into the global localizer.
func LoadFromFile(lang Language, path string) error {
	return globalLocalizer.LoadFromFile(lang, path)
}

// DetectLanguage returns the language derived from the GLASSBOX_LANG environment
// variable, falling back to English for unrecognised values.
func DetectLanguage() Language {
	return detectLanguage()
}
