package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type TranslationFile struct {
	Locale       string            `json:"locale"`
	Translations map[string]string `json:"translations"`
}

type Translator struct {
	translations  map[string]map[string]string
	defaultLocale string
	mu            sync.RWMutex
}

func NewTranslator(defaultLocale string) *Translator {
	return &Translator{
		translations:  make(map[string]map[string]string),
		defaultLocale: defaultLocale,
	}
}

func (t *Translator) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read i18n file %s: %w", path, err)
	}

	var tf TranslationFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return fmt.Errorf("invalid i18n JSON %s: %w", path, err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.translations[tf.Locale] == nil {
		t.translations[tf.Locale] = make(map[string]string)
	}
	for key, val := range tf.Translations {
		t.translations[tf.Locale][key] = val
	}
	return nil
}

func (t *Translator) LoadJSON(data []byte) error {
	var tf TranslationFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return fmt.Errorf("invalid i18n JSON: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.translations[tf.Locale] == nil {
		t.translations[tf.Locale] = make(map[string]string)
	}
	for key, val := range tf.Translations {
		t.translations[tf.Locale][key] = val
	}
	return nil
}

func (t *Translator) Translate(locale string, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if trans, ok := t.translations[locale]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}

	if locale != t.defaultLocale {
		if trans, ok := t.translations[t.defaultLocale]; ok {
			if val, ok := trans[key]; ok {
				return val
			}
		}
	}

	return key
}

func (t *Translator) HasLocale(locale string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.translations[locale]
	return ok
}
