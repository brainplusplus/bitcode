# Master Design: Runtime Engine Redesign + Module Setting

**Date**: 14 July 2026
**Status**: Approved
**Scope**: Script runtime architecture, bridge API, multi-runtime support, admin panel migration

> **Deprecated**: The earlier document `2026-07-14-admin-to-module-design.md` (archived in `archived/`) is superseded by this master document and its phase documents. That document only covered admin panel migration — this redesign is far broader, covering the entire script runtime architecture.

---

## Table of Contents

1. [Context](#1-context)
2. [Problems with Current Architecture](#2-problems-with-current-architecture)
3. [Vision](#3-vision)
4. [Architecture Overview](#4-architecture-overview)
5. [Phase Overview](#5-phase-overview)
6. [Naming Convention](#6-naming-convention)
7. [Phase Documents](#7-phase-documents)

---

## 1. Context

BitCode is a **general purpose low-code engine** — not just an ERP framework. The scripting system must handle diverse use cases: business logic, crawling with headless browsers, proxy manipulation, concurrency, ML/data science, document conversion, and more.

The current script runtime (child process Node.js/Python with stub bridge) is a prototype that cannot support these requirements. This redesign introduces a multi-runtime architecture with a unified bridge API.

---

## 2. Problems with Current Architecture

### 2.1 Script Runtime

| Problem | Detail |
|---------|--------|
| **Bridge API is stub** | `ctx.db.query` returns `{ rows: [] }`, `ctx.http.get` returns `{ status: 200, data: {} }` — nothing actually works |
| **No package management** | No `package.json`, no `node_modules`, no `requirements.txt`, no venv — anywhere |
| **No dependency isolation** | All modules share one Node.js process and one Python process |
| **Require resolution broken** | `require()` resolves from `plugins/typescript/`, not from the module's directory |
| **External dependency** | Requires Node.js and Python installed on server |
| **No env/config access** | Scripts cannot read environment variables, session data, or module config |

### 2.2 Admin Panel

| Problem | Detail |
|---------|--------|
| **Hardcoded Go** | ~2645 lines of Go rendering HTML via `strings.Builder` |
| **Ignores engine** | Base module has `group_list.json`, `group_form.json` etc. but admin.go re-implements everything |
| **Not customizable** | Users cannot override or extend admin pages |
| **Contradicts philosophy** | "JSON is the source code" — except for the admin panel |

---

## 3. Vision

### 3.1 Multi-Runtime Engine

Five script runtimes across two categories, with unified pool management:

```
┌─────────── Embedded (zero dependency, single binary) ───────────┐
│                                                                  │
│  goja (JavaScript ES6+)         quickjs (JavaScript ES2023)      │
│  • Pure Go                      • Pure Go (via Wazero WASM)      │
│  • Lightest, fastest Go interop • async/await, modules, BigInt   │
│  • For: validation, onchange,   • For: complex business logic,   │
│    computed fields, hot-path      modern JS syntax                │
│                                                                  │
│  yaegi (Go)                                                      │
│  • Pure Go, full Go spec                                         │
│  • Goroutine support, os/exec via whitelist                      │
│  • For: concurrency, custom bridges, system integration          │
│                                                                  │
├─────────── External (optional, full ecosystem) ─────────────────┤
│                                                                  │
│  Node.js (child process)        Python (child process)           │
│  • Full npm ecosystem           • Full pip ecosystem             │
│  • Per-module package.json      • Per-module requirements.txt    │
│  • For: crawling, headless      • For: ML, data science,         │
│    browser, proxy, heavy IO       scraping, automation           │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 3.1b Runtime Selection Format

```
runtime: "javascript"              → default engine (goja, configurable)
runtime: "javascript:goja"         → explicit goja
runtime: "javascript:quickjs"      → explicit quickjs
runtime: "go"                      → yaegi
runtime: "node"                    → Node.js child process
runtime: "python"                  → Python child process
```

### 3.1c Unified Pool Config

All runtimes share the same pool configuration. Developer thinks "is this fast or slow work?" — not per-language tuning.

```yaml
# bitcode.yaml
runtime:
  worker:                          # fast scripts — ALL runtimes
    pool_size: 4
    default_step_timeout: "30s"
    max_step_timeout: "5m"
    max_process_timeout: "10m"
    max_executions: 1000
    max_memory_mb: 0               # 0 = unlimited (default). Overridable by module/process.
    hard_max_memory_mb: 0          # 0 = no ceiling (default). NOT overridable. max_memory_mb cannot exceed this.

  background:                      # long-running scripts — ALL runtimes
    pool_size: 2
    default_step_timeout: "5m"
    max_step_timeout: "0"          # unlimited
    max_process_timeout: "0"       # unlimited
    max_executions: 100
    max_memory_mb: 0               # 0 = unlimited (default)
    hard_max_memory_mb: 0          # 0 = no ceiling (default)

  # Runtime-specific (only what's truly unique per runtime)
  javascript:
    default_engine: "goja"         # "goja" | "quickjs"
  node:
    enabled: "auto"
    command: "node"
    min_version: "20.0"
  python:
    enabled: "auto"
    command: "python3"
    min_version: "3.10.0"
  go:
    default_engine: "yaegi"
```

### 3.1d Resolution Hierarchy

All overridable configs follow the same resolution order: **step → process → module → project → hardcoded**.

| Config | Step | Process | Module | Project | Hard Limit | Hardcoded |
|--------|------|---------|--------|---------|------------|-----------|
| `pool` | ✅ | ✅ | ✅ `default_pool` | — | — | `"worker"` |
| step timeout | ✅ `timeout` | ✅ `default_step_timeout` | ✅ `default_step_timeout` | ✅ per pool | ✅ `max_step_timeout` | `30s` |
| process timeout | — | ✅ `timeout` | — | ✅ per pool | ✅ `max_process_timeout` | `5m` |
| runtime engine | ✅ `runtime` | ✅ `runtime` | ✅ `runtime_defaults` | ✅ `javascript.default_engine` | — | `"goja"` |
| max_memory_mb | — | ✅ | ✅ | ✅ per pool | ✅ `hard_max_memory_mb` | `0` (unlimited) |

Admin-only configs (NOT overridable by developer):

| Config | Where | Why admin-only |
|--------|-------|----------------|
| `pool_size` | `bitcode.yaml` | Infrastructure capacity |
| `max_executions` | `bitcode.yaml` | Process recycling tuning |
| `hard_max_memory_mb` | `bitcode.yaml` | Absolute memory ceiling. `max_memory_mb` cannot exceed this. 0 = no ceiling. |
| `max_step_timeout` | `bitcode.yaml` | Absolute timeout ceiling |
| `max_process_timeout` | `bitcode.yaml` | Absolute timeout ceiling |
| `node.enabled/command` | `bitcode.yaml` | Runtime availability |
| `python.enabled/command` | `bitcode.yaml` | Runtime availability |

### 3.2 Unified Bridge API

All runtimes share identical API surface — `bitcode.*`:

```
bitcode.model(name)           → CRUD operations (search, create, write, delete)
bitcode.db.query(sql, params) → Raw SQL
bitcode.http.get/post/put/del → HTTP client
bitcode.cache.get/set/del     → Redis/memory cache
bitcode.fs.read/write         → Sandboxed filesystem
bitcode.exec(cmd, ...args)    → Whitelisted os/exec
bitcode.emit(event, data)     → EventBus
bitcode.call(process, input)  → Execute another process
bitcode.log(level, message)   → Structured logging
bitcode.env(key)              → Environment variables (whitelist/blacklist)
bitcode.session               → Current user context (userId, tenantId, groups)
bitcode.config(key)           → Module settings
```

### 3.3 Security Rules

Environment access controlled per module in `module.json`:

```json
{
  "env_allow": ["STRIPE_API_KEY", "SMTP_*", "CRM_*"],
  "env_deny": ["DB_PASSWORD", "JWT_SECRET"],
  "exec_allow": ["pandoc", "wkhtmltopdf", "ffmpeg"],
  "exec_deny": ["rm", "shutdown"]
}
```

### 3.4 Per-Module Dependency Isolation

```
modules/crm/
├── package.json        ← npm dependencies (Node.js scripts)
├── node_modules/       ← isolated
├── requirements.txt    ← pip dependencies (Python scripts)
├── .venv/              ← isolated
├── go.mod              ← Go dependencies (Go scripts + bridges)
├── bridges/            ← custom bridges for this module
└── scripts/            ← scripts in any runtime
```

### 3.5 Custom Bridges

```
project/
├── bridges/                    ← project-level (all modules)
│   ├── whatsapp.go
│   └── telegram.go
└── modules/crm/
    └── bridges/                ← module-level (crm only)
        └── hubspot.go
```

### 3.6 Module "setting"

Admin panel rebuilt as a JSON module using all 4 runtimes — the ultimate stress test:

- goja scripts for lightweight dashboard stats
- yaegi scripts for concurrent security sync with goroutines
- Node.js scripts for Excel/PDF export with npm packages
- Python scripts for audit analytics with pandas
- Custom bridge (`engine_meta.go`) to expose engine internals

---

## 4. Architecture Overview

### 4.1 Runtime Selection in Process JSON

```json
{ "type": "script", "runtime": "javascript", "script": "scripts/validate.js" }
{ "type": "script", "runtime": "go",         "script": "scripts/sync.go" }
{ "type": "script", "runtime": "node",       "script": "scripts/crawl.js" }
{ "type": "script", "runtime": "python",     "script": "scripts/analyze.py" }
```

| Runtime value | Engine | Embedded? | Ecosystem |
|---------------|--------|-----------|-----------|
| `javascript` | goja | Yes | Bridge only |
| `go` | yaegi | Yes | Go stdlib + go.mod |
| `node` | Node.js child process | No | Full npm |
| `python` | Python child process | No | Full pip |

### 4.2 Bridge Implementation Per Runtime

| Runtime | How bridge is exposed |
|---------|----------------------|
| goja | `vm.Set("bitcode", bridgeObject)` — Go object injected into JS VM |
| yaegi | `i.Use(exports)` — `bitcode` as importable Go package |
| Node.js | JSON-RPC over stdin/stdout — `bitcode.model()` sends request to Go, waits for response |
| Python | JSON-RPC over stdin/stdout — same protocol as Node.js |

### 4.3 Graceful Degradation

```
Server without Node.js/Python:
  → goja + yaegi work ✅
  → Node.js/Python scripts → clear error: "Node.js not installed"
  → 90%+ functionality intact

Server with Node.js only:
  → goja + yaegi + Node.js work ✅
  → Python scripts → clear error

Server with everything:
  → All 4 runtimes work ✅
```

### 4.4 Cross-Runtime Execution

**Each step in a process can use a different runtime.** The engine routes each step to the correct runtime pool. Data flows between steps via JSON — universal across all runtimes.

#### 4.4.1 Mixed Runtime in One Process

```json
{
  "name": "crawl_analyze_save",
  "pool": "background",
  "timeout": "2h",
  "steps": [
    {
      "type": "script",
      "name": "crawl_pages",
      "runtime": "node",
      "script": "scripts/crawl.js",
      "timeout": "90m"
    },
    {
      "type": "script",
      "name": "analyze_with_ml",
      "runtime": "python",
      "script": "scripts/analyze.py",
      "timeout": "10m"
    },
    {
      "type": "script",
      "name": "validate_fast",
      "runtime": "javascript",
      "script": "scripts/validate.js",
      "timeout": "5s"
    },
    {
      "type": "script",
      "name": "parallel_save",
      "runtime": "go",
      "script": "scripts/save.go",
      "timeout": "5m"
    }
  ]
}
```

This process uses **all 4 runtimes** in sequence: Node.js → Python → goja → yaegi. Each step picks the runtime best suited for the task.

#### 4.4.2 How Data Flows Between Runtimes

```
┌──────────────────────────────────────────────────────────┐
│ Process: crawl_analyze_save                              │
│                                                          │
│  Step 0: crawl.js (Node.js)                              │
│    Input:  { url: "https://example.com", pages: 100 }    │
│    Output: { pages: [{title, body, url}, ...] }          │
│    → stored in execCtx.Variables["crawl_pages"]          │
│                                                          │
│  Step 1: analyze.py (Python)                             │
│    Input:  params.variables.crawl_pages                  │
│    → Python receives the crawled pages as JSON           │
│    Output: { scores: [{url, score, category}, ...] }     │
│    → stored in execCtx.Variables["analyze_with_ml"]      │
│                                                          │
│  Step 2: validate.js (goja — embedded, instant)          │
│    Input:  params.variables.analyze_with_ml               │
│    → goja receives ML scores as JSON                     │
│    Output: { valid: [...], invalid: [...] }               │
│    → stored in execCtx.Variables["validate_fast"]        │
│                                                          │
│  Step 3: save.go (yaegi — embedded, concurrent)          │
│    Input:  params.variables.validate_fast                 │
│    → yaegi receives validated data as Go map              │
│    → Uses goroutines to save in parallel                 │
│    Output: { saved: 95, errors: 5 }                      │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

**Key point**: Data is serialized to JSON between steps. All runtimes speak JSON. No special conversion needed.

The `into` field controls where step output is stored:

```json
{ "type": "script", "runtime": "node", "script": "crawl.js", "into": "crawl_result" }
```
→ Output stored in `execCtx.Variables["crawl_result"]`
→ Next step accesses via `params.variables.crawl_result`

If `into` is not set, output is stored with the step name as key.

#### 4.4.3 Cross-Runtime via `bitcode.call()`

A script in one runtime can call a process that uses a different runtime:

```javascript
// scripts/orchestrate.js (Node.js)
export default {
  async execute(bitcode, params) {
    // This process internally uses Python for ML scoring
    const mlResult = await bitcode.call("ml_scoring", { leads: params.leads });
    // mlResult came from Python, but we use it here in Node.js
    
    // This process internally uses Go for parallel processing
    const saveResult = await bitcode.call("parallel_save", { data: mlResult.scores });
    // saveResult came from yaegi, transparent to us
    
    return { processed: saveResult.count };
  }
};
```

```json
// processes/ml_scoring.json
{
  "name": "ml_scoring",
  "steps": [
    { "type": "script", "runtime": "python", "script": "scripts/score.py" }
  ]
}
```

The calling script doesn't know (or care) what runtime the called process uses. `bitcode.call()` is runtime-agnostic.

#### 4.4.4 When to Use Which Runtime

| Task | Best Runtime | Why |
|------|-------------|-----|
| Field validation, computed fields | `javascript` (goja) | Instant, embedded, no overhead |
| Business logic, CRUD workflows | `javascript` (goja) | Fast, sandboxed, always available |
| Concurrent processing, batch ops | `go` (yaegi) | Goroutines, type safety |
| Custom bridges, system integration | `go` (yaegi) | Full Go stdlib access |
| Crawling, headless browser, proxy | `node` (Node.js) | Puppeteer, Playwright, npm ecosystem |
| npm-dependent tasks (Excel, PDF) | `node` (Node.js) | exceljs, pdfkit, etc. |
| ML/AI, data science | `python` (Python) | pandas, scikit-learn, torch |
| Data analysis, statistics | `python` (Python) | numpy, scipy, matplotlib |

**Rule of thumb**: Start with `javascript` (goja). If you need npm packages → `node`. If you need pip packages → `python`. If you need concurrency/performance → `go`.

#### 4.4.5 Pool Selection for Mixed-Runtime Processes

When a process has steps with different runtimes, the `pool` field applies to the **process scheduling**, not individual steps:

```json
{
  "name": "mixed_pipeline",
  "pool": "background",
  "steps": [
    { "type": "script", "runtime": "node", "script": "crawl.js" },
    { "type": "script", "runtime": "python", "script": "analyze.py" }
  ]
}
```

- `pool: "background"` means: use **background pool** for both Node.js and Python steps
- Step 0 → Node.js background pool process
- Step 1 → Python background pool process
- Each step gets a process from its runtime's pool, but the pool type (worker/background) is determined by the process-level `pool` field

If a step needs a different pool than the process default, it can override:

```json
{
  "name": "mixed_pipeline",
  "pool": "worker",
  "steps": [
    { "type": "script", "runtime": "node", "script": "quick_check.js" },
    { "type": "script", "runtime": "python", "script": "heavy_ml.py", "pool": "background" }
  ]
}
```

Step 0 → Node.js **worker** pool (from process default).
Step 1 → Python **background** pool (step-level override).

#### 4.4.6 Edge Cases

| Edge Case | Behavior |
|-----------|----------|
| Step uses `runtime: "node"` but Node.js not installed | Error: `RUNTIME_NOT_AVAILABLE`. Other steps in the process don't execute (process fails). |
| Step uses `runtime: "python"` but Python not installed | Same — process fails at that step. Previous steps' results are preserved in execution log. |
| Step uses `runtime: "javascript"` (goja) | Always works — embedded, no external dependency. |
| Step uses `runtime: "go"` (yaegi) | Always works — embedded, no external dependency. |
| Data too large between steps (e.g., 500MB crawl result) | JSON serialization may be slow. Consider writing to `bitcode.storage` or `bitcode.cache` instead of passing via variables. |
| Step output is not JSON-serializable (e.g., Python object) | Script must return JSON-serializable data (dict, list, str, int, float, bool, None). Non-serializable types cause error. |
| Circular `bitcode.call()` (A calls B calls A) | Max nesting depth = 10 (existing engine limit). Error: "maximum nesting depth exceeded". |

---

## 5. Phase Overview

| Phase | Title | Depends On | Status | Deliverable |
|-------|-------|------------|--------|-------------|
| **6A** | Schema Compatibility | — (independent) | ✅ Done | 8 new field types, storage hints, plural table naming, display labels, auto-validators |
| **1** | Bridge API Design | — | ✅ Done | 20 namespace interfaces, tls-client HTTP, bulk ops, execution log, factory |
| **1.5** | Multi-Tenancy Architecture | Phase 1 | ✅ Done | shared_table strategy, auto tenant_id column, tenant_scoped per model, conditional filtering |
| **4** | Embedded Runtime: goja + quickjs | Phase 1 | 🔲 Next | `runtime: "javascript"` — single binary, no Node.js needed |
| **5** | Embedded Runtime: yaegi | Phase 1 | 🔲 Pending | `runtime: "go"` — goroutines, bridges/, go.mod, exec whitelist |
| **2** | Fix Node.js Child Process | Phase 1, 1.5 | 🔲 Pending | 6 TS scripts in samples/erp work with real bridge |
| **3** | Fix Python Child Process | Phase 1, 1.5 | 🔲 Pending | 6 PY scripts in samples/erp work with real bridge |
| **6B** | Polymorphic Relations | Phase 6A | 🔲 Pending | morph_to, morph_one, morph_many, morph_to_many, morph_by_many |
| **6C** | Engine Enhancements | Phase 6A, Phase 1 | 🔲 Pending | Array-backed models (Sushi-style), view modifiers, metadata API, eager loading fixes |
| **7** | Module "setting" | All phases (1-6C) | 🔲 Pending | Admin panel as JSON module, 4+ runtimes stress test, admin.go deprecation |

### Dependency Graph

```
Phase 1 (Bridge API Design)
  │
  ├──► Phase 1.5 (Multi-Tenancy) ──┐
  │                                 │
  ├──► Phase 2 (Node.js) ◄─────────┤
  ├──► Phase 3 (Python)  ◄─────────┤
  ├──► Phase 4 (goja)    ──────────┤
  └──► Phase 5 (yaegi)   ──────────┘
                                    │
Phase 6A (Schema Compat) ──────────┤ (independent, can start anytime)
                                    │
                                    ├──► Phase 6B (Polymorphic Relations)
                                    ├──► Phase 6C (Engine Enhancements)
                                    └──► Phase 7 (Module "setting")
```

Phase 1.5 must complete before Phase 2-3 (runtime implementations need correct tenant behavior).
Phase 4-5 can start after Phase 1 (embedded runtimes don't depend on tenant DB changes).
Phase 6A is independent — parser/migration level, no runtime dependency.
Phase 6B depends on Phase 6A (new field types needed for morph columns).
Phase 6C depends on Phase 6A (display_field, title_field format, etc.).
Phase 7 needs all phases complete — it uses all 4+ runtimes as stress test.

---

## 6. Naming Convention

- Bridge API namespace: **`bitcode`** (not `erp` — this is a general purpose engine)
- Admin module name: **`setting`**
- Runtime values: `javascript`, `go`, `node`, `python`
- Design docs: `2026-07-14-runtime-engine-phase-{N}-{topic}.md`

---

## 7. Phase Documents

| Phase | Document |
|-------|----------|
| Master (this) | `2026-07-14-runtime-engine-redesign-master.md` |
| Phase 1 | `2026-07-14-runtime-engine-phase-1-bridge-api-design.md` |
| Phase 1.5 | `2026-07-14-runtime-engine-phase-1.5-multi-tenancy.md` |
| Phase 2 | `2026-07-14-runtime-engine-phase-2-fix-nodejs.md` |
| Phase 3 | `2026-07-14-runtime-engine-phase-3-fix-python.md` |
| Phase 4 | `2026-07-14-runtime-engine-phase-4-embedded-js.md` |
| Phase 5 | `2026-07-14-runtime-engine-phase-5-yaegi.md` |
| Phase 6A | `2026-07-14-runtime-engine-phase-6a-schema-compatibility.md` |
| Phase 6B | `2026-07-14-runtime-engine-phase-6b-polymorphic-relations.md` |
| Phase 6C | `2026-07-14-runtime-engine-phase-6c-engine-enhancements.md` |
| Phase 7 | `2026-07-14-runtime-engine-phase-7-module-setting.md` |
| Archived | `archived/2026-07-14-admin-to-module-design.md` (deprecated, superseded by this redesign) |
