package format

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

type mockSequenceProvider struct {
	values map[string]int64
	calls  []sequenceCall
}

type sequenceCall struct {
	modelName   string
	fieldName   string
	sequenceKey string
	step        int
}

func newMockSequenceProvider() *mockSequenceProvider {
	return &mockSequenceProvider{values: make(map[string]int64)}
}

func (m *mockSequenceProvider) NextValue(modelName, fieldName, sequenceKey string, step int) (int64, error) {
	key := fmt.Sprintf("%s|%s|%s|%d", modelName, fieldName, sequenceKey, step)
	m.values[key]++
	m.calls = append(m.calls, sequenceCall{
		modelName:   modelName,
		fieldName:   fieldName,
		sequenceKey: sequenceKey,
		step:        step,
	})
	return m.values[key], nil
}

func fixedContext() *FormatContext {
	return &FormatContext{
		Data: map[string]any{
			"customer": "Acme Corp",
			"dept":     "finance",
			"nik":      "EMP001234",
			"code":     42,
		},
		Session: map[string]any{
			"user_id": "usr-123",
			"role":    "manager",
		},
		Settings: map[string]string{
			"company": "BitCode",
			"region":  "ID",
		},
		ModelName: "invoice",
		Module:    "sales",
		Now:       time.Date(2026, time.April, 24, 9, 30, 45, 0, time.UTC),
	}
}

func TestResolveDataToken(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())

	got, err := engine.Resolve("{data.customer}", fixedContext(), "invoice", "number", "", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "Acme Corp" {
		t.Fatalf("expected Acme Corp, got %q", got)
	}
}

func TestResolveTimeTokens(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())
	ctx := fixedContext()

	got, err := engine.Resolve("{time.now}|{time.year}|{time.month}|{time.day}|{time.hour}|{time.minute}|{time.date}|{time.unix}", ctx, "invoice", "number", "", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	want := strings.Join([]string{
		"2026-04-24T09:30:45Z",
		"2026",
		"04",
		"24",
		"09",
		"30",
		"2026-04-24",
		"1777023045",
	}, "|")

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveSessionSettingAndModelTokens(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())

	got, err := engine.Resolve("{session.user_id}/{setting.company}/{model.name}/{model.module}", fixedContext(), "invoice", "number", "", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "usr-123/BitCode/invoice/sales" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestResolveFunctions(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())

	got, err := engine.Resolve("{upper(data.dept)}|{lower(setting.company)}|{substring(data.nik,0,3)}|{hash(data.customer)}", fixedContext(), "invoice", "number", "", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	parts := strings.Split(got, "|")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(parts))
	}
	if parts[0] != "FINANCE" {
		t.Fatalf("expected FINANCE, got %q", parts[0])
	}
	if parts[1] != "bitcode" {
		t.Fatalf("expected bitcode, got %q", parts[1])
	}
	if parts[2] != "EMP" {
		t.Fatalf("expected EMP, got %q", parts[2])
	}
	if matched := regexp.MustCompile(`^[a-f0-9]{8}$`).MatchString(parts[3]); !matched {
		t.Fatalf("expected 8-char hex hash, got %q", parts[3])
	}
}

func TestResolveUUIDTokens(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())

	got, err := engine.Resolve("{uuid4}|{uuid7}", fixedContext(), "invoice", "number", "", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	parts := strings.Split(got, "|")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	for _, part := range parts {
		if len(part) != 36 {
			t.Fatalf("expected uuid length 36, got %q", part)
		}
	}
}

func TestResolveRandomTokens(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())

	got, err := engine.Resolve("{random(12)}|{random_fixed(6,10,99)}", fixedContext(), "invoice", "number", "", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	parts := strings.Split(got, "|")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if len(parts[0]) != 12 {
		t.Fatalf("expected random length 12, got %d", len(parts[0]))
	}
	if matched := regexp.MustCompile(`^[A-Za-z0-9]{12}$`).MatchString(parts[0]); !matched {
		t.Fatalf("expected alphanumeric random string, got %q", parts[0])
	}
	if len(parts[1]) != 6 {
		t.Fatalf("expected fixed random length 6, got %d", len(parts[1]))
	}
	if matched := regexp.MustCompile(`^0000[1-9][0-9]?$`).MatchString(parts[1]); !matched {
		t.Fatalf("expected zero-padded value between 10 and 99, got %q", parts[1])
	}
}

func TestResolveCombinedTemplateSequenceLast(t *testing.T) {
	sp := newMockSequenceProvider()
	engine := NewEngine(sp)

	got, err := engine.Resolve("INV/{time.year}/{sequence(6)}", fixedContext(), "invoice", "number", "never", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "INV/2026/000001" {
		t.Fatalf("expected INV/2026/000001, got %q", got)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("expected 1 sequence call, got %d", len(sp.calls))
	}
	if sp.calls[0].sequenceKey != "invoice:number" {
		t.Fatalf("expected base sequence key, got %q", sp.calls[0].sequenceKey)
	}
}

func TestResolveKeyResetModeUsesResolvedTemplateWithoutSequence(t *testing.T) {
	sp := newMockSequenceProvider()
	engine := NewEngine(sp)

	got, err := engine.Resolve("INV/{time.year}/{data.dept}/{sequence(4)}", fixedContext(), "invoice", "number", "key", 3)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "INV/2026/finance/0001" {
		t.Fatalf("expected INV/2026/finance/0001, got %q", got)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("expected 1 sequence call, got %d", len(sp.calls))
	}
	if sp.calls[0].sequenceKey != "INV/2026/finance/" {
		t.Fatalf("expected resolved key reset sequence key, got %q", sp.calls[0].sequenceKey)
	}
	if sp.calls[0].step != 3 {
		t.Fatalf("expected step 3, got %d", sp.calls[0].step)
	}
}

func TestResolveTimeBasedResetModeUsesTimeComponent(t *testing.T) {
	sp := newMockSequenceProvider()
	engine := NewEngine(sp)

	got, err := engine.Resolve("INV/{sequence(4)}", fixedContext(), "invoice", "number", "monthly", 1)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "INV/0001" {
		t.Fatalf("expected INV/0001, got %q", got)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("expected 1 sequence call, got %d", len(sp.calls))
	}
	if sp.calls[0].sequenceKey != "invoice:number:2026-04" {
		t.Fatalf("expected monthly sequence key, got %q", sp.calls[0].sequenceKey)
	}
}

func TestResolveMissingDataFieldReturnsError(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())

	_, err := engine.Resolve("{data.missing}", fixedContext(), "invoice", "number", "", 1)
	if err == nil {
		t.Fatal("expected error for missing data field")
	}
}

func TestResolveWithoutSequenceProviderReturnsError(t *testing.T) {
	engine := NewEngine(nil)

	_, err := engine.Resolve("{sequence(4)}", fixedContext(), "invoice", "number", "", 1)
	if err == nil {
		t.Fatal("expected error when sequence provider is nil")
	}
}

func TestResolveInvalidFunctionArgumentsReturnError(t *testing.T) {
	engine := NewEngine(newMockSequenceProvider())

	_, err := engine.Resolve("{substring(data.nik,nope,3)}", fixedContext(), "invoice", "number", "", 1)
	if err == nil {
		t.Fatal("expected error for invalid substring argument")
	}
}
