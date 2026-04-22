# Fix Process Engine: Nested Steps for Branching & Looping

**Date**: 2026-04-22
**Status**: Approved
**Scope**: Phase 1 — fix broken control flow; Phase 2 — DAG migration (separate doc)

## Problem

The process engine has 3 control flow step types (`if`, `switch`, `loop`) that are partially broken:

1. **`if`/`switch`**: Set `_goto` variable but executor has no jump logic — runs all steps sequentially regardless
2. **`loop`**: Sets `_index`/`_item` variables but never executes nested sub-steps
3. **`call`**: Works correctly (sub-process execution with context merging)

## Solution: Nested Steps Model

Replace label-based goto with nested step arrays. Control flow steps contain their own sub-steps that execute inline.

### Parser Changes (`internal/compiler/parser/process.go`)

Add to `StepDefinition`:

```go
// if — nested branches
ThenSteps []StepDefinition `json:"then_steps,omitempty"`
ElseSteps []StepDefinition `json:"else_steps,omitempty"`

// switch — nested case branches
CaseSteps map[string][]StepDefinition `json:"case_steps,omitempty"`
```

`loop` already has `Steps []StepDefinition` — no parser change needed.

Keep old `Then`/`Else` string fields for backward compat (ignored when `ThenSteps`/`ElseSteps` present).

### Executor Changes (`internal/runtime/executor/executor.go`)

Extract reusable step runner:

```go
func (e *Executor) ExecuteSteps(ctx context.Context, execCtx *Context, steps []StepDefinition) error
```

`Execute()` delegates to `ExecuteSteps()`. All nested handlers call `ExecuteSteps()` recursively.

Add max recursion depth (10) to prevent infinite nesting.

### Handler Changes (`internal/runtime/executor/steps/control.go`)

**IfHandler**: Evaluate condition, call `executor.ExecuteSteps(then_steps)` or `executor.ExecuteSteps(else_steps)`.

**SwitchHandler**: Resolve field value, call `executor.ExecuteSteps(case_steps[value])` or `case_steps["default"]`.

**LoopHandler**: Iterate over list, for each item set `_index`/`_item` then call `executor.ExecuteSteps(step.Steps)`.

All three handlers need an `Executor` reference (LoopHandler already has one).

### JSON Format

**if with nested steps:**
```json
{
  "type": "if",
  "condition": "{{input.expected_revenue > 5000}}",
  "then_steps": [
    { "type": "assign", "variable": "deal_type", "value": "enterprise" },
    { "type": "emit", "event": "lead.high_value" }
  ],
  "else_steps": [
    { "type": "assign", "variable": "deal_type", "value": "standard" }
  ]
}
```

**switch with case_steps:**
```json
{
  "type": "switch",
  "field": "status",
  "case_steps": {
    "draft": [
      { "type": "update", "model": "order", "set": { "status": "confirmed" } }
    ],
    "default": [
      { "type": "log", "message": "Unknown status" }
    ]
  }
}
```

**loop with sub-steps (already supported in parser):**
```json
{
  "type": "loop",
  "over": "{{order_lines}}",
  "steps": [
    { "type": "update", "model": "stock", "set": { "quantity": "{{_item.qty}}" } }
  ]
}
```

### Sample Updates

Update `samples/erp/modules/crm/processes/convert_lead.json` and create additional samples demonstrating nested branching, loops, and call-inside-branch patterns.

### Tests

- Nested if: both branches execute correct sub-steps
- Nested switch: correct case branch executes
- Loop: sub-steps execute for each item with correct `_index`/`_item`
- Recursion depth limit (>10 levels returns error)
- Backward compat: old `then`/`else` string format still parses without error
- Call inside nested branch works

## Phase 2: DAG-based Engine (Separate Design)

After Phase 1 is stable, redesign executor as a directed acyclic graph:
- Nodes with explicit connections (edges)
- Parallel fan-out / fan-in
- Merge nodes (wait for all branches)
- Visual representation support

This is a separate design document.
