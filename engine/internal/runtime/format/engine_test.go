package format

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type mockSequenceProvider struct {
	counter map[string]int64
}

func newMockSequenceProvider() *mockSequenceProvider {
	return &mockSequenceProvider{counter: make(map[string]int64)}
}

func (m *mockSequenceProvider) NextValue(modelName, fieldName, sequenceKey string, step int) (int64, error) {
	key := modelName + ":" + fieldName + ":" + sequenceKey
	m.counter[key]++
	return m.counter[key], nil
}

func fixedTime() time.Time {
	return time.Date(2026, 4, 24, 10, 30, 0, 0, time.UTC)
}

func baseCtx() *FormatContext {
	return &FormatContext{
		Data: map[string]any{
			"nik":           "3201234567",
			"customer_type": "corporate",
			"name":          "Budi Santoso",
			"dept":          "sales",
			"email":         "budi@test.com",
			"branch_code":   "JKT",
		},
		Session: map[string]any{
			"user_id":    "usr-001",
			"username":   "admin",
			"tenant_id":  "tenant-001",
			"group_id":   "grp-001",
			"group_code": "sales_team",
		},
		Settings: map[string]string{
			"company_code":  "ACME",
			"kode_instansi": "KEMENDIKBUD",
		},
		ModelName: "invoice",
		Module:    "sales",
		Now:       fixedTime(),
	}
}

func TestResolveDataToken(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{data.nik}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "3201234567" {
		t.Fatalf("expected 3201234567, got %s", result)
	}
}

func TestResolveTimeTokens(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	ctx := baseCtx()

	tests := []struct {
		tmpl string
		want string
	}{
		{"{time.year}", "2026"},
		{"{time.month}", "04"},
		{"{time.day}", "24"},
		{"{time.hour}", "10"},
		{"{time.minute}", "30"},
		{"{time.date}", "2026-04-24"},
		{"{time.unix}", fmt.Sprintf("%d", fixedTime().Unix())},
	}

	for _, tt := range tests {
		result, err := e.Resolve(tt.tmpl, ctx, "test", "id", "never", 1)
		if err != nil {
			t.Fatalf("template %s: %v", tt.tmpl, err)
		}
		if result != tt.want {
			t.Fatalf("template %s: expected %s, got %s", tt.tmpl, tt.want, result)
		}
	}
}

func TestResolveSessionTokens(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	ctx := baseCtx()

	result, err := e.Resolve("{session.user_id}-{session.group_code}", ctx, "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "usr-001-sales_team" {
		t.Fatalf("expected usr-001-sales_team, got %s", result)
	}
}

func TestResolveSettingToken(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{setting.company_code}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "ACME" {
		t.Fatalf("expected ACME, got %s", result)
	}
}

func TestResolveModelTokens(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{model.name}/{model.module}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "invoice/sales" {
		t.Fatalf("expected invoice/sales, got %s", result)
	}
}

func TestResolveSequence(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("INV/{time.year}/{sequence(6)}", baseCtx(), "invoice", "code", "yearly", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "INV/2026/000001" {
		t.Fatalf("expected INV/2026/000001, got %s", result)
	}
}

func TestResolveSequenceMiddle(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("INV/{sequence(4)}/{time.year}", baseCtx(), "invoice", "code", "yearly", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "INV/0001/2026" {
		t.Fatalf("expected INV/0001/2026, got %s", result)
	}
}

func TestResolveUpper(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{upper(data.dept)}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "SALES" {
		t.Fatalf("expected SALES, got %s", result)
	}
}

func TestResolveLower(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{lower(data.name)}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "budi santoso" {
		t.Fatalf("expected budi santoso, got %s", result)
	}
}

func TestResolveSubstring(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{substring(data.nik,0,3)}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "320" {
		t.Fatalf("expected 320, got %s", result)
	}
}

func TestResolveHash(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{hash(data.email)}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 8 {
		t.Fatalf("expected 8 char hash, got %d chars: %s", len(result), result)
	}
}

func TestResolveRandom(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{random(8)}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 8 {
		t.Fatalf("expected 8 chars, got %d: %s", len(result), result)
	}
}

func TestResolveRandomFixed(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{random_fixed(3,200,999)}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 chars, got %d: %s", len(result), result)
	}
}

func TestResolveUUID4(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{uuid4}", baseCtx(), "test", "id", "never", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 36 {
		t.Fatalf("expected UUID length 36, got %d: %s", len(result), result)
	}
}

func TestResolveComplexTemplate(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{upper(data.dept)}-{sequence(4)}-{time.year}", baseCtx(), "order", "code", "yearly", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "SALES-0001-2026" {
		t.Fatalf("expected SALES-0001-2026, got %s", result)
	}
}

func TestResolveSuratKeluarFormat(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	result, err := e.Resolve("{sequence(4)}/{upper(data.dept)}/{setting.kode_instansi}/{time.month}/{time.year}", baseCtx(), "surat", "nomor", "monthly", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "0001/SALES/KEMENDIKBUD/04/2026" {
		t.Fatalf("expected 0001/SALES/KEMENDIKBUD/04/2026, got %s", result)
	}
}

func TestResolveKeyResetMode(t *testing.T) {
	sp := newMockSequenceProvider()
	e := NewEngine(sp)

	result, err := e.Resolve("INV/{time.year}/{sequence(6)}", baseCtx(), "invoice", "code", "key", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result != "INV/2026/000001" {
		t.Fatalf("expected INV/2026/000001, got %s", result)
	}

	for key := range sp.counter {
		if !strings.Contains(key, "INV/2026/") {
			t.Fatalf("expected sequence key to contain INV/2026/, got key: %s", key)
		}
	}
}

func TestResolveMissingDataField(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	_, err := e.Resolve("{data.nonexistent}", baseCtx(), "test", "id", "never", 1)
	if err == nil {
		t.Fatal("expected error for missing data field")
	}
}

func TestResolveUnknownToken(t *testing.T) {
	e := NewEngine(newMockSequenceProvider())
	_, err := e.Resolve("{unknown_token}", baseCtx(), "test", "id", "never", 1)
	if err == nil {
		t.Fatal("expected error for unknown token")
	}
}
