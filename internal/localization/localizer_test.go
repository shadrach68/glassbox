// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package localization

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		expect Language
	}{
		{"english default", "", English},
		{"english explicit", "en", English},
		{"spanish", "es", Spanish},
		{"chinese", "zh", Chinese},
		{"invalid fallback", "fr", English},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GLASSBOX_LANG", tt.env)
			lang := detectLanguage()
			if lang != tt.expect {
				t.Errorf("expected %s, got %s", tt.expect, lang)
			}
		})
	}
}

func TestLocalizerSetLanguage(t *testing.T) {
	loc := New()

	err := loc.SetLanguage(Spanish)
	if err != nil {
		t.Errorf("failed to set valid language: %v", err)
	}

	if loc.GetLanguage() != Spanish {
		t.Error("language not updated")
	}

	err = loc.SetLanguage(Language("fr"))
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestLocalizerRegisterMessages(t *testing.T) {
	loc := New()

	msgs := map[string]string{
		"greeting": "Hello",
		"farewell": "Goodbye",
	}

	err := loc.RegisterMessages(English, msgs)
	if err != nil {
		t.Errorf("failed to register messages: %v", err)
	}

	if loc.Get("greeting") != "Hello" {
		t.Error("message not retrieved correctly")
	}
}

func TestLocalizerFallback(t *testing.T) {
	loc := New()

	msgs := map[string]string{
		"key1": "English message",
	}

	_ = loc.RegisterMessages(English, msgs)
	_ = loc.RegisterMessages(Spanish, map[string]string{})

	_ = loc.SetLanguage(Spanish)

	result := loc.Get("key1")
	if result != "English message" {
		t.Errorf("expected fallback to English, got: %s", result)
	}
}

func TestTranslateWithArgs(t *testing.T) {
	loc := New()

	msgs := map[string]string{
		"error.network": "invalid network: %s",
	}

	_ = loc.RegisterMessages(English, msgs)

	result := loc.Translate("error.network", "testnet")
	if result != "invalid network: testnet" {
		t.Errorf("expected formatted message, got: %s", result)
	}
}

func TestLoadTranslations(t *testing.T) {
	err := LoadTranslations()
	if err != nil {
		t.Errorf("failed to load translations: %v", err)
	}

	if Get("cli.debug.short") == "" {
		t.Error("translation not loaded")
	}
}

func TestMissingKeyFallback(t *testing.T) {
	loc := New()
	result := loc.Get("nonexistent.key")
	if result != "nonexistent.key" {
		t.Errorf("expected key as fallback, got: %s", result)
	}
}

func TestLoadFromFile_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "en.json")
	content := `{"greeting": "Hello from file", "farewell": "Goodbye from file"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loc := New()
	if err := loc.LoadFromFile(English, path); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if got := loc.Get("greeting"); got != "Hello from file" {
		t.Errorf("expected 'Hello from file', got %q", got)
	}
	if got := loc.Get("farewell"); got != "Goodbye from file" {
		t.Errorf("expected 'Goodbye from file', got %q", got)
	}
}

func TestLoadFromFile_FallbackAfterLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "es.json")
	content := `{"greeting": "Hola"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loc := New()
	_ = loc.RegisterMessages(English, map[string]string{"greeting": "Hello", "farewell": "Bye"})
	if err := loc.LoadFromFile(Spanish, path); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	_ = loc.SetLanguage(Spanish)

	if got := loc.Get("greeting"); got != "Hola" {
		t.Errorf("expected Spanish 'Hola', got %q", got)
	}
	// 'farewell' only exists in English — should fall back
	if got := loc.Get("farewell"); got != "Bye" {
		t.Errorf("expected English fallback 'Bye', got %q", got)
	}
}

func TestLoadFromFile_MissingFile(t *testing.T) {
	loc := New()
	err := loc.LoadFromFile(English, "/nonexistent/path/en.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	loc := New()
	err := loc.LoadFromFile(English, path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadFromFile_UnsupportedLanguage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fr.json")
	if err := os.WriteFile(path, []byte(`{"key": "val"}`), 0644); err != nil {
		t.Fatal(err)
	}

	loc := New()
	err := loc.LoadFromFile(Language("fr"), path)
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestSupportedLanguages(t *testing.T) {
	langs := SupportedLanguages()
	if len(langs) == 0 {
		t.Error("expected at least one supported language")
	}
	found := false
	for _, l := range langs {
		if l == English {
			found = true
		}
	}
	if !found {
		t.Error("expected English in supported languages")
	}
}

func TestDetectLanguagePublic(t *testing.T) {
	t.Setenv("GLASSBOX_LANG", "es")
	lang := DetectLanguage()
	if lang != Spanish {
		t.Errorf("expected Spanish, got %s", lang)
	}
}

func TestGetLanguageGlobal(t *testing.T) {
	// Reset global state after the test.
	orig := globalLocalizer.GetLanguage()
	t.Cleanup(func() { _ = globalLocalizer.SetLanguage(orig) })

	_ = SetLanguage(Chinese)
	if got := GetLanguage(); got != Chinese {
		t.Errorf("expected Chinese, got %s", got)
	}
}

func TestGlobalLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "en_extra.json")
	content := `{"test.file_load": "loaded from file"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := LoadFromFile(English, path); err != nil {
		t.Fatalf("global LoadFromFile failed: %v", err)
	}

	if got := Get("test.file_load"); got != "loaded from file" {
		t.Errorf("expected 'loaded from file', got %q", got)
	}
}
