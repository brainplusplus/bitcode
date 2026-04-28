# Handoff Context — Runtime Engine Redesign

**Date**: 28 July 2026
**Branch**: `master`
**Last commit**: `3288b38` feat(engine): wire Phase 4 embedded JS into executor pipeline

---

## WAJIB DIBACA SEBELUM MULAI

**WAJIB berpikir kritis, detail, mateng, lengkap dan jujur** — dalam bikin plan, design, implementation, execution, dan diskusi. Ini bukan saran, ini WAJIB. Jangan skip, jangan shortcut.

**Moto user: simplicity, flexible, dan powerfull.**

Artinya:
- Jangan over-engineer. Kalau bisa simple, buat simple.
- Jangan under-engineer. Kalau butuh complex, buat complex tapi tetap clean.
- Jangan bohong tentang status. Kalau belum selesai, bilang belum selesai. Kalau ada bug, bilang ada bug.
- Jangan asal mindahkan/hapus file yang bukan milik task kamu — ada task lain yang sedang dikerjakan secara paralel.
- Selalu verifikasi compile + test sebelum commit.
- Update docs per phase (bukan setelah semua selesai).
- Sebelum coding, BACA design doc + implementation plan dulu. Jangan langsung coding.
- Setelah coding, EVALUASI kritis: apa yang kurang? apa yang salah? apa tech debt?

---

## USER REQUESTS (AS-IS)

- "kenapa halaman admin itu pake admin.go? kan enginenya sudah ready dengan json, process, maupun script?"
- "moto saya, simplicity, flexible, dan powerfull"
- "WAJIB berpikir kritis, detail, mateng, lengkap dan jujur"
- "bridge api pake bitcode ya bukan erp, kan general purpose complex solution"
- "nanti bikin design doc maupun implementation doc kalau bisa per phase, jadi biar bisa detail, lengkap, jelas"
- "mongodb jangan lupakan"
- "morphs (polymorphic relations) lakukan juga ya, saya butuh banget ini"
- "SHOULD (nice-to-have) itu eksekusi juga ya"
- "max_memory_mb dan hard_max_memory_mb itu defaultnya 0 (unlimited) ya, tapi tidak boleh max_memory_mb > hard_max_memory_mb"
- "jangan di remove, mending bikin aja folder tambahan namanya archieved"
- "bikin implementation plan untuk setiap phase (1 phase 1 implementation plan), jika sudah baru eksekusi"
- "untuk update dokumentasi-dokumentasi terkait apa yang sudah dikerjakan itu saat selesai per phase"
- "jangan asal mindahin broken file untrack yak, soalnya lagi ada yg ngerjain task lain"

---

## GOAL

Continue implementing the BitCode low-code engine runtime redesign. 4 phases done, 6 remaining. Next: Phase 5 (yaegi).

---

## COMPLETED PHASES

| Phase | Commit | Summary |
|-------|--------|---------|
| **6A** | `eb6a06a` | 8 new field types (uuid, ip, ipv6, year, vector, binary, json:object, json:array), storage hints, NumberFormatConfig, plural table naming via jinzhu/inflection, display labels, auto-validators, duplicate model detection. 17 new tests. |
| **1** | `94046af` | Bridge API — 26 files in `engine/internal/runtime/bridge/`. 20 namespace interfaces (ModelHandle, SudoModelHandle, DB, HTTPClient, Cache, FS, EnvReader, ConfigReader, EventEmitter, ProcessCaller, CommandExecutor, Logger, EmailSender, Notifier, Storage, I18N, SecurityChecker, AuditLogger, Crypto, ExecutionLog). Factory wiring. tls-client HTTP. BulkUpdate/BulkDelete/BulkUpsert added to GenericRepository + MongoRepository. process_execution + process_execution_step JSON models. 27 new tests. |
| **1.5** | `90945cb` | Multi-Tenancy — `TenantScoped *bool` + `IsTenantScoped()` on ModelDefinition (default true). MigrateModel auto-adds tenant_id column + index. ALTER TABLE for existing tables. Conditional tenant filter in repository (only for tenant-scoped models). TenantConfig expanded with Isolation + Column. ValidateTenantConfig at startup. 8 new tests. |
| **4** | `6861a47` + `3288b38` | Embedded JS — goja (ES6+, pure Go) + QuickJS (ES2023, Wazero WASM). Shared EmbeddedRuntime + VM interfaces. ExecuteEmbedded() with timeout/panic recovery/context cancel. EngineRegistry with resolution. goja proxy maps all 20 bridge namespaces as Go functions. QuickJS uses host functions (__bc_*) + JS wrapper (bitcode_init.go). Compilation cache with mtime invalidation. Script signature detection (object.execute, direct function, expression). EmbeddedScriptRunner adapter wired into executor pipeline. .js defaults to goja. 21 new tests. |

---

## PENDING PHASES (in execution order)

```
5   yaegi — Go scripting (estimated 6-8 days)
 ↓
2   Fix Node.js — child process + real bridge (estimated 8-10 days)
 ↓
3   Fix Python — child process + real bridge (estimated 6-8 days)
 ↓
6B  Polymorphic Relations — morph_to, morph_one, morph_many, morph_to_many, morph_by_many (estimated 7-9 days)
 ↓
6C  Engine Enhancements + Array Models (estimated 10-14 days)
 ↓
7   Module Setting — admin panel migration (estimated 10-14 days)
```

---

## NEXT: PHASE 5 (yaegi) — WHAT YOU NEED TO KNOW

### Design Doc
`docs/plans/2026-07-14-runtime-engine-phase-5-yaegi.md` (892 lines, very detailed)

### Implementation Plan
`docs/plans/impl/phase-5-implementation.md` (69 lines, 5 streams)

### Key Differences from Phase 4
| Aspect | Phase 4 (goja/qjs) | Phase 5 (yaegi) |
|--------|-------------------|-----------------|
| Language | JavaScript | Go |
| Bridge injection | Go maps/host functions | `interp.Exports` as Go package symbols |
| Script imports | `bitcode.*` global | `import "bitcode"` then `bitcode.Model()` |
| Concurrency | Single-threaded | Real goroutines |
| Timeout | `vm.Interrupt()` / `rt.Close()` | `context.Context` cooperative (no force kill) |
| Custom extensions | Not supported | `bridges/` folder with .go files |
| External deps | Not supported | Per-module `go.mod` |
| Stdlib | Sandboxed (no OS access) | Filtered (block unsafe/syscall/os.exec, allow os.ReadFile) |

### What to Reuse from Phase 4
- `embedded.EmbeddedRuntime` + `embedded.VM` interfaces — yaegi implements these
- `embedded.ExecuteEmbedded()` — shared executor with timeout/panic recovery
- `embedded.EngineRegistry` — register "yaegi" alongside "goja" and "quickjs"
- `embedded.ScriptRunnerConfig` — same adapter pattern
- `embedded/bridge_helper.go` — same conversion helpers (ParseSearchOpts etc.)
- `steps/script.go` — already routes based on runtime, just add "go" detection

### What to Create New
```
engine/internal/runtime/embedded/yaegi/
├── runtime.go          # YaegiRuntime implements EmbeddedRuntime
├── vm.go               # YaegiVM implements VM (InjectBridge via interp.Exports)
├── symbols.go          # Bridge symbols: bitcode package as reflect.Value map
├── stdlib_filter.go    # Remove unsafe, syscall, os/exec from stdlib
├── bridge_loader.go    # Load bridges/ folder .go files as yaegi symbols
└── yaegi_test.go       # Tests
```

### Dependencies to Add
```
go get github.com/traefik/yaegi@latest
```

### Integration Points (in app.go)
Line ~153: `embeddedReg.Register("yaegi", yaegiRT.New())` — add alongside goja + quickjs
Line ~462: `detectRuntimeFromExtension` in script.go already handles `.go` → `"go"`

### Tricky Parts (from design doc)
1. **Timeout is cooperative** — yaegi has no `Interrupt()`. Scripts must check `ctx.Done()`. Infinite loop without ctx check = goroutine leak. This is a documented limitation.
2. **Bridge injection as Go package** — yaegi uses `interp.Exports` map with `reflect.ValueOf()`. Each bridge method must be wrapped as `reflect.Value`. This is different from goja (direct Go functions) and qjs (host functions).
3. **bridges/ folder** — project-level `bridges/*.go` and module-level `modules/crm/bridges/*.go` are loaded as additional yaegi symbols. This extends `bitcode.*` namespace.
4. **go.mod per module** — modules can have their own `go.mod` for external Go packages. `yaegi extract` pipeline converts Go packages to yaegi-compatible symbols.
5. **Stdlib filter** — must explicitly remove `os/exec`, `unsafe`, `syscall` from `stdlib.Symbols` before passing to `interp.Use()`.

---

## CURRENT STATE — BUILD & TEST

### How to Build (safe, excludes broken untracked files)
```bash
cd engine
go build ./internal/...
```

### How to Test (safe)
```bash
cd engine
go test ./internal/runtime/... ./internal/compiler/... ./internal/domain/... ./internal/infrastructure/... ./pkg/... ./embedded/...
```
Result: **526 tests pass**, 31 packages, 0 failures.

### DO NOT run `go test ./...` or `go build ./...`
These will fail because of 3 **untracked files from another task** (offline mode):
- `engine/internal/infrastructure/persistence/offline_schema.go`
- `engine/internal/infrastructure/persistence/sync_schema.go`
- `engine/internal/presentation/api/sync_handler.go`

These reference `FieldDefinition.Name` which was changed in Phase 6A. **DO NOT touch, move, or delete these files** — another task owns them.

### Test Counts by Package
| Package | Tests |
|---------|-------|
| `runtime/bridge` | 27 |
| `runtime/embedded` | 10 |
| `runtime/embedded/goja` | 11 |
| `infrastructure/persistence` (tenant) | 8 |
| `infrastructure/persistence` (existing) | ~103 |
| Other packages | ~367 |

---

## KEY FILES — QUICK REFERENCE

### Design Docs & Plans
| File | Lines | Purpose |
|------|-------|---------|
| `docs/plans/2026-07-14-runtime-engine-redesign-master.md` | ~530 | Master doc, phase overview, dependency graph, status |
| `docs/plans/2026-07-14-runtime-engine-phase-5-yaegi.md` | 892 | Phase 5 design (READ THIS FIRST) |
| `docs/plans/impl/phase-5-implementation.md` | 69 | Phase 5 implementation plan (5 streams) |
| `docs/plans/impl/` | 10 files | All implementation plans |

### Bridge API (Phase 1) — `engine/internal/runtime/bridge/`
| File | Lines | Purpose |
|------|-------|---------|
| `interfaces.go` | 114 | 20 namespace interfaces (ModelHandle, DB, HTTPClient, etc.) |
| `context.go` | 67 | Context struct — single entry point for all runtimes |
| `factory.go` | 57 | Factory wiring all 20 bridges |
| `model.go` | 578 | ModelHandle + SudoModelHandle (CRUD, bulk, relations, sudo) |
| `types.go` | 173 | SearchOptions, HTTPOptions, Session, SecurityRules, etc. |
| `errors.go` | 84 | BridgeError type + 20 error codes |
| `http.go` | 157 | tls-client HTTP (fingerprinting, proxy, cookie jar) |
| `fs.go` | 122 | Sandboxed filesystem |
| `exec.go` | 72 | Command executor with whitelist |
| `bridge_test.go` | 360 | 27 tests |

### Embedded Runtimes (Phase 4) — `engine/internal/runtime/embedded/`
| File | Lines | Purpose |
|------|-------|---------|
| `runtime.go` | 21 | EmbeddedRuntime + VM interfaces |
| `executor.go` | 59 | ExecuteEmbedded() — timeout, panic recovery, context cancel |
| `registry.go` | 56 | EngineRegistry + Resolve() |
| `bridge_helper.go` | 201 | ParseSearchOpts, ParseHTTPOpts, ToInt, ToStringSlice, etc. |
| `script_runner.go` | 60 | EmbeddedScriptRunner adapter for executor pipeline |
| `goja/runtime.go` | 15 | GojaRuntime |
| `goja/vm.go` | 99 | GojaVM — InjectBridge, Execute, Interrupt, compilation cache |
| `goja/proxy.go` | 194 | All 20 bridge namespaces as Go function maps |
| `qjs/runtime.go` | 17 | QJSRuntime |
| `qjs/vm.go` | 70 | QJSVM — host functions + JS wrapper |
| `qjs/proxy.go` | 297 | Host function registration (__bc_* flat functions) |
| `qjs/bitcode_init.go` | 73 | JS wrapper creating bitcode.* API |

### App Wiring — `engine/internal/app.go`
| Line | What |
|------|------|
| ~34 | Import aliases: `jsrt`, `gojaRT`, `qjsRT` |
| ~118 | `EmbeddedRegistry *jsrt.EngineRegistry` field on App struct |
| ~150-152 | Initialize registry: `jsrt.NewRegistry()`, register goja + quickjs |
| ~249 | `EmbeddedRegistry: embeddedReg` in App struct literal |
| ~459-466 | Create `EmbeddedScriptRunner`, wire into `ScriptHandler` |

### Executor Pipeline — `engine/internal/runtime/executor/steps/script.go`
- `ScriptHandler` has `Runner` (child process) + `EmbeddedRunner` (embedded)
- `selectRunner()` checks `EmbeddedRunner.CanHandle(runtime)` first
- `detectRuntimeFromExtension()`: `.js`→javascript, `.go`→go, `.ts`→node, `.py`→python

---

## IMPORTANT DECISIONS (accumulated across sessions)

- Two-layer type system: semantic type (developer writes) + storage hint (optional override)
- No "numeric" or "money" type — use decimal/currency + storage:"numeric"
- ip = both IPv4/IPv6, ip:v4 = strict IPv4, ip:v6/ipv6 = strict IPv6
- title_field supports format templates via format engine ({data.code} - {data.name})
- search_field auto-extracts string/text fields from title_field format
- display_field on many2one overrides target model's title_field
- Table naming: project config table_naming:"plural" + per-model override via table.plural
- Junction tables (many2many) always singular, morph junction tables always plural
- Duplicate model in same module = startup ERROR, cross-module = OK, inheritance = OK
- 5 morph types: morph_to, morph_one, morph_many, morph_to_many, morph_by_many (not inverse:true)
- Currency resolution: field → record (currency_field) → session → project → hardcoded USD
- NumberFormat (not FieldMask) to avoid conflict with existing Mask bool (password masking)
- .js files default to embedded goja (NOT Node.js child process)
- runtime field in step JSON overrides auto-detection
- goja does NOT support async/await — design doc is correct
- qjs interrupt via rt.Close() (less graceful but acceptable per-execution VM)
- Compilation cache with mtime invalidation for goja (sync.Map)
- max_memory_mb default 0 (unlimited), tapi tidak boleh max_memory_mb > hard_max_memory_mb
- User (brainplus) is sole developer — estimates are for one person full-time
- User prefers Indonesian for discussion, English for code/docs

---

## KNOWN TECH DEBT (documented in docs/features.md)

| # | Item | Current State | Target |
|---|------|--------------|--------|
| 1 | Storage + AttachmentRepository | wraps StorageDriver only | Phase 1 iter 2 |
| 2 | Email template rendering | raw HTML only | Phase 1 iter 2 |
| 3 | Notify user targeting | broadcast to channel | Phase 1 iter 2 |
| 4 | Execution log cleanup cron | config exists, no goroutine | Phase 1 iter 2 |
| 5 | Execution retry | returns "not yet implemented" | Phase 1 iter 2 |
| 6 | Executor integration | not wrapped with auto execution log | Phase 1 iter 2 |
| 7 | BulkUpdate/BulkDelete hooks | skips before/after hooks | Phase 6B |
| 8 | QuickJS unit tests | only compile check, no VM execution tests | Phase 4 iter 2 |

---

## EXPLICIT CONSTRAINTS

- "bridge api pake bitcode ya bukan erp, kan general purpose complex solution"
- "max_memory_mb dan hard_max_memory_mb defaultnya 0 (unlimited), tapi tidak boleh max_memory_mb > hard_max_memory_mb"
- "jangan di remove, mending bikin aja folder tambahan namanya archieved"
- "mongodb jangan lupakan"
- "SHOULD (nice-to-have) itu eksekusi juga ya"
- "jangan asal mindahin broken file untrack, soalnya lagi ada yg ngerjain task lain"
- Update docs per phase (bukan setelah semua selesai)
- Docs yang perlu di-update per phase: master doc (status), architecture.md, features.md, codebase.md, feature-specific docs

---

## COMMIT HISTORY (this session)

```
3288b38 feat(engine): wire Phase 4 embedded JS into executor pipeline + update docs
6861a47 feat(engine): implement Phase 4 embedded JS runtimes — goja + QuickJS
71c7a95 docs: update architecture, features, codebase docs for Phase 6A, 1, 1.5 completion
90945cb feat(engine): implement Phase 1.5 Multi-Tenancy
94046af feat(engine): implement Phase 1 Bridge API
eb6a06a feat(engine): implement Phase 6A schema compatibility
```
