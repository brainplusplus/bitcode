package steps

import (
	"context"
	"testing"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/runtime/executor"
)

func TestValidateHandler_EqPass(t *testing.T) {
	h := &ValidateHandler{}
	execCtx := &executor.Context{
		Input: map[string]any{"status": "draft"},
	}
	step := parser.StepDefinition{
		Type:  parser.StepValidate,
		Rules: map[string]map[string]any{"status": {"eq": "draft"}},
		Error: "status must be draft",
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("should pass: %v", err)
	}
}

func TestValidateHandler_EqFail(t *testing.T) {
	h := &ValidateHandler{}
	execCtx := &executor.Context{
		Input: map[string]any{"status": "confirmed"},
	}
	step := parser.StepDefinition{
		Type:  parser.StepValidate,
		Rules: map[string]map[string]any{"status": {"eq": "draft"}},
		Error: "Only draft orders can be confirmed",
	}

	err := h.Execute(context.Background(), execCtx, step)
	if err == nil {
		t.Fatal("should fail validation")
	}
	if err.Error() != "Only draft orders can be confirmed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateHandler_Required(t *testing.T) {
	h := &ValidateHandler{}
	execCtx := &executor.Context{
		Input: map[string]any{},
	}
	step := parser.StepDefinition{
		Type:  parser.StepValidate,
		Rules: map[string]map[string]any{"name": {"required": true}},
	}

	err := h.Execute(context.Background(), execCtx, step)
	if err == nil {
		t.Fatal("should fail for missing required field")
	}
}

func TestEmitHandler(t *testing.T) {
	h := &EmitHandler{}
	execCtx := &executor.Context{
		Input: map[string]any{},
	}
	step := parser.StepDefinition{
		Type:  parser.StepEmit,
		Event: "order.confirmed",
		Data:  map[string]any{"order_id": "123"},
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(execCtx.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(execCtx.Events))
	}
	if execCtx.Events[0].Name != "order.confirmed" {
		t.Errorf("expected order.confirmed, got %s", execCtx.Events[0].Name)
	}
}

func TestAssignHandler(t *testing.T) {
	h := &AssignHandler{}
	execCtx := &executor.Context{
		Input:     map[string]any{},
		Variables: map[string]any{},
	}
	step := parser.StepDefinition{
		Type:     parser.StepAssign,
		Variable: "total",
		Value:    42.5,
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCtx.Variables["total"] != 42.5 {
		t.Errorf("expected 42.5, got %v", execCtx.Variables["total"])
	}
}

func TestIfHandler_TrueBranch(t *testing.T) {
	h := &IfHandler{}
	execCtx := &executor.Context{
		Input:     map[string]any{"total": 15000},
		Variables: map[string]any{},
	}
	step := parser.StepDefinition{
		Type:      parser.StepIf,
		Condition: "{{input.total > 10000}}",
		Then:      "notify_manager",
		Else:      "send_confirmation",
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCtx.Variables["_goto"] != "notify_manager" {
		t.Errorf("expected notify_manager, got %v", execCtx.Variables["_goto"])
	}
}

func TestParseProcess(t *testing.T) {
	data := []byte(`{
		"name": "confirm_order",
		"steps": [
			{ "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "Only draft" },
			{ "type": "update", "model": "order", "set": { "status": "confirmed" } },
			{ "type": "emit", "event": "order.confirmed" }
		]
	}`)

	proc, err := parser.ParseProcess(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc.Name != "confirm_order" {
		t.Errorf("expected confirm_order, got %s", proc.Name)
	}
	if len(proc.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(proc.Steps))
	}
}

func TestParseProcess_MissingName(t *testing.T) {
	data := []byte(`{"steps": [{"type": "emit", "event": "test"}]}`)
	_, err := parser.ParseProcess(data)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseProcess_NoSteps(t *testing.T) {
	data := []byte(`{"name": "empty"}`)
	_, err := parser.ParseProcess(data)
	if err == nil {
		t.Fatal("expected error for no steps")
	}
}

type mockStepExecutor struct {
	executedSteps []parser.StepDefinition
}

func (m *mockStepExecutor) ExecuteSteps(ctx context.Context, execCtx *executor.Context, steps []parser.StepDefinition) error {
	m.executedSteps = append(m.executedSteps, steps...)
	for _, s := range steps {
		if s.Type == parser.StepAssign {
			execCtx.Variables[s.Variable] = s.Value
		}
	}
	return nil
}

func TestIfHandler_NestedThenSteps(t *testing.T) {
	mock := &mockStepExecutor{}
	h := &IfHandler{Executor: mock}
	execCtx := &executor.Context{
		Input:     map[string]any{"total": 15000},
		Variables: map[string]any{},
	}
	step := parser.StepDefinition{
		Type:      parser.StepIf,
		Condition: "{{input.total > 10000}}",
		ThenSteps: []parser.StepDefinition{
			{Type: parser.StepAssign, Variable: "tier", Value: "enterprise"},
		},
		ElseSteps: []parser.StepDefinition{
			{Type: parser.StepAssign, Variable: "tier", Value: "standard"},
		},
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.executedSteps) != 1 {
		t.Fatalf("expected 1 step executed, got %d", len(mock.executedSteps))
	}
	if mock.executedSteps[0].Variable != "tier" || mock.executedSteps[0].Value != "enterprise" {
		t.Errorf("expected then_steps to execute, got %+v", mock.executedSteps[0])
	}
}

func TestIfHandler_NestedElseSteps(t *testing.T) {
	mock := &mockStepExecutor{}
	h := &IfHandler{Executor: mock}
	execCtx := &executor.Context{
		Input:     map[string]any{"active": false},
		Variables: map[string]any{"active": false},
	}
	step := parser.StepDefinition{
		Type:      parser.StepIf,
		Condition: "{{active}}",
		ThenSteps: []parser.StepDefinition{
			{Type: parser.StepAssign, Variable: "result", Value: "yes"},
		},
		ElseSteps: []parser.StepDefinition{
			{Type: parser.StepAssign, Variable: "result", Value: "no"},
		},
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.executedSteps) != 1 {
		t.Fatalf("expected 1 step executed, got %d", len(mock.executedSteps))
	}
	if mock.executedSteps[0].Value != "no" {
		t.Errorf("expected else_steps to execute, got %+v", mock.executedSteps[0])
	}
}

func TestIfHandler_BackwardCompat(t *testing.T) {
	h := &IfHandler{}
	execCtx := &executor.Context{
		Input:     map[string]any{"total": 15000},
		Variables: map[string]any{},
	}
	step := parser.StepDefinition{
		Type:      parser.StepIf,
		Condition: "{{input.total > 10000}}",
		Then:      "notify_manager",
		Else:      "send_confirmation",
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCtx.Variables["_goto"] != "notify_manager" {
		t.Errorf("expected notify_manager, got %v", execCtx.Variables["_goto"])
	}
}

func TestSwitchHandler_CaseSteps(t *testing.T) {
	mock := &mockStepExecutor{}
	h := &SwitchHandler{Executor: mock}
	execCtx := &executor.Context{
		Input:     map[string]any{},
		Variables: map[string]any{"priority": "high"},
	}
	step := parser.StepDefinition{
		Type:  parser.StepSwitch,
		Field: "{{priority}}",
		CaseSteps: map[string][]parser.StepDefinition{
			"high": {
				{Type: parser.StepAssign, Variable: "handler", Value: "senior"},
			},
			"low": {
				{Type: parser.StepAssign, Variable: "handler", Value: "junior"},
			},
			"default": {
				{Type: parser.StepAssign, Variable: "handler", Value: "auto"},
			},
		},
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.executedSteps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(mock.executedSteps))
	}
	if mock.executedSteps[0].Value != "senior" {
		t.Errorf("expected senior, got %v", mock.executedSteps[0].Value)
	}
}

func TestSwitchHandler_CaseStepsDefault(t *testing.T) {
	mock := &mockStepExecutor{}
	h := &SwitchHandler{Executor: mock}
	execCtx := &executor.Context{
		Input:     map[string]any{},
		Variables: map[string]any{"priority": "unknown"},
	}
	step := parser.StepDefinition{
		Type:  parser.StepSwitch,
		Field: "{{priority}}",
		CaseSteps: map[string][]parser.StepDefinition{
			"high": {
				{Type: parser.StepAssign, Variable: "handler", Value: "senior"},
			},
			"default": {
				{Type: parser.StepAssign, Variable: "handler", Value: "auto"},
			},
		},
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.executedSteps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(mock.executedSteps))
	}
	if mock.executedSteps[0].Value != "auto" {
		t.Errorf("expected auto (default), got %v", mock.executedSteps[0].Value)
	}
}

func TestLoopHandler_ExecutesSubSteps(t *testing.T) {
	mock := &mockStepExecutor{}
	h := &LoopHandler{Executor: mock}
	execCtx := &executor.Context{
		Input:     map[string]any{},
		Variables: map[string]any{"items": []any{"a", "b", "c"}},
	}
	step := parser.StepDefinition{
		Type: parser.StepLoop,
		Over: "{{items}}",
		Steps: []parser.StepDefinition{
			{Type: parser.StepAssign, Variable: "processed", Value: true},
		},
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.executedSteps) != 3 {
		t.Fatalf("expected 3 step executions (one per item), got %d", len(mock.executedSteps))
	}
	if execCtx.Variables["_index"] != 2 {
		t.Errorf("expected _index=2 after loop, got %v", execCtx.Variables["_index"])
	}
	if execCtx.Variables["_item"] != "c" {
		t.Errorf("expected _item=c after loop, got %v", execCtx.Variables["_item"])
	}
}

func TestLoopHandler_EmptyList(t *testing.T) {
	mock := &mockStepExecutor{}
	h := &LoopHandler{Executor: mock}
	execCtx := &executor.Context{
		Input:     map[string]any{},
		Variables: map[string]any{"items": []any{}},
	}
	step := parser.StepDefinition{
		Type: parser.StepLoop,
		Over: "{{items}}",
		Steps: []parser.StepDefinition{
			{Type: parser.StepAssign, Variable: "processed", Value: true},
		},
	}

	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.executedSteps) != 0 {
		t.Errorf("expected 0 step executions for empty list, got %d", len(mock.executedSteps))
	}
}

func TestParseProcess_NestedSteps(t *testing.T) {
	data := []byte(`{
		"name": "nested_test",
		"steps": [
			{
				"type": "if",
				"condition": "{{input.active}}",
				"then_steps": [
					{ "type": "assign", "variable": "status", "value": "active" }
				],
				"else_steps": [
					{ "type": "assign", "variable": "status", "value": "inactive" }
				]
			},
			{
				"type": "switch",
				"field": "priority",
				"case_steps": {
					"high": [{ "type": "log", "message": "High priority" }],
					"default": [{ "type": "log", "message": "Normal" }]
				}
			},
			{
				"type": "loop",
				"over": "{{items}}",
				"steps": [
					{ "type": "emit", "event": "item.processed" }
				]
			}
		]
	}`)

	proc, err := parser.ParseProcess(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc.Name != "nested_test" {
		t.Errorf("expected nested_test, got %s", proc.Name)
	}
	if len(proc.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(proc.Steps))
	}
	if len(proc.Steps[0].ThenSteps) != 1 {
		t.Errorf("expected 1 then_step, got %d", len(proc.Steps[0].ThenSteps))
	}
	if len(proc.Steps[0].ElseSteps) != 1 {
		t.Errorf("expected 1 else_step, got %d", len(proc.Steps[0].ElseSteps))
	}
	if len(proc.Steps[1].CaseSteps) != 2 {
		t.Errorf("expected 2 case_steps, got %d", len(proc.Steps[1].CaseSteps))
	}
	if len(proc.Steps[2].Steps) != 1 {
		t.Errorf("expected 1 loop sub-step, got %d", len(proc.Steps[2].Steps))
	}
}

type mockTranslator struct {
	translations map[string]string
}

func (m *mockTranslator) Translate(locale string, key string) string {
	if val, ok := m.translations[locale+":"+key]; ok {
		return val
	}
	return key
}

func TestInterpolateTranslation(t *testing.T) {
	tr := &mockTranslator{
		translations: map[string]string{
			"id:Hello World": "Halo Dunia",
		},
	}
	execCtx := &executor.Context{
		Input:      map[string]any{},
		Variables:  map[string]any{},
		Locale:     "id",
		Translator: tr,
	}

	h := &AssignHandler{}
	step := parser.StepDefinition{
		Type:     parser.StepAssign,
		Variable: "greeting",
		Value:    "test",
	}
	if err := h.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := execCtx.T("Hello World")
	if result != "Halo Dunia" {
		t.Errorf("expected 'Halo Dunia', got '%s'", result)
	}
}

func TestContextT_NoTranslator(t *testing.T) {
	execCtx := &executor.Context{
		Input:     map[string]any{},
		Variables: map[string]any{},
	}
	result := execCtx.T("Hello")
	if result != "Hello" {
		t.Errorf("expected 'Hello' fallback, got '%s'", result)
	}
}

func TestContextT_NoLocale(t *testing.T) {
	tr := &mockTranslator{translations: map[string]string{"id:Hello": "Halo"}}
	execCtx := &executor.Context{
		Input:      map[string]any{},
		Variables:  map[string]any{},
		Translator: tr,
	}
	result := execCtx.T("Hello")
	if result != "Hello" {
		t.Errorf("expected 'Hello' fallback (no locale), got '%s'", result)
	}
}

func TestValidateHandler_TranslatedError(t *testing.T) {
	tr := &mockTranslator{
		translations: map[string]string{
			"id:Only draft orders can be confirmed": "Hanya pesanan draf yang bisa dikonfirmasi",
		},
	}
	h := &ValidateHandler{}
	execCtx := &executor.Context{
		Input:      map[string]any{"status": "confirmed"},
		Variables:  map[string]any{},
		Locale:     "id",
		Translator: tr,
	}
	step := parser.StepDefinition{
		Type:  parser.StepValidate,
		Rules: map[string]map[string]any{"status": {"eq": "draft"}},
		Error: "Only draft orders can be confirmed",
	}

	err := h.Execute(context.Background(), execCtx, step)
	if err == nil {
		t.Fatal("should fail validation")
	}
	if err.Error() != "Hanya pesanan draf yang bisa dikonfirmasi" {
		t.Errorf("expected translated error, got: %s", err.Error())
	}
}

func TestValidateHandler_ErrorWithTFunction(t *testing.T) {
	tr := &mockTranslator{
		translations: map[string]string{
			"id:process.error.draft_only": "Hanya pesanan draf yang bisa dikonfirmasi",
		},
	}
	h := &ValidateHandler{}
	execCtx := &executor.Context{
		Input:      map[string]any{"status": "confirmed"},
		Variables:  map[string]any{},
		Locale:     "id",
		Translator: tr,
	}
	step := parser.StepDefinition{
		Type:  parser.StepValidate,
		Rules: map[string]map[string]any{"status": {"eq": "draft"}},
		Error: "{{t('process.error.draft_only')}}",
	}

	err := h.Execute(context.Background(), execCtx, step)
	if err == nil {
		t.Fatal("should fail validation")
	}
	if err.Error() != "Hanya pesanan draf yang bisa dikonfirmasi" {
		t.Errorf("expected translated error via t(), got: %s", err.Error())
	}
}

func TestValidateHandler_NoTranslator_Passthrough(t *testing.T) {
	h := &ValidateHandler{}
	execCtx := &executor.Context{
		Input:     map[string]any{"status": "confirmed"},
		Variables: map[string]any{},
	}
	step := parser.StepDefinition{
		Type:  parser.StepValidate,
		Rules: map[string]map[string]any{"status": {"eq": "draft"}},
		Error: "Only draft orders can be confirmed",
	}

	err := h.Execute(context.Background(), execCtx, step)
	if err == nil {
		t.Fatal("should fail validation")
	}
	if err.Error() != "Only draft orders can be confirmed" {
		t.Errorf("expected original error (no translator), got: %s", err.Error())
	}
}

func TestParseProcess_WithLabels(t *testing.T) {
	data := []byte(`{
		"name": "labeled_process",
		"steps": [
			{ "type": "validate", "label": "Check Status", "rules": { "status": { "eq": "draft" } }, "error": "Must be draft" },
			{ "type": "update", "label": "Update Record", "model": "order", "set": { "status": "confirmed" } }
		]
	}`)

	proc, err := parser.ParseProcess(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc.Steps[0].Label != "Check Status" {
		t.Errorf("expected label 'Check Status', got '%s'", proc.Steps[0].Label)
	}
	if proc.Steps[1].Label != "Update Record" {
		t.Errorf("expected label 'Update Record', got '%s'", proc.Steps[1].Label)
	}
}

func TestInterpolateTranslation_TFunction(t *testing.T) {
	tr := &mockTranslator{
		translations: map[string]string{
			"id:greeting":  "Selamat datang",
			"id:farewell":  "Sampai jumpa",
		},
	}
	execCtx := &executor.Context{
		Input:      map[string]any{"name": "Budi"},
		Variables:  map[string]any{},
		Locale:     "id",
		Translator: tr,
	}

	logH := &LogHandler{}
	step := parser.StepDefinition{
		Type:    parser.StepLog,
		Message: "{{t('greeting')}}, {{input.name}}! {{t('farewell')}}.",
	}
	if err := logH.Execute(context.Background(), execCtx, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
