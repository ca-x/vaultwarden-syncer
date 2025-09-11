package i18n

import (
	"testing"
)

func TestTranslator(t *testing.T) {
	translator := New()

	// Test English translations
	englishHello := translator.T(English, "auth.login.title")
	if englishHello != "Login" {
		t.Errorf("Expected 'Login', got '%s'", englishHello)
	}

	// Test Chinese translations
	chineseHello := translator.T(Chinese, "auth.login.title")
	if chineseHello != "登录" {
		t.Errorf("Expected '登录', got '%s'", chineseHello)
	}

	// Test fallback to English
	unknownLangHello := translator.T(Language("fr"), "auth.login.title")
	if unknownLangHello != "Login" {
		t.Errorf("Expected 'Login' (fallback), got '%s'", unknownLangHello)
	}

	// Test parameterized translations
	englishStorageCount := translator.T(English, "dashboard.storage_count", 3)
	if englishStorageCount != "3 storage backend(s) configured" {
		t.Errorf("Expected '3 storage backend(s) configured', got '%s'", englishStorageCount)
	}

	chineseStorageCount := translator.T(Chinese, "dashboard.storage_count", 3)
	if chineseStorageCount != "已配置 3 个存储后端" {
		t.Errorf("Expected '已配置 3 个存储后端', got '%s'", chineseStorageCount)
	}

	// Test unknown key
	unknownKey := translator.T(English, "unknown.key")
	if unknownKey != "unknown.key" {
		t.Errorf("Expected 'unknown.key', got '%s'", unknownKey)
	}
}

func TestLanguageDetection(t *testing.T) {
	translator := New()

	// Test Chinese detection
	chineseLang := translator.DetectLanguageFromHeader("zh-CN,zh;q=0.9,en;q=0.8")
	if chineseLang != Chinese {
		t.Errorf("Expected Chinese, got '%s'", chineseLang)
	}

	// Test English detection
	englishLang := translator.DetectLanguageFromHeader("en-US,en;q=0.9")
	if englishLang != English {
		t.Errorf("Expected English, got '%s'", englishLang)
	}

	// Test fallback to English
	fallbackLang := translator.DetectLanguageFromHeader("")
	if fallbackLang != English {
		t.Errorf("Expected English (fallback), got '%s'", fallbackLang)
	}
}

func TestSupportedLanguages(t *testing.T) {
	translator := New()
	supported := translator.GetSupportedLanguages()

	// Should have at least English and Chinese
	if len(supported) < 2 {
		t.Errorf("Expected at least 2 supported languages, got %d", len(supported))
	}

	// Check that English and Chinese are supported
	foundEnglish := false
	foundChinese := false
	for _, lang := range supported {
		if lang == English {
			foundEnglish = true
		}
		if lang == Chinese {
			foundChinese = true
		}
	}

	if !foundEnglish {
		t.Error("Expected English to be supported")
	}

	if !foundChinese {
		t.Error("Expected Chinese to be supported")
	}
}
