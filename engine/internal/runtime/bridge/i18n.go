package bridge

import (
	"github.com/bitcode-framework/bitcode/internal/infrastructure/i18n"
)

type i18nBridge struct {
	translator *i18n.Translator
}

func newI18NBridge(translator *i18n.Translator) *i18nBridge {
	return &i18nBridge{translator: translator}
}

func (i *i18nBridge) Translate(locale, key string) string {
	return i.translator.Translate(locale, key)
}
