package sync

import (
	"testing"
)

func TestDetectInventoryFields_WithDeltas(t *testing.T) {
	payload := map[string]interface{}{
		"id":        "rec-1",
		"name":      "Widget",
		"qty_delta": float64(-5),
		"price":     float64(100),
	}

	deltas := DetectInventoryFields(payload)

	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta field, got %d", len(deltas))
	}

	if deltas["qty"] != -5 {
		t.Errorf("expected qty delta = -5, got %v", deltas["qty"])
	}
}

func TestDetectInventoryFields_MultipleDeltas(t *testing.T) {
	payload := map[string]interface{}{
		"qty_delta":   float64(-3),
		"stock_delta": float64(10),
		"name":        "Gadget",
	}

	deltas := DetectInventoryFields(payload)

	if len(deltas) != 2 {
		t.Fatalf("expected 2 delta fields, got %d", len(deltas))
	}

	if deltas["qty"] != -3 {
		t.Errorf("expected qty delta = -3, got %v", deltas["qty"])
	}
	if deltas["stock"] != 10 {
		t.Errorf("expected stock delta = 10, got %v", deltas["stock"])
	}
}

func TestDetectInventoryFields_NoDeltas(t *testing.T) {
	payload := map[string]interface{}{
		"name":  "Widget",
		"price": float64(100),
		"qty":   float64(50),
	}

	deltas := DetectInventoryFields(payload)

	if len(deltas) != 0 {
		t.Errorf("expected 0 delta fields, got %d", len(deltas))
	}
}

func TestDetectInventoryFields_IntegerDelta(t *testing.T) {
	payload := map[string]interface{}{
		"qty_delta": 7,
	}

	deltas := DetectInventoryFields(payload)

	if deltas["qty"] != 7 {
		t.Errorf("expected qty delta = 7, got %v", deltas["qty"])
	}
}

func TestDetectInventoryFields_Int64Delta(t *testing.T) {
	payload := map[string]interface{}{
		"qty_delta": int64(-12),
	}

	deltas := DetectInventoryFields(payload)

	if deltas["qty"] != -12 {
		t.Errorf("expected qty delta = -12, got %v", deltas["qty"])
	}
}

func TestDetectInventoryFields_IgnoresNonNumeric(t *testing.T) {
	payload := map[string]interface{}{
		"qty_delta": "not-a-number",
	}

	deltas := DetectInventoryFields(payload)

	if len(deltas) != 0 {
		t.Errorf("expected 0 deltas for non-numeric value, got %d", len(deltas))
	}
}

func TestDetectInventoryFields_ShortFieldName(t *testing.T) {
	payload := map[string]interface{}{
		"_delta": float64(5),
	}

	deltas := DetectInventoryFields(payload)

	if len(deltas) != 0 {
		t.Errorf("expected 0 deltas for field '_delta' (too short), got %d", len(deltas))
	}
}

func TestIsValidTableName_ForInventory(t *testing.T) {
	if !isValidTableName("products") {
		t.Error("'products' should be valid")
	}
	if !isValidTableName("inventory_items") {
		t.Error("'inventory_items' should be valid")
	}
	if isValidTableName("") {
		t.Error("empty string should be invalid")
	}
	if isValidTableName("drop;table") {
		t.Error("SQL injection should be invalid")
	}
}
