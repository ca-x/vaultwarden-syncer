package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locales/*.json
var localeFiles embed.FS

// Language represents supported languages
type Language string

const (
	English Language = "en"
	Chinese Language = "zh"
)

// Translator handles internationalization
type Translator struct {
	translations map[Language]map[string]string
	fallback     Language
}

// New creates a new translator instance
func New() *Translator {
	t := &Translator{
		translations: make(map[Language]map[string]string),
		fallback:     English,
	}

	// Load default translations
	t.loadDefaultTranslations()

	return t
}

// SetFallback sets the fallback language
func (t *Translator) SetFallback(lang Language) {
	t.fallback = lang
}

// AddTranslations adds translations for a specific language
func (t *Translator) AddTranslations(lang Language, translations map[string]string) {
	if t.translations[lang] == nil {
		t.translations[lang] = make(map[string]string)
	}

	for key, value := range translations {
		t.translations[lang][key] = value
	}
}

// LoadTranslationsFromJSON loads translations from JSON data
func (t *Translator) LoadTranslationsFromJSON(lang Language, jsonData []byte) error {
	var translations map[string]string
	if err := json.Unmarshal(jsonData, &translations); err != nil {
		return fmt.Errorf("failed to unmarshal translations: %w", err)
	}

	t.AddTranslations(lang, translations)
	return nil
}

// T translates a key for the specified language
func (t *Translator) T(lang Language, key string, args ...interface{}) string {
	// Try to get translation for requested language
	if langTranslations, exists := t.translations[lang]; exists {
		if translation, exists := langTranslations[key]; exists {
			if len(args) > 0 {
				return fmt.Sprintf(translation, args...)
			}
			return translation
		}
	}

	// Fallback to default language
	if fallbackTranslations, exists := t.translations[t.fallback]; exists {
		if translation, exists := fallbackTranslations[key]; exists {
			if len(args) > 0 {
				return fmt.Sprintf(translation, args...)
			}
			return translation
		}
	}

	// Return key if no translation found
	return key
}

// GetSupportedLanguages returns list of supported languages
func (t *Translator) GetSupportedLanguages() []Language {
	var languages []Language
	for lang := range t.translations {
		languages = append(languages, lang)
	}
	return languages
}

// DetectLanguageFromHeader detects language from Accept-Language header
func (t *Translator) DetectLanguageFromHeader(acceptLang string) Language {
	if acceptLang == "" {
		return t.fallback
	}

	// Simple detection based on common patterns
	acceptLang = strings.ToLower(acceptLang)

	if strings.Contains(acceptLang, "zh") {
		return Chinese
	}

	return English
}

// loadDefaultTranslations loads built-in translations from embedded files
func (t *Translator) loadDefaultTranslations() {
	// Load English translations
	if enData, err := localeFiles.ReadFile("locales/en.json"); err == nil {
		t.LoadTranslationsFromJSON(English, enData)
	}

	// Load Chinese translations
	if zhData, err := localeFiles.ReadFile("locales/zh.json"); err == nil {
		t.LoadTranslationsFromJSON(Chinese, zhData)
	}
}
