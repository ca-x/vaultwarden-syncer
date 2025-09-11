package i18n

import (
	"context"

	"github.com/labstack/echo/v4"
)

// contextKey is used for context values
type contextKey string

const (
	// LanguageContextKey is the key for storing language in context
	LanguageContextKey contextKey = "language"
	// TranslatorContextKey is the key for storing translator in context
	TranslatorContextKey contextKey = "translator"
)

// Middleware creates an i18n middleware
func Middleware(translator *Translator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Detect language from various sources
			lang := detectLanguage(c, translator)

			// Store language and translator in context
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, LanguageContextKey, lang)
			ctx = context.WithValue(ctx, TranslatorContextKey, translator)

			// Update request context
			c.SetRequest(c.Request().WithContext(ctx))

			// Set response header for language
			c.Response().Header().Set("Content-Language", string(lang))

			return next(c)
		}
	}
}

// detectLanguage detects language from multiple sources
func detectLanguage(c echo.Context, translator *Translator) Language {
	// 1. Check query parameter
	if langParam := c.QueryParam("lang"); langParam != "" {
		if lang := Language(langParam); isValidLanguage(translator, lang) {
			return lang
		}
	}

	// 2. Check cookie
	if cookie, err := c.Cookie("language"); err == nil {
		if lang := Language(cookie.Value); isValidLanguage(translator, lang) {
			return lang
		}
	}

	// 3. Check Accept-Language header
	acceptLang := c.Request().Header.Get("Accept-Language")
	return translator.DetectLanguageFromHeader(acceptLang)
}

// isValidLanguage checks if the language is supported
func isValidLanguage(translator *Translator, lang Language) bool {
	supportedLangs := translator.GetSupportedLanguages()
	for _, supported := range supportedLangs {
		if supported == lang {
			return true
		}
	}
	return false
}

// GetLanguageFromContext retrieves language from context
func GetLanguageFromContext(ctx context.Context) Language {
	if lang, ok := ctx.Value(LanguageContextKey).(Language); ok {
		return lang
	}
	return English // fallback
}

// GetTranslatorFromContext retrieves translator from context
func GetTranslatorFromContext(ctx context.Context) *Translator {
	if translator, ok := ctx.Value(TranslatorContextKey).(*Translator); ok {
		return translator
	}
	return nil
}

// T is a helper function to translate text from context
func T(ctx context.Context, key string, args ...interface{}) string {
	translator := GetTranslatorFromContext(ctx)
	if translator == nil {
		return key
	}

	lang := GetLanguageFromContext(ctx)
	return translator.T(lang, key, args...)
}
