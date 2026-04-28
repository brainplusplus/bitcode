# Phase 2 Implementation Plan: Fix Node.js Child Process

**Estimated effort**: 8-10 days
**Prerequisites**: Phase 1 (bridge interfaces), Phase 1.5 (tenant behavior)
**Test command**: `go test ./internal/runtime/plugin/...` + manual test with sample scripts

---

## Implementation Order

```
Stream 1: Bidirectional JSON-RPC Protocol (Day 1-3)
  ↓
Stream 2: Process Pool (Worker + Background) (Day 3-5)
  ↓
Stream 3: Node.js Runtime Rewrite (Day 5-7)
  ↓
Stream 4: Bun Auto-Detection + TypeScript (Day 7-8)
  ↓
Stream 5: npm per-Module + Integration (Day 8-9)
  ↓
Stream 6: Tests (Day 9-10)
```

---

## Stream 1: JSON-RPC Protocol

**Directory**: `internal/runtime/plugin/jsonrpc/`

- `protocol.go` — JSON-RPC 2.0 message types (Request, Response, Notification)
- `transport.go` — Bidirectional stdin/stdout transport with message framing
- `server.go` — Go-side server: handles calls FROM Node.js TO Go (bridge calls)
- `client.go` — Go-side client: sends calls FROM Go TO Node.js (execute script)

**Key**: Bidirectional — Go calls Node.js to execute scripts, Node.js calls Go for bridge API. Both over same stdin/stdout pipe.

## Stream 2: Process Pool

**File**: `internal/runtime/plugin/pool.go`

Unified pool for ALL child process runtimes (Node.js, Python):
- Worker pool: short-lived scripts (API handlers, event handlers)
- Background pool: long-running scripts (cron, batch)
- Config: `runtime.pool.worker_size`, `runtime.pool.background_size`
- Health check: ping process, restart if dead
- Graceful shutdown: drain queue, wait for running scripts

## Stream 3: Node.js Runtime Rewrite

**File**: `plugins/typescript/` → rewrite entirely

- `runtime.js` — Main entry: starts JSON-RPC server, registers bridge proxy
- `bridge.js` — Proxy that translates `bitcode.*` calls to JSON-RPC calls to Go
- `loader.js` — Script loader with esbuild for TypeScript transpilation

## Stream 4: Bun + TypeScript

- Auto-detect: check `bun --version` first, fallback to `node --version`
- esbuild bundled for TypeScript → JavaScript transpilation
- Config: `runtime.node.engine: "auto"` | `"node"` | `"bun"`

## Stream 5: npm per-Module

- Each module can have `package.json` in its directory
- Engine runs `npm install` (or `bun install`) per module on startup
- `node_modules` per module, not global

## Definition of Done

- [ ] Bidirectional JSON-RPC works (Go ↔ Node.js)
- [ ] Process pool manages worker + background processes
- [ ] All 6 TypeScript sample scripts execute with real bridge calls
- [ ] Bun auto-detection works
- [ ] TypeScript transpilation via esbuild works
- [ ] npm per-module works
- [ ] Crash recovery: process restarts on unexpected exit
- [ ] Timeout: script killed after configured timeout
