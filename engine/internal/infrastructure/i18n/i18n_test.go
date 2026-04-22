package i18n

import "testing"

func TestTranslator_Basic(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadJSON([]byte(`{
		"locale": "id",
		"translations": {
			"Sales Order": "Pesanan Penjualan",
			"Customer": "Pelanggan"
		}
	}`))

	result := tr.Translate("id", "Sales Order")
	if result != "Pesanan Penjualan" {
		t.Errorf("expected Pesanan Penjualan, got %s", result)
	}
}

func TestTranslator_FallbackToDefault(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadJSON([]byte(`{
		"locale": "en",
		"translations": { "Hello": "Hello" }
	}`))

	result := tr.Translate("id", "Hello")
	if result != "Hello" {
		t.Errorf("expected fallback to en, got %s", result)
	}
}

func TestTranslator_FallbackToKey(t *testing.T) {
	tr := NewTranslator("en")
	result := tr.Translate("id", "Unknown Key")
	if result != "Unknown Key" {
		t.Errorf("expected key as fallback, got %s", result)
	}
}

func TestTranslator_HasLocale(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadJSON([]byte(`{"locale": "id", "translations": {"a": "b"}}`))

	if !tr.HasLocale("id") {
		t.Error("should have locale id")
	}
	if tr.HasLocale("fr") {
		t.Error("should not have locale fr")
	}
}
