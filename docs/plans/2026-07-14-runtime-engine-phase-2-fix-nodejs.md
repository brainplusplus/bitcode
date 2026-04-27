# Phase 2: Fix Node.js Child Process + Real Bridge

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: Phase 1 (bridge API interfaces), Phase 1.5 (multi-tenancy)
**Unlocks**: Phase 6 (engine enhancements), Phase 7 (module setting)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal) — prerequisites, runtime optional config, success criteria
2. [Current Architecture — What's Wrong](#2-current-architecture--whats-wrong)
3. [New Architecture — Bidirectional JSON-RPC](#3-new-architecture--bidirectional-json-rpc)
4. [Protocol Design](#4-protocol-design) — message types, bridge method mapping, transactions
5. [Node.js Runtime Rewrite](#5-nodejs-runtime-rewrite) — runtime.js, bitcode proxy, design decisions
6. [Go Side Changes](#6-go-side-changes) — manager, bridge handler, script handler
7. [Per-Module Dependency Isolation](#7-per-module-dependency-isolation) — structure, require resolution
8. [npm Package Management](#8-npm-package-management-step-by-step) — step-by-step guide, CLI commands
9. [Script Signature Migration](#9-script-signature-migration) — old vs new, backward compatibility
10. [Critical Design Decisions](#10-critical-design-decisions) — pool, TypeScript, crash recovery, binary, console
11. [Edge Cases](#11-edge-cases)
12. [Implementation Tasks](#12-implementation-tasks)

---

## 1. Goal

Make the Node.js child process runtime **actually work** — scripts can call all 20 `bitcode.*` bridge methods with real results, not stubs.

### Prerequisites

- **Node.js 20+** (minimum). Recommended: Node.js 22 LTS.
  - Node.js 18: EOL April 2025 — not supported
  - Node.js 20 LTS: supported until April 2026
  - Node.js 22 LTS: recommended, supported until April 2027
  - Why 20+: esbuild (TypeScript transpilation) requires Node.js 18+, but 18 is EOL
- **npm** (bundled with Node.js)
- **Optional**: Node.js runtime is NOT required to run the engine. See §1.2.

### 1.1 Success Criteria

- All 6 TypeScript scripts in `samples/erp` execute with real DB operations
- `bitcode.model("lead").search({...})` returns actual records from database
- `bitcode.email.send({...})` actually sends email
- Per-module `package.json` + `node_modules` isolation works
- `require()` resolves from module directory, not `plugins/typescript/`
- Backward compatible: old-style scripts (`definePlugin`) still work
- Error contract: `BridgeError` thrown on failures with correct codes

### 1.2 Runtime is Optional — Can Be Disabled

Every runtime (node, python, goja, yaegi) is independently configurable:

```yaml
# bitcode.yaml
runtime:
  node:
    enabled: true              # default: true (auto-detect Node.js in PATH)
    enabled: false             # explicitly disable — no Node.js process started
    enabled: "auto"            # default: start if Node.js found in PATH, skip if not
    
    command: "node"            # path to Node.js binary (default: "node")
    min_version: "20.0.0"     # minimum required version
    
# Pool config is UNIFIED — applies to ALL runtimes (node, python, goja, qjs, yaegi)
# See runtime.worker and runtime.background in bitcode.yaml
# No per-runtime pool config — simplicity over granularity
    
  python:
    enabled: "auto"            # same pattern
    command: "python3"
    
  javascript:                  # goja — always available (embedded)
    enabled: true              # cannot be disabled (zero cost when not used)
    
  go:                          # yaegi — always available (embedded)
    enabled: true              # cannot be disabled (zero cost when not used)
```

**Default: `enabled: "auto"`** — engine checks if Node.js is in PATH at startup:
- Found → start process pool, log `[INFO] Node.js 22.5.0 runtime ready (pool: 4)`
- Not found → skip, log `[INFO] Node.js runtime not available (not in PATH). Scripts with runtime: "node" will use fallback or fail.`
- Found but version < 20 → skip, log `[WARN] Node.js 16.x found but minimum 20.0.0 required. Skipping.`

**When disabled/unavailable and script needs it:**
```
Process step: { "type": "script", "runtime": "node", "script": "scripts/crawl.js" }
→ BridgeError { code: "RUNTIME_NOT_AVAILABLE", message: "Node.js runtime is not available. Install Node.js 20+ or set runtime.node.enabled in bitcode.yaml" }
```

**Why this matters:**
- Embedded runtimes (goja, yaegi) are always available — zero dependency
- External runtimes (node, python) are optional — not everyone needs them
- Production server without Node.js can still run 100% of goja/yaegi scripts
- Developer laptop with Node.js gets full npm ecosystem access

### 1.3 What This Phase Does NOT Do

- Does not add goja or yaegi (Phase 4-5)
- Does not change process engine or view system (Phase 6)
- Does not build module setting (Phase 7)

---

## 2. Current Architecture — What's Wrong

### Current Flow (One-Way)

```
Go Engine                          Node.js Process
    │                                    │
    │ ── stdin: { method: "execute",     │
    │      params: { script, params } }  │
    │ ──────────────────────────────────► │
    │                                    │ vm.runInNewContext(code, sandbox)
    │                                    │ sandbox.ctx = { db: STUB, http: STUB }
    │                                    │ plugin.execute(STUB_ctx, params)
    │                                    │
    │ ◄────────────────────────────────── │
    │    stdout: { result: {...} }       │
    │                                    │
```

### Problems

| # | Problem | Impact |
|---|---------|--------|
| 1 | **Bridge is stub** — `ctx.db.query()` returns `{ rows: [] }` | Scripts can't do anything useful |
| 2 | **One-way communication** — Go sends request, Node.js returns result. No way for Node.js to call back to Go. | Bridge methods can't work — they need Go to execute DB/HTTP/etc. |
| 3 | **No module context** — Node.js doesn't know which module the script belongs to | Can't resolve `require()` from module dir, can't apply security rules |
| 4 | **Single process** — all modules share one Node.js process | Dependency conflicts, no isolation |
| 5 | **`require()` resolves from `plugins/typescript/`** — not from module dir | npm packages per module impossible |
| 6 | **No error structure** — errors are plain strings | No `BridgeError` with codes |
| 7 | **`import { definePlugin }` pattern** — requires `@bitcode/sdk` package that doesn't exist | Confusing, unnecessary wrapper |

### Root Cause

The fundamental issue is **one-way communication**. For bridge to work, Node.js must be able to call Go mid-execution:

```javascript
// Script calls bitcode.model("lead").search(...)
// This needs to:
// 1. Send request to Go: "please execute model.search with these params"
// 2. Wait for Go to execute and return results
// 3. Continue script execution with the results
```

This requires **bidirectional JSON-RPC**.

---

## 3. New Architecture — Bidirectional JSON-RPC

### New Flow

```
Go Engine                          Node.js Process
    │                                    │
    │ ── { method: "execute",            │
    │      params: { script, params,     │
    │               module, session } }  │
    │ ──────────────────────────────────► │
    │                                    │ Load script from module dir
    │                                    │ Create bitcode proxy object
    │                                    │ script.execute(bitcode, params)
    │                                    │
    │                                    │ // Script calls bitcode.model("lead").search(...)
    │ ◄────────────────────────────────── │
    │    { method: "bridge.model.search", │
    │      params: { model: "lead",      │
    │               opts: {...} },       │
    │      id: 1 }                       │
    │                                    │
    │ Go executes: bridge.Model("lead")  │
    │              .Search(opts)         │
    │                                    │
    │ ── { result: [records...], id: 1 } │
    │ ──────────────────────────────────► │
    │                                    │ // Script receives records
    │                                    │ // Script continues...
    │                                    │
    │                                    │ // Script calls bitcode.email.send(...)
    │ ◄────────────────────────────────── │
    │    { method: "bridge.email.send",  │
    │      params: {...}, id: 2 }        │
    │                                    │
    │ Go executes: bridge.Email().Send() │
    │                                    │
    │ ── { result: null, id: 2 }         │
    │ ──────────────────────────────────► │
    │                                    │ // Script continues...
    │                                    │ // Script returns final result
    │                                    │
    │ ◄────────────────────────────────── │
    │    { method: "execute.complete",   │
    │      result: { success: true } }   │
    │                                    │
```

### Key Insight

The communication is now **interleaved**:
1. Go sends `execute` request
2. Node.js starts executing script
3. Script calls `bitcode.*` → Node.js sends bridge request to Go
4. Go executes bridge method, sends result back
5. Script continues, may call more bridge methods
6. Script finishes, Node.js sends final result

**Go must be able to both send AND receive on stdin/stdout simultaneously.** This means the Go side needs a goroutine reading responses while the main goroutine processes bridge requests.

---

## 4. Protocol Design

> **Cross-Runtime Execution**: Steps in a single process can use different runtimes (Node.js → Python → goja → yaegi). Data flows via JSON between steps. See master doc §4.4 for full documentation including data flow diagrams, pool selection for mixed-runtime processes, and edge cases.

### Message Types

```
Go → Node.js:
  { type: "execute", id: 1, params: { script, params, module, session, securityRules } }
  { type: "bridge_response", id: 42, result: {...} }
  { type: "bridge_response", id: 42, error: { code, message, details } }

Node.js → Go:
  { type: "bridge_request", id: 42, method: "model.search", params: {...} }
  { type: "bridge_request", id: 43, method: "email.send", params: {...} }
  { type: "execute_complete", id: 1, result: {...} }
  { type: "execute_error", id: 1, error: { code, message, stack } }
```

### Bridge Method Mapping

Every `bitcode.*` call maps to a bridge method string:

```
bitcode.model("lead").search(opts)           → "model.search"       { model: "lead", opts }
bitcode.model("lead").get(id)                → "model.get"          { model: "lead", id }
bitcode.model("lead").create(data)           → "model.create"       { model: "lead", data }
bitcode.model("lead").write(id, data)        → "model.write"        { model: "lead", id, data }
bitcode.model("lead").delete(id)             → "model.delete"       { model: "lead", id }
bitcode.model("lead").count(opts)            → "model.count"        { model: "lead", opts }
bitcode.model("lead").sum(field, opts)       → "model.sum"          { model: "lead", field, opts }
bitcode.model("lead").upsert(data, unique)   → "model.upsert"      { model: "lead", data, unique }
bitcode.model("lead").createMany(records)    → "model.createMany"   { model: "lead", records }
bitcode.model("lead").writeMany(ids, data)   → "model.writeMany"    { model: "lead", ids, data }
bitcode.model("lead").deleteMany(ids)        → "model.deleteMany"   { model: "lead", ids }
bitcode.model("lead").addRelation(...)       → "model.addRelation"  { model, id, field, relatedIds }
bitcode.model("lead").removeRelation(...)    → "model.removeRelation" { ... }
bitcode.model("lead").setRelation(...)       → "model.setRelation"  { ... }
bitcode.model("lead").loadRelation(...)      → "model.loadRelation" { ... }
bitcode.model("lead").sudo().search(opts)    → "model.search"       { model: "lead", opts, sudo: true }
bitcode.model("lead").sudo().hardDelete(id)  → "model.hardDelete"   { model: "lead", id, sudo: true }
bitcode.model("lead").sudo().withTenant("x") → "model.search"       { ..., sudo: true, tenant: "x" }

bitcode.db.query(sql, args)                  → "db.query"           { sql, args }
bitcode.db.execute(sql, args)                → "db.execute"         { sql, args }

bitcode.http.get(url, opts)                  → "http.request"       { method: "GET", url, opts }
bitcode.http.post(url, opts)                 → "http.request"       { method: "POST", url, opts }

bitcode.cache.get(key)                       → "cache.get"          { key }
bitcode.cache.set(key, val, opts)            → "cache.set"          { key, value, opts }
bitcode.cache.del(key)                       → "cache.del"          { key }

bitcode.env(key)                             → "env.get"            { key }
bitcode.config(key)                          → "config.get"         { key }
bitcode.log(level, msg, data)                → "log"                { level, msg, data }
bitcode.emit(event, data)                    → "emit"               { event, data }
bitcode.call(process, input)                 → "call"               { process, input }

bitcode.fs.read(path)                        → "fs.read"            { path }
bitcode.fs.write(path, content)              → "fs.write"           { path, content }
bitcode.fs.exists(path)                      → "fs.exists"          { path }
bitcode.fs.list(path)                        → "fs.list"            { path }
bitcode.fs.mkdir(path)                       → "fs.mkdir"           { path }
bitcode.fs.remove(path)                      → "fs.remove"          { path }

bitcode.exec(cmd, args, opts)                → "exec"               { cmd, args, opts }

bitcode.email.send(opts)                     → "email.send"         { opts }
bitcode.notify.send(opts)                    → "notify.send"        { opts }
bitcode.notify.broadcast(channel, data)      → "notify.broadcast"   { channel, data }
bitcode.storage.upload(opts)                 → "storage.upload"     { opts }
bitcode.storage.url(id)                      → "storage.url"        { id }
bitcode.storage.download(id)                 → "storage.download"   { id }
bitcode.storage.delete(id)                   → "storage.delete"     { id }
bitcode.t(key)                               → "t"                  { key }
bitcode.security.permissions(model)          → "security.permissions" { model }
bitcode.security.hasGroup(group)             → "security.hasGroup"  { group }
bitcode.security.groups()                    → "security.groups"    {}
bitcode.audit.log(opts)                      → "audit.log"          { opts }
bitcode.crypto.encrypt(text)                 → "crypto.encrypt"     { text }
bitcode.crypto.decrypt(text)                 → "crypto.decrypt"     { text }
bitcode.crypto.hash(value)                   → "crypto.hash"        { value }
bitcode.crypto.verify(value, hash)           → "crypto.verify"      { value, hash }
bitcode.execution.search(opts)               → "execution.search"   { opts }
bitcode.execution.get(id, opts)              → "execution.get"      { id, opts }
bitcode.execution.current()                  → "execution.current"  {}
bitcode.execution.retry(id)                  → "execution.retry"    { id }
bitcode.execution.cancel(id)                 → "execution.cancel"   { id }

bitcode.tx(fn)                               → Special: see §4.1
bitcode.session                              → Injected at execute time, no RPC needed
```

### 4.1 Transaction Handling

`bitcode.tx()` is special — it wraps multiple bridge calls in one DB transaction:

```
Node.js → Go:  { type: "bridge_request", method: "tx.begin", id: 50 }
Go → Node.js:  { type: "bridge_response", id: 50, result: { txId: "tx-uuid" } }

// All subsequent bridge calls include txId:
Node.js → Go:  { type: "bridge_request", method: "model.create", id: 51, txId: "tx-uuid", params: {...} }
Go → Node.js:  { type: "bridge_response", id: 51, result: {...} }

Node.js → Go:  { type: "bridge_request", method: "model.write", id: 52, txId: "tx-uuid", params: {...} }
Go → Node.js:  { type: "bridge_response", id: 52, result: {...} }

// Commit or rollback:
Node.js → Go:  { type: "bridge_request", method: "tx.commit", id: 53, txId: "tx-uuid" }
Go → Node.js:  { type: "bridge_response", id: 53, result: { committed: true } }

// On error (automatic):
Node.js → Go:  { type: "bridge_request", method: "tx.rollback", id: 54, txId: "tx-uuid" }
```

Go side holds the `*gorm.DB` transaction in a map keyed by `txId`. Bridge calls with `txId` use the transaction DB instead of the main DB.

### 4.2 Synchronous Properties

These don't need RPC — injected at execute time:

```javascript
bitcode.session.userId      // injected in execute params, no RPC
bitcode.session.tenantId    // injected in execute params, no RPC
bitcode.session.groups      // injected in execute params, no RPC
bitcode.session.locale      // injected in execute params, no RPC
bitcode.session.context     // injected in execute params, no RPC
```

`bitcode.env()` and `bitcode.config()` could also be pre-loaded and injected to avoid RPC for simple lookups. But for security (env whitelist), it's safer to go through Go.

---

## 5. Node.js Runtime Rewrite

### 5.1 New File: `plugins/node/runtime.js`

Rename from `plugins/typescript/index.js` to `plugins/node/runtime.js`. Complete rewrite:

```javascript
// plugins/node/runtime.js
const readline = require('readline');
const path = require('path');
const vm = require('vm');
const fs = require('fs');
const Module = require('module');

// Pending bridge requests: id → { resolve, reject }
const pendingBridgeRequests = new Map();
let nextBridgeId = 1;

// --- Communication Layer ---

function sendToGo(message) {
  process.stdout.write(JSON.stringify(message) + '\n');
}

function bridgeCall(method, params, txId) {
  return new Promise((resolve, reject) => {
    const id = nextBridgeId++;
    pendingBridgeRequests.set(id, { resolve, reject });
    const msg = { type: 'bridge_request', id, method, params };
    if (txId) msg.txId = txId;
    sendToGo(msg);
  });
}

// --- Bridge Proxy (bitcode.*) ---

function createBitcodeProxy(session, securityRules, moduleDir) {
  function createModelHandle(modelName, sudoMode, tenantOverride, skipVal) {
    const baseParams = { model: modelName };
    if (sudoMode) baseParams.sudo = true;
    if (tenantOverride) baseParams.tenant = tenantOverride;
    if (skipVal) baseParams.skipValidation = true;

    const handle = {
      search: (opts = {}) => bridgeCall('model.search', { ...baseParams, opts }),
      get: (id, opts) => bridgeCall('model.get', { ...baseParams, id, opts }),
      create: (data) => bridgeCall('model.create', { ...baseParams, data }),
      write: (id, data) => bridgeCall('model.write', { ...baseParams, id, data }),
      delete: (id) => bridgeCall('model.delete', { ...baseParams, id }),
      count: (opts = {}) => bridgeCall('model.count', { ...baseParams, opts }),
      sum: (field, opts = {}) => bridgeCall('model.sum', { ...baseParams, field, opts }),
      upsert: (data, unique) => bridgeCall('model.upsert', { ...baseParams, data, unique }),
      createMany: (records) => bridgeCall('model.createMany', { ...baseParams, records }),
      writeMany: (ids, data) => bridgeCall('model.writeMany', { ...baseParams, ids, data }),
      deleteMany: (ids) => bridgeCall('model.deleteMany', { ...baseParams, ids }),
      upsertMany: (records, unique) => bridgeCall('model.upsertMany', { ...baseParams, records, unique }),
      addRelation: (id, field, ids) => bridgeCall('model.addRelation', { ...baseParams, id, field, relatedIds: ids }),
      removeRelation: (id, field, ids) => bridgeCall('model.removeRelation', { ...baseParams, id, field, relatedIds: ids }),
      setRelation: (id, field, ids) => bridgeCall('model.setRelation', { ...baseParams, id, field, relatedIds: ids }),
      loadRelation: (id, field) => bridgeCall('model.loadRelation', { ...baseParams, id, field }),
    };

    if (!sudoMode) {
      handle.sudo = () => createModelHandle(modelName, true, null, false);
    } else {
      handle.hardDelete = (id) => bridgeCall('model.hardDelete', { ...baseParams, id });
      handle.hardDeleteMany = (ids) => bridgeCall('model.hardDeleteMany', { ...baseParams, ids });
      handle.withTenant = (tid) => createModelHandle(modelName, true, tid, skipVal);
      handle.skipValidation = () => createModelHandle(modelName, true, tenantOverride, true);
    }

    return handle;
  }

  return {
    model: (name) => createModelHandle(name, false, null, false),

    db: {
      query: (sql, ...args) => bridgeCall('db.query', { sql, args }),
      execute: (sql, ...args) => bridgeCall('db.execute', { sql, args }),
    },

    http: {
      get: (url, opts) => bridgeCall('http.request', { method: 'GET', url, ...opts }),
      post: (url, opts) => bridgeCall('http.request', { method: 'POST', url, ...opts }),
      put: (url, opts) => bridgeCall('http.request', { method: 'PUT', url, ...opts }),
      patch: (url, opts) => bridgeCall('http.request', { method: 'PATCH', url, ...opts }),
      delete: (url, opts) => bridgeCall('http.request', { method: 'DELETE', url, ...opts }),
    },

    cache: {
      get: (key) => bridgeCall('cache.get', { key }),
      set: (key, value, opts) => bridgeCall('cache.set', { key, value, ...opts }),
      del: (key) => bridgeCall('cache.del', { key }),
    },

    env: (key) => bridgeCall('env.get', { key }),
    session: session,  // injected, no RPC
    config: (key) => bridgeCall('config.get', { key }),
    log: (level, msg, data) => bridgeCall('log', { level, msg, data }),
    emit: (event, data) => bridgeCall('emit', { event, data }),
    call: (process, input) => bridgeCall('call', { process, input }),

    fs: {
      read: (p) => bridgeCall('fs.read', { path: p }),
      write: (p, content) => bridgeCall('fs.write', { path: p, content }),
      exists: (p) => bridgeCall('fs.exists', { path: p }),
      list: (p) => bridgeCall('fs.list', { path: p }),
      mkdir: (p) => bridgeCall('fs.mkdir', { path: p }),
      remove: (p) => bridgeCall('fs.remove', { path: p }),
    },

    exec: (cmd, args, opts) => bridgeCall('exec', { cmd, args, ...opts }),

    email: {
      send: (opts) => bridgeCall('email.send', opts),
    },

    notify: {
      send: (opts) => bridgeCall('notify.send', opts),
      broadcast: (channel, data) => bridgeCall('notify.broadcast', { channel, data }),
    },

    storage: {
      upload: (opts) => bridgeCall('storage.upload', opts),
      url: (id) => bridgeCall('storage.url', { id }),
      download: (id) => bridgeCall('storage.download', { id }),
      delete: (id) => bridgeCall('storage.delete', { id }),
    },

    t: (key) => bridgeCall('t', { key }),

    security: {
      permissions: (model) => bridgeCall('security.permissions', { model }),
      hasGroup: (group) => bridgeCall('security.hasGroup', { group }),
      groups: () => bridgeCall('security.groups', {}),
    },

    audit: {
      log: (opts) => bridgeCall('audit.log', opts),
    },

    crypto: {
      encrypt: (text) => bridgeCall('crypto.encrypt', { text }),
      decrypt: (text) => bridgeCall('crypto.decrypt', { text }),
      hash: (value) => bridgeCall('crypto.hash', { value }),
      verify: (value, hash) => bridgeCall('crypto.verify', { value, hash }),
    },

    execution: {
      search: (opts) => bridgeCall('execution.search', opts),
      get: (id, opts) => bridgeCall('execution.get', { id, ...opts }),
      current: () => bridgeCall('execution.current', {}),
      retry: (id) => bridgeCall('execution.retry', { id }),
      cancel: (id) => bridgeCall('execution.cancel', { id }),
    },

    tx: async (fn) => {
      const { txId } = await bridgeCall('tx.begin', {});
      const origBridgeCall = bridgeCall;
      // Override bridgeCall to include txId for duration of fn
      const txBridgeCall = (method, params) => origBridgeCall(method, params, txId);
      // TODO: create tx-scoped bitcode proxy that uses txBridgeCall
      try {
        const result = await fn(/* tx-scoped bitcode */);
        await bridgeCall('tx.commit', { txId });
        return result;
      } catch (e) {
        await bridgeCall('tx.rollback', { txId });
        throw e;
      }
    },
  };
}

// --- Script Execution ---

async function executeScript(scriptPath, params, module, session, securityRules) {
  const moduleDir = module ? path.resolve('modules', module) : path.dirname(scriptPath);
  const bitcode = createBitcodeProxy(session, securityRules, moduleDir);

  if (!fs.existsSync(scriptPath)) {
    throw new Error(`script not found: ${scriptPath}`);
  }

  const code = fs.readFileSync(scriptPath, 'utf-8');

  // Create require that resolves from module directory
  const moduleRequire = Module.createRequire(path.resolve(moduleDir, 'node_modules', '_bridge.js'));

  const sandbox = {
    module: { exports: {} },
    exports: {},
    require: moduleRequire,
    console,
    bitcode,
    params,
    // Legacy support
    ctx: bitcode,
    setTimeout, setInterval, clearTimeout, clearInterval,
    Promise, Buffer, URL, URLSearchParams,
  };

  vm.runInNewContext(code, sandbox, { filename: scriptPath, timeout: 30000 });

  const plugin = sandbox.module.exports.default || sandbox.module.exports;

  if (typeof plugin === 'function') {
    return await plugin(bitcode, params);
  } else if (plugin && typeof plugin.execute === 'function') {
    // New style: execute(bitcode, params)
    // Legacy style: execute(ctx, params) — ctx is same as bitcode
    return await plugin.execute(bitcode, params);
  } else {
    return { executed: true, script: scriptPath };
  }
}

// --- Message Loop ---

const rl = readline.createInterface({ input: process.stdin, terminal: false });

rl.on('line', async (line) => {
  let message;
  try {
    message = JSON.parse(line);
  } catch (e) {
    sendToGo({ type: 'error', error: 'invalid JSON' });
    return;
  }

  if (message.type === 'execute') {
    // Go is asking us to execute a script
    const { id, params } = message;
    try {
      const result = await executeScript(
        params.script,
        params.params || {},
        params.module,
        params.session || {},
        params.securityRules || {}
      );
      sendToGo({ type: 'execute_complete', id, result });
    } catch (e) {
      sendToGo({
        type: 'execute_error', id,
        error: { code: e.code || 'SCRIPT_ERROR', message: e.message, stack: e.stack }
      });
    }
  } else if (message.type === 'bridge_response') {
    // Go is responding to our bridge request
    const pending = pendingBridgeRequests.get(message.id);
    if (pending) {
      pendingBridgeRequests.delete(message.id);
      if (message.error) {
        const err = new Error(message.error.message);
        err.code = message.error.code;
        err.details = message.error.details;
        err.retryable = message.error.retryable;
        pending.reject(err);
      } else {
        pending.resolve(message.result);
      }
    }
  }
});

process.stderr.write('[plugin:node] ready\n');
```

### 5.2 Key Design Decisions

**Why `vm.runInNewContext` instead of `require()`?**
- Sandboxing: script can't access Node.js internals unless we explicitly expose them
- We control what's in the sandbox (`bitcode`, `params`, `require`, `console`)
- `require` is a custom one that resolves from module dir

**Why `Module.createRequire(moduleDir)`?**
- This is the official Node.js way to create a `require` function that resolves from a specific directory
- `require('axios')` will look in `modules/crm/node_modules/axios/`
- Not in `plugins/node/node_modules/` (which doesn't exist)

**Why expose `setTimeout`, `Promise`, `Buffer`?**
- `vm.runInNewContext` creates a completely empty context
- Without these, basic async patterns break
- `Promise` is needed for `async/await`
- `Buffer` is needed for binary data (file upload/download)
- `setTimeout` is needed for retry patterns

---

## 6. Go Side Changes

### 6.1 Bidirectional Communication in Manager

Current `manager.go` is synchronous: send request, read one response. New version must handle interleaved messages:

```go
// plugin/manager.go — new Execute flow

func (m *Manager) Execute(ctx context.Context, pluginName string, script string, 
    params map[string]any, bridgeCtx *bridge.Context) (any, error) {
    
    p := m.getPlugin(pluginName)
    
    execID := p.nextID()
    
    // Send execute request
    p.send(Message{
        Type:   "execute",
        ID:     execID,
        Params: map[string]any{
            "script":        script,
            "params":        params,
            "module":        bridgeCtx.ModuleName(),
            "session":       bridgeCtx.Session(),
            "securityRules": bridgeCtx.SecurityRules(),
        },
    })
    
    // Message loop: handle bridge requests until execute completes
    for {
        msg, err := p.receive()
        if err != nil {
            return nil, err
        }
        
        switch msg.Type {
        case "bridge_request":
            // Script is calling a bridge method — execute it
            result, bridgeErr := m.handleBridgeRequest(ctx, bridgeCtx, msg)
            if bridgeErr != nil {
                p.send(Message{
                    Type:  "bridge_response",
                    ID:    msg.ID,
                    Error: bridgeErr,
                })
            } else {
                p.send(Message{
                    Type:   "bridge_response",
                    ID:     msg.ID,
                    Result: result,
                })
            }
            
        case "execute_complete":
            if msg.ID == execID {
                return msg.Result, nil
            }
            
        case "execute_error":
            if msg.ID == execID {
                return nil, &bridge.BridgeError{
                    Code:    msg.Error.Code,
                    Message: msg.Error.Message,
                }
            }
        }
    }
}
```

### 6.2 Bridge Request Handler

```go
func (m *Manager) handleBridgeRequest(ctx context.Context, bc *bridge.Context, msg Message) (any, *bridge.BridgeError) {
    switch msg.Method {
    // Model CRUD
    case "model.search":
        return bc.Model(msg.Params["model"].(string)).Search(parseSearchOpts(msg.Params))
    case "model.get":
        return bc.Model(msg.Params["model"].(string)).Get(msg.Params["id"].(string))
    case "model.create":
        return bc.Model(msg.Params["model"].(string)).Create(msg.Params["data"].(map[string]any))
    // ... all 50+ bridge methods mapped here
    
    // Transaction
    case "tx.begin":
        txId := bc.TxBegin()
        return map[string]any{"txId": txId}, nil
    case "tx.commit":
        return nil, bc.TxCommit(msg.Params["txId"].(string))
    case "tx.rollback":
        return nil, bc.TxRollback(msg.Params["txId"].(string))
    
    default:
        return nil, &bridge.BridgeError{Code: "UNKNOWN_METHOD", Message: "unknown bridge method: " + msg.Method}
    }
}
```

### 6.3 ScriptHandler Changes

```go
// steps/script.go — updated

type ScriptHandler struct {
    Runner       ScriptRunner
    BridgeFactory *bridge.Factory
}

func (h *ScriptHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
    // Create bridge context for this script execution
    session := bridge.Session{
        UserID:   execCtx.UserID,
        Locale:   execCtx.Locale,
        // ... populated from execCtx
    }
    rules := h.loadSecurityRules(step.ModuleName)
    bridgeCtx := h.BridgeFactory.NewContext(step.ModuleName, session, rules)
    
    params := map[string]any{
        "input":     execCtx.Input,
        "variables": execCtx.Variables,
        "result":    execCtx.Result,
        "user_id":   execCtx.UserID,
    }
    
    // Pass bridge context to runner
    result, err := h.Runner.RunWithBridge(ctx, step.Script, params, bridgeCtx)
    if err != nil {
        return fmt.Errorf("script %s failed: %w", step.Script, err)
    }
    
    // ... store result
}
```

---

## 7. Per-Module Dependency Isolation

### 7.1 Module Structure

```
modules/crm/
├── module.json
├── package.json          ← npm dependencies for this module
├── node_modules/         ← isolated, only for this module
├── scripts/
│   ├── on_deal_won.ts
│   └── crawl_linkedin.js
└── ...
```

### 7.2 CLI Command

```bash
# Install npm dependencies for a specific module
bitcode module install-deps crm

# What it does:
# 1. cd modules/crm
# 2. npm install (reads package.json, creates node_modules/)
# 3. Done

# Install all module dependencies
bitcode module install-deps --all
```

### 7.3 How require() Resolves

```javascript
// In runtime.js:
const moduleRequire = Module.createRequire(
  path.resolve(moduleDir, 'node_modules', '_bridge.js')
);

// Now when script does:
const axios = require('axios');
// It resolves: modules/crm/node_modules/axios/

// Standard Node.js resolution:
// 1. modules/crm/node_modules/axios/
// 2. modules/node_modules/axios/  (parent)
// 3. node_modules/axios/  (root)
// This is correct — module-level first, then fallback to project-level
```

### 7.4 Edge Cases

| Edge Case | Behavior |
|-----------|----------|
| Module has no `package.json` | `require()` still works for Node.js built-ins and project-level packages |
| Two modules need different versions of same package | Each has own `node_modules/` — no conflict |
| Script requires package not installed | Standard Node.js error: "Cannot find module 'xxx'" |
| `bitcode module install-deps crm` but no package.json | No-op, warning logged |
| Module in embedded (go:embed) | Scripts extracted to temp dir at startup. `npm install` runs there. |

---

## 8. npm Package Management (Step-by-Step)

### 8.1 How Developers Add npm Packages to a Module

**Step 1: Create `package.json` in module directory**

```bash
cd modules/crm
npm init -y
```

This creates `modules/crm/package.json`:
```json
{
  "name": "bitcode-module-crm",
  "version": "1.0.0",
  "private": true,
  "dependencies": {}
}
```

**Step 2: Install packages**

```bash
# Option A: npm directly
cd modules/crm
npm install axios cheerio

# Option B: bitcode CLI (recommended)
bitcode module install-deps crm
# → reads modules/crm/package.json, runs npm install in modules/crm/
```

Result:
```
modules/crm/
├── package.json
├── node_modules/          ← created by npm
│   ├── axios/
│   └── cheerio/
├── package-lock.json      ← created by npm
└── scripts/
    └── crawl_leads.js     ← can now require('axios')
```

**Step 3: Use in script**

```javascript
// modules/crm/scripts/crawl_leads.js
const axios = require('axios');
const cheerio = require('cheerio');

export default {
  async execute(bitcode, params) {
    const resp = await axios.get('https://example.com/leads');
    const $ = cheerio.load(resp.data);
    // ... parse HTML, create leads via bitcode.model()
  }
};
```

### 8.2 CLI Commands

```bash
# Install dependencies for one module
bitcode module install-deps crm
# → cd modules/crm && npm install

# Install dependencies for ALL modules that have package.json
bitcode module install-deps --all
# → finds all modules/*/package.json, runs npm install in each

# Add a package to a module
bitcode module add-package crm axios
# → cd modules/crm && npm install axios --save

# Remove a package from a module
bitcode module remove-package crm axios
# → cd modules/crm && npm uninstall axios --save
```

### 8.3 How require() Resolution Works

```
Script: modules/crm/scripts/crawl_leads.js
require('axios')

Node.js resolution order:
1. modules/crm/node_modules/axios/          ← module-level (first priority)
2. modules/node_modules/axios/              ← parent directory
3. node_modules/axios/                      ← project root
4. $NODE_PATH/axios/                        ← global
5. Node.js built-ins (fs, path, http, etc.) ← always available

This is standard Node.js module resolution — no custom logic needed.
Module.createRequire(moduleDir) handles this automatically.
```

### 8.4 What About node_modules in .gitignore?

Developers should add to `.gitignore`:
```
modules/*/node_modules/
modules/*/package-lock.json
```

And in CI/CD or deployment:
```bash
bitcode module install-deps --all
```

### 8.5 What About Embedded Modules?

Embedded modules (in `engine/embedded/modules/`) are compiled into the Go binary. They can't have `node_modules/` inside the binary.

**Solution**: At startup, engine extracts embedded module scripts to a temp directory. If the embedded module has a `package.json`, engine runs `npm install` in the temp directory.

```go
func (a *App) extractEmbeddedModules() {
    for _, mod := range embeddedModules {
        tmpDir := filepath.Join(os.TempDir(), "bitcode", "modules", mod.Name)
        extractFS(mod.FS, tmpDir)
        
        if fileExists(filepath.Join(tmpDir, "package.json")) {
            exec.Command("npm", "install", "--production").Dir(tmpDir).Run()
        }
    }
}
```

### 8.6 Edge Cases

| Edge Case | Behavior |
|-----------|----------|
| Module has no `package.json` | `require()` only resolves Node.js built-ins and project-level packages. No error. |
| `npm install` fails (network error) | CLI shows error. Module still works — bridge methods (`bitcode.*`) don't need npm. |
| Two modules need different versions of `axios` | Each has own `node_modules/`. No conflict. Correct version resolved per module. |
| Script requires package not installed | Standard Node.js error: `Error: Cannot find module 'axios'` |
| `bitcode module install-deps crm` but no `package.json` | No-op, warning: "No package.json found in modules/crm/" |
| Package has native bindings (e.g., `sharp`, `bcrypt`) | Works — npm compiles native modules for current platform. Same as any Node.js project. |
| `node_modules/` is very large (e.g., `puppeteer` ~300MB) | Per-module isolation means only modules that need it pay the cost. Other modules unaffected. |

---

## 9. Script Signature Migration

### Old Style (current)

```typescript
import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    console.log(`Deal won: ${params.input.name}`);
    return { success: true };
  }
});
```

### New Style

```typescript
export default {
  async execute(bitcode, params) {
    const lead = params.input;
    await bitcode.model("activity").create({
      lead_id: lead.id,
      type: "task",
      summary: "Send welcome package"
    });
    await bitcode.email.send({
      to: "manager@company.com",
      subject: "Deal Won: " + lead.name,
      body: "<h1>Revenue: $" + lead.expected_revenue + "</h1>"
    });
    bitcode.log("info", "Deal won processed", { leadId: lead.id });
    return { success: true };
  }
};
```

### Backward Compatibility

Runtime supports both:

```javascript
// In runtime.js executeScript():
const plugin = sandbox.module.exports.default || sandbox.module.exports;

if (typeof plugin === 'function') {
  // Style: module.exports = function(bitcode, params) { ... }
  return await plugin(bitcode, params);
} else if (plugin && typeof plugin.execute === 'function') {
  // Style: module.exports = { execute(bitcode, params) { ... } }
  // Also works with old: definePlugin({ execute(ctx, params) { ... } })
  // Because ctx === bitcode (same object passed)
  return await plugin.execute(bitcode, params);
}
```

`import { definePlugin }` will still work because:
1. `definePlugin` is just a passthrough function: `const definePlugin = (obj) => obj`
2. We can provide a shim in the sandbox: `sandbox.definePlugin = (obj) => obj`
3. Or: `require('@bitcode/sdk')` returns `{ definePlugin: (obj) => obj }`

### Process JSON: `runtime` Field

```json
// Old (still works — "typescript" maps to "node" internally)
{ "type": "script", "runtime": "typescript", "script": "scripts/on_deal_won.ts" }

// New (preferred)
{ "type": "script", "runtime": "node", "script": "scripts/on_deal_won.ts" }
```

In `manager.go`:
```go
func (m *Manager) detectRuntime(script string, explicitRuntime string) string {
    if explicitRuntime != "" {
        // Map legacy names
        if explicitRuntime == "typescript" { return "node" }
        return explicitRuntime
    }
    // Auto-detect from extension
    if strings.HasSuffix(script, ".py") { return "python" }
    if strings.HasSuffix(script, ".go") { return "go" }
    return "node"  // default for .js, .ts
}
```

---

## 10. Critical Design Decisions

### 10.1 Dual-Pool Architecture: Worker + Background

**Problem**: One pool can't serve both fast scripts (validation, 30s max) and long scripts (crawling, hours). Long scripts starve short scripts.

**Decision: Two pools per runtime** — `worker` (fast, strict timeout) and `background` (long-running, relaxed timeout).

```yaml
# bitcode.yaml — UNIFIED pool config (applies to ALL runtimes)
runtime:
  worker:                          # fast scripts (API, webhook, cron)
    pool_size: 4                   # per runtime type (4 node + 4 python processes)
    default_step_timeout: "30s"
    max_step_timeout: "5m"
    max_process_timeout: "10m"
    max_executions: 1000
    max_memory_mb: 0               # 0 = unlimited (default). Overridable by module/process.
    hard_max_memory_mb: 0          # 0 = no ceiling (default). NOT overridable. max_memory_mb cannot exceed this.

  background:                      # long-running scripts (crawling, ML, migration)
    pool_size: 2
    default_step_timeout: "5m"
    max_step_timeout: "0"          # unlimited
    max_process_timeout: "0"       # unlimited
    max_executions: 100
    max_memory_mb: 0               # 0 = unlimited (default)
    hard_max_memory_mb: 0          # 0 = no ceiling (default)

  # Runtime-specific (only what's truly unique per runtime)
  node:
    enabled: "auto"
    # Engine auto-detection order:
    # 1. Check "bun" in PATH → use Bun (faster startup, native TS)
    # 2. Check "node" in PATH → use Node.js (most compatible)
    # 3. Neither found → disabled
    command: ""                    # empty = auto-detect (bun > node). Set "bun" or "node" to force.
    min_version: "20.0"           # for Node.js. Bun: "1.2.15". Auto-adjusted per engine.
```

**Why unified**: Developer thinks "is this fast or slow work?" — not "what's the Node.js timeout vs Python timeout?" One config, all runtimes. Crawling in Node.js and ML in Python both need the same thing: background pool with high memory limit.

**How developer selects pool** — resolution order: step → process → module → project → hardcoded:

```json
// Step level (most specific)
{ "type": "script", "runtime": "node", "script": "crawl.js", "pool": "background" }

// Process level
{ "name": "import_csv", "pool": "background", "steps": [...] }

// Module level (module.json)
{ "name": "crm", "default_pool": "background" }

// No override → default "worker"
```

**Memory override** — same resolution order:

```json
// Module level: all scripts in analytics module get 4GB
{ "name": "analytics", "max_memory_mb": 4096 }

// Process level: this specific training gets 6GB
{ "name": "train_model", "max_memory_mb": 6144, "steps": [...] }

// Capped by admin hard limit (bitcode.yaml: hard_max_memory_mb)
```

**Why two pools, not N:**
- `worker` covers 90% of use cases (API-triggered, webhooks, cron quick jobs)
- `background` covers 10% (migration, crawling, ML, batch processing)
- YAGNI — more pools can be added later if needed
- Proven pattern: Sidekiq (default + long_running), Celery (multiple queues)

```go
type DualPool struct {
    worker     *ProcessPool
    background *ProcessPool
}

func (d *DualPool) Execute(ctx context.Context, poolName string, ...) (any, error) {
    pool := d.worker
    if poolName == "background" {
        pool = d.background
    }
    
    proc := <-pool.available  // blocks if all busy
    defer func() { pool.available <- proc }()
    
    return proc.Execute(ctx, ...)
}
```

**Edge cases:**

| Scenario | Behavior |
|----------|----------|
| `pool: "background"` but background pool disabled | Fallback to worker pool, log warning, worker timeout applies |
| Background pool all busy | Queue — script waits until process available |
| Worker pool all busy | Queue — but short timeouts mean queue moves fast |
| Engine shutdown while background script running | Graceful: wait max 30s, then force kill, log warning |

### 10.2 Timeout Hierarchy (4 Layers)

**Problem**: Different steps in one process need different timeouts. Crawling = 90 minutes. Save to DB = 10 seconds. One timeout doesn't fit all.

**Decision: 4-layer timeout with override at every level.**

```
Layer 1: Engine global defaults (bitcode.yaml)     ← lowest priority
Layer 2: Pool defaults (bitcode.yaml per pool)      
Layer 3: Process level (process JSON)               
Layer 4: Step level (process JSON per step)         ← highest priority

All capped by:
  Pool max_step_timeout (hard limit per pool)
  Engine max_step_timeout (hard limit global)
  Remaining process time (if process has timeout)
```

**Example — crawl 1000 pages + save DB:**

```json
{
  "name": "crawl_and_save",
  "pool": "background",
  "timeout": "2h",
  "default_step_timeout": "10m",
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
      "name": "validate_data",
      "runtime": "node",
      "script": "scripts/validate.js"
    },
    {
      "type": "script",
      "name": "save_to_db",
      "runtime": "node",
      "script": "scripts/save.js",
      "timeout": "5m"
    }
  ]
}
```

**Resolution per step:**

```
Step "crawl_pages":
  step.timeout = 90m (explicitly set)
  capped by pool.max_step_timeout = 0 (unlimited) → no cap
  capped by process remaining = 2h at start → 90m < 2h ✅
  → Effective: 90m

Step "validate_data":
  step.timeout = not set
  process.default_step_timeout = 10m (process-level default)
  → Effective: min(10m, remaining process time)

Step "save_to_db":
  step.timeout = 5m (explicitly set)
  → Effective: min(5m, remaining process time)
```

**Resolution algorithm (Go) — unified for timeout and memory:**

```go
// resolveStepTimeout — same pattern for all overridable configs
func resolveStepTimeout(step StepDefinition, process ProcessDefinition,
    module ModuleDefinition, poolConfig PoolConfig, elapsed time.Duration) time.Duration {

    // Resolution: step → process → module → pool default → hardcoded
    timeout := step.Timeout
    if timeout == 0 { timeout = process.DefaultStepTimeout }
    if timeout == 0 { timeout = module.DefaultStepTimeout }
    if timeout == 0 { timeout = poolConfig.DefaultStepTimeout }
    if timeout == 0 { timeout = 30 * time.Second }

    // Cap by hard limit (admin, NOT overridable)
    if poolConfig.MaxStepTimeout > 0 && timeout > poolConfig.MaxStepTimeout {
        timeout = poolConfig.MaxStepTimeout
    }
    // Cap by remaining process time
    if process.Timeout > 0 {
        remaining := process.Timeout - elapsed
        if remaining <= 0 { return 0 }
        if timeout > remaining { timeout = remaining }
    }
    return timeout
}

// resolveMaxMemory — same resolution pattern
// 0 = unlimited (no memory-based recycling)
func resolveMaxMemory(process ProcessDefinition, module ModuleDefinition,
    poolConfig PoolConfig) int {

    // Resolution: process → module → pool default
    mem := process.MaxMemoryMB
    if mem == 0 { mem = module.MaxMemoryMB }
    if mem == 0 { mem = poolConfig.MaxMemoryMB }
    // If still 0 → unlimited (no memory-based recycling)

    // Cap by hard limit (admin, NOT overridable)
    // hard_max = 0 means no ceiling
    if poolConfig.HardMaxMemoryMB > 0 && mem > poolConfig.HardMaxMemoryMB {
        mem = poolConfig.HardMaxMemoryMB
    }
    return mem  // 0 = unlimited
}

// validateMemoryConfig — called at startup and when loading modules/processes
func validateMemoryConfig(maxMem, hardMax int) error {
    if hardMax > 0 && maxMem > 0 && maxMem > hardMax {
        return fmt.Errorf("max_memory_mb (%d) cannot exceed hard_max_memory_mb (%d)", maxMem, hardMax)
    }
    return nil
}

// resolvePool — same resolution pattern
func resolvePool(step StepDefinition, process ProcessDefinition,
    module ModuleDefinition) string {

    if step.Pool != "" { return step.Pool }
    if process.Pool != "" { return process.Pool }
    if module.DefaultPool != "" { return module.DefaultPool }
    return "worker"
}
```

**Default timeouts per step type:**

| Step Type | Default | Why |
|-----------|---------|-----|
| `script` | from config (30s) | Could be anything |
| `http` | 30s | Network call |
| `query`, `create`, `update`, `delete` | 10s | DB ops should be fast |
| `log`, `assign`, `emit` | 5s | Near-instant |
| `call` | inherits called process timeout | Recursive |
| `if`, `switch` | 5s | Condition evaluation |
| `loop` | same as process timeout | Could iterate many times |

**How timeout is enforced in Node.js:**

```javascript
// vm.runInNewContext timeout = step timeout (CPU time only)
vm.runInNewContext(code, sandbox, {
    filename: scriptPath,
    timeout: stepTimeoutMs
});
// Note: vm.timeout only limits CPU time.
// I/O waits (bridge calls) don't count against this timer.
// This is correct — we don't want to timeout during legitimate HTTP waits.
```

For total step timeout (including I/O): Go side uses `context.WithTimeout`:

```go
stepCtx, cancel := context.WithTimeout(ctx, stepTimeout)
defer cancel()

select {
case res := <-resultCh:
    return res
case <-stepCtx.Done():
    // Total step timeout (CPU + I/O)
    // For Node.js: process survives (vm.timeout handles CPU)
    // For Python: must kill process (no vm.timeout equivalent)
    return nil, &bridge.BridgeError{Code: "STEP_TIMEOUT", ...}
}
```

### 10.3 TypeScript Compilation

**Problem**: Scripts are `.ts` files but `vm.runInNewContext` only runs JavaScript. Who compiles?

**Options**:

| Option | How | Pros | Cons |
|--------|-----|------|------|
| **A. No TypeScript** | Only `.js` files | Simple | Developers expect TS |
| **B. Runtime transpile** | Use `esbuild` or `swc` to transpile before execute | Fast (esbuild: <10ms) | Extra dependency |
| **C. Pre-compile** | `bitcode build` CLI command compiles TS → JS | No runtime overhead | Extra build step |
| **D. Strip types only** | Node.js 22+ has `--experimental-strip-types` | Zero config | Requires Node.js 22+ |

**Decision: Option B — Runtime transpile with esbuild**.

Why:
- esbuild transpiles TypeScript in <10ms — negligible overhead
- No build step needed — developer writes `.ts`, engine handles it
- esbuild is a single binary, can be bundled
- Fallback: if file is `.js`, skip transpile

```javascript
// In runtime.js:
const esbuild = require('esbuild');

function loadScript(scriptPath) {
    const code = fs.readFileSync(scriptPath, 'utf-8');
    if (scriptPath.endsWith('.ts') || scriptPath.endsWith('.tsx')) {
        const result = esbuild.transformSync(code, {
            loader: 'ts',
            format: 'cjs',
            target: 'node18',
            sourcemap: 'inline',  // for stack traces
        });
        return result.code;
    }
    return code;
}
```

**Implication**: `esbuild` must be installed. Options:
- Bundle with engine: `plugins/node/node_modules/esbuild/` (pre-installed)
- Or: `bitcode setup` installs it
- Or: engine auto-installs on first TypeScript execution

### 10.4 Process Crash Recovery

**Problem**: If Node.js process crashes (OOM, unhandled exception, segfault), all subsequent script executions fail.

**Decision: Auto-restart with backoff**.

```go
func (pool *ProcessPool) monitorProcess(proc *PluginProcess) {
    err := proc.cmd.Wait()  // blocks until process exits
    
    if err != nil {
        log.Printf("[WARN] Node.js process crashed: %v", err)
    }
    
    // Remove from pool
    pool.remove(proc)
    
    // Restart with backoff
    backoff := time.Second
    for attempts := 0; attempts < 5; attempts++ {
        time.Sleep(backoff)
        newProc, err := pool.startProcess()
        if err == nil {
            pool.add(newProc)
            log.Printf("[INFO] Node.js process restarted successfully")
            return
        }
        backoff *= 2
    }
    log.Printf("[ERROR] Failed to restart Node.js process after 5 attempts")
}
```

### 10.5 Memory Management

**Problem**: Long-running Node.js process accumulates memory from script executions.

**Decision: Process recycling**.

```yaml
# Unified pool config (applies to all runtimes)
runtime:
  worker:
    max_executions: 1000    # restart process after N executions
    max_memory_mb: 0        # 0 = unlimited. Set to e.g. 512 to recycle when RSS exceeds.
  background:
    max_executions: 100
    max_memory_mb: 0
```

```go
func (proc *PluginProcess) shouldRecycle() bool {
    if proc.executionCount >= proc.config.MaxExecutions {
        return true
    }
    // Check RSS via /proc/[pid]/status or os.Process
    // maxMem = 0 means unlimited (no memory-based recycling)
    if proc.config.MaxMemoryMB > 0 && proc.getMemoryMB() > proc.config.MaxMemoryMB {
        return true
    }
    return false
}
```

After recycling: gracefully drain (finish current execution), then kill and replace.

### 10.6 Binary Data over JSON-RPC

**Problem**: JSON doesn't support binary data. `bitcode.storage.upload({ content: buffer })` needs to send binary.

**Decision: Base64 encoding for binary fields**.

```javascript
// Node.js side:
bitcode.storage.upload({
    filename: "report.pdf",
    content: buffer  // Buffer object
});

// Before sending via JSON-RPC, runtime.js converts:
// content: Buffer → content: { _type: "binary", data: base64string }

// Go side:
// Detects _type: "binary", decodes base64 → []byte
```

```javascript
// In runtime.js, before sending bridge_request:
function serializeParams(params) {
    return JSON.parse(JSON.stringify(params, (key, value) => {
        if (Buffer.isBuffer(value)) {
            return { _type: 'binary', encoding: 'base64', data: value.toString('base64') };
        }
        return value;
    }));
}

// In Go bridge_handler.go, when receiving:
func decodeBinaryFields(params map[string]any) {
    for key, val := range params {
        if m, ok := val.(map[string]any); ok {
            if m["_type"] == "binary" {
                decoded, _ := base64.StdEncoding.DecodeString(m["data"].(string))
                params[key] = decoded
            }
        }
    }
}
```

**Size concern**: Base64 adds ~33% overhead. For large files (>10MB), consider streaming instead of buffering. But for Phase 2, base64 is sufficient. Streaming can be added later.

### 10.7 Console Output

**Problem**: `console.log()` in scripts goes to Node.js stderr. Developer can't see it.

**Decision: Intercept and route to engine logger**.

```javascript
// In runtime.js sandbox:
const scriptConsole = {
    log: (...args) => bridgeCall('log', { level: 'info', msg: args.map(String).join(' ') }),
    warn: (...args) => bridgeCall('log', { level: 'warn', msg: args.map(String).join(' ') }),
    error: (...args) => bridgeCall('log', { level: 'error', msg: args.map(String).join(' ') }),
    debug: (...args) => bridgeCall('log', { level: 'debug', msg: args.map(String).join(' ') }),
};

const sandbox = {
    console: scriptConsole,  // intercept console
    // ...
};
```

Now `console.log("hello")` in script → bridge RPC → Go logger → structured log with module/script context.

### 10.8 Hot Reload

**Problem**: Developer edits script file. Does it take effect immediately?

**Current behavior**: `fs.readFileSync(scriptPath)` every execution → **yes, hot reload works** for the script itself.

**But**: `require('some-module')` is cached by Node.js module system. If developer edits a required module, it won't reload.

**Decision**: For Phase 2, accept this limitation. Document it:
- Script files: hot reload ✅ (re-read every execution)
- Required npm packages: no hot reload ❌ (Node.js module cache)
- To reload npm packages: restart engine or recycle process pool

### 10.9 Bun Support (Auto-Detected)

Engine auto-detects Bun vs Node.js at startup:

```go
func detectJSEngine(forceCommand string) (command string, engine string, err error) {
    // Force override — skip auto-detect
    if forceCommand != "" {
        path, err := exec.LookPath(forceCommand)
        if err != nil {
            return "", "", fmt.Errorf("%s not found in PATH", forceCommand)
        }
        if strings.Contains(forceCommand, "bun") {
            return path, "bun", nil
        }
        return path, "nodejs", nil
    }
    
    // Auto-detect: Bun first (preferred — faster, TS native)
    if path, err := exec.LookPath("bun"); err == nil {
        version := getVersion(path, "--version")
        if semver.Compare(version, "1.2.15") >= 0 {  // vm.runInNewContext fixed in 1.2.15
            log.Printf("[INFO] Bun %s detected, using as JS runtime (faster, native TS)", version)
            return path, "bun", nil
        }
        log.Printf("[WARN] Bun %s found but 1.2.15+ required for vm support. Skipping.", version)
    }
    
    // Fallback: Node.js
    if path, err := exec.LookPath("node"); err == nil {
        version := getVersion(path, "--version")
        if semver.Compare(version, "20.0.0") >= 0 {
            log.Printf("[INFO] Node.js %s detected, using as JS runtime", version)
            return path, "nodejs", nil
        }
        log.Printf("[WARN] Node.js %s found but 20.0+ required. Skipping.", version)
    }
    
    return "", "", fmt.Errorf("neither Bun (1.2.15+) nor Node.js (20+) found in PATH")
}
```

**Auto-detect order**: Bun → Node.js → disabled.
**Force override**: `command: "bun"` or `command: "node"` skips auto-detect.

**Differences when running under Bun:**

| Aspect | Node.js | Bun |
|--------|---------|-----|
| TypeScript | esbuild transpile needed | Native — no transpile |
| Startup | ~50ms | ~5ms |
| `vm.runInNewContext` | Full support | Supported (since v1.2.15) |
| npm packages | Full | Full |
| `vm.timeout` | Supported | Check Bun version |

**TypeScript handling in runtime.js:**

```javascript
function loadScript(scriptPath) {
    const code = fs.readFileSync(scriptPath, 'utf-8');
    if (scriptPath.endsWith('.ts')) {
        if (typeof Bun !== 'undefined') {
            return code;  // Bun runs TS natively
        }
        const esbuild = require('esbuild');
        return esbuild.transformSync(code, { loader: 'ts', format: 'cjs', sourcemap: 'inline' }).code;
    }
    return code;
}
```

**Override auto-detection:**

```yaml
runtime:
  node:
    command: "node"    # force Node.js even if Bun is available
    # command: "bun"   # force Bun even if Node.js is preferred
```

### 10.10 Script File Auto-Detection

File extension determines default runtime. `runtime` field in process JSON is optional — only needed to override:

```
.ts  → default: "node" (TypeScript needs Bun native or esbuild)
.js  → default: "javascript" (embedded goja, lightest)
.py  → default: "python"
.go  → default: "go" (yaegi)
```

**Override examples:**

```json
// .js file but want Node.js (need npm packages)
{ "type": "script", "script": "scripts/crawl.js", "runtime": "node" }

// .js file but want QuickJS (need async/await in embedded)
{ "type": "script", "script": "scripts/complex.js", "runtime": "javascript:quickjs" }
```

**Validation — incompatible combinations:**

```
.ts + runtime: "javascript"       → ERROR: "TypeScript requires runtime 'node'. Embedded JS runtimes don't support TypeScript."
.ts + runtime: "javascript:goja"  → ERROR: same
.ts + runtime: "python"           → ERROR: "cannot run .ts file with python runtime"
.js + runtime: "python"           → ERROR: "cannot run .js file with python runtime"
.py + runtime: "node"             → ERROR: "cannot run .py file with node runtime"
.go + runtime: "javascript"       → ERROR: "cannot run .go file with javascript runtime"
```

### 10.11 Node.js/Bun Not Installed

**Problem**: Engine starts but Node.js is not installed. What happens?

**Decision: Graceful degradation with clear errors**.

```go
func (a *App) startPluginRuntimes() {
    if err := a.PluginManager.StartNodePool(); err != nil {
        log.Printf("[WARN] Node.js runtime not available: %v", err)
        log.Printf("[WARN] Scripts with runtime 'node' will fail. Install Node.js to enable.")
        // Engine continues — other runtimes (goja, yaegi) still work
    }
}

// When script with runtime: "node" is triggered:
func (m *Manager) Execute(...) (any, error) {
    if !m.IsRunning("node") {
        return nil, &bridge.BridgeError{
            Code:    "RUNTIME_NOT_AVAILABLE",
            Message: "Node.js runtime is not available. Install Node.js and restart the engine.",
        }
    }
    // ...
}
```

---

## 11. Edge Cases

### 11.1 Communication

| Edge Case | Behavior |
|-----------|----------|
| Node.js process crashes mid-execution | Go detects broken pipe, returns error, restarts process |
| Bridge request timeout (Go takes too long) | Node.js has per-request timeout (30s default). Rejects promise. |
| Multiple concurrent script executions | Each execution has unique `execID`. Messages are routed by ID. Go side uses mutex per plugin process. |
| Node.js sends malformed JSON | Go logs warning, ignores message |
| Go sends bridge_response for unknown ID | Node.js logs warning, ignores |
| Script calls 1000 bridge methods | All work. Each is a separate RPC. Performance: ~0.1ms per RPC (local pipe). |

### 11.2 Script Execution

| Edge Case | Behavior |
|-----------|----------|
| Script has syntax error | `vm.runInNewContext` throws. Caught, returned as `execute_error`. |
| Script throws unhandled error | Caught by try/catch in `executeScript`. Returned as `execute_error`. |
| Script runs forever (infinite loop) | `vm.runInNewContext` has `timeout: 30000`. Throws after 30s. |
| Script calls `process.exit()` | Not available in sandbox (not exposed). |
| Script calls `require('fs')` | Works — Node.js built-in. But `bitcode.fs` is preferred (sandboxed). |
| Script calls `require('child_process')` | Works — Node.js built-in. Security risk. Consider blocking in future. |

### 11.3 Dependencies

| Edge Case | Behavior |
|-----------|----------|
| `require('axios')` but not installed | Error: "Cannot find module 'axios'" |
| Two modules need different axios versions | Each has own `node_modules/`. No conflict. |
| Module has no `package.json` | `require()` falls back to project-level `node_modules/` then Node.js built-ins |
| `npm install` fails (network error) | CLI shows error. Module works without npm packages (bridge methods still available). |

### 11.4 Transaction

| Edge Case | Behavior |
|-----------|----------|
| Script calls `bitcode.tx()` | `tx.begin` → Go creates GORM transaction, stores in map. All bridge calls with `txId` use tx DB. |
| Error inside `bitcode.tx()` callback | `tx.rollback` sent automatically. Error propagated to script. |
| Script forgets to await inside `bitcode.tx()` | Transaction may commit before async operations complete. Document: always `await` inside `tx()`. |
| Transaction timeout (30s) | Go auto-rollbacks. Node.js receives error. |
| Nested `bitcode.tx()` | Go uses savepoints. Inner rollback doesn't affect outer. |

### 11.5 Performance

| Operation | Expected Latency |
|-----------|-----------------|
| Bridge RPC round-trip (Go ↔ Node.js via pipe) | ~0.05-0.2ms |
| `bitcode.model("lead").search({})` (10 records) | ~1-5ms (RPC + DB query) |
| `bitcode.http.get("https://...")` | ~100-5000ms (network) |
| Script with 10 bridge calls | ~10-50ms total RPC overhead |
| Script with 100 bridge calls | ~50-200ms total RPC overhead |

Pipe-based IPC is fast. The bottleneck is always the actual operation (DB query, HTTP request), not the RPC.

---

## 12. Implementation Tasks

### Files to Create

```
engine/plugins/node/
├── runtime.js              # Complete rewrite — bidirectional JSON-RPC + bitcode proxy
├── package.json            # esbuild dependency for TypeScript transpilation

engine/internal/runtime/plugin/
├── bridge_handler.go       # handleBridgeRequest() — routes bridge methods to bridge.Context
├── message.go              # Message types (execute, bridge_request, bridge_response, etc.)
├── pool.go                 # ProcessPool — manages N Node.js processes
```

### Files to Modify

```
engine/internal/runtime/plugin/manager.go
  → Rewrite Execute() for bidirectional communication
  → Add message loop (handle bridge_request while waiting for execute_complete)
  → Replace single process with ProcessPool
  → Rename StartTypescript() → StartNodePool()
  → Map "typescript" → "node" in detectRuntime()
  → Add process crash recovery + auto-restart
  → Add process recycling (max_executions, max_memory_mb)

engine/internal/runtime/executor/steps/script.go
  → Add BridgeFactory to ScriptHandler
  → Create bridge.Context per execution
  → Pass to Runner.RunWithBridge()

engine/internal/app.go
  → Update plugin startup: StartNodePool() instead of StartTypescript()
  → Wire BridgeFactory into ScriptHandler
  → Add PoolConfig from bitcode.yaml

engine/internal/config.go
  → Add RuntimeConfig with node pool settings

engine/cmd/bitcode/main.go (or equivalent CLI)
  → Add `bitcode module install-deps <name>` command
```

### Files to Delete/Move

```
engine/plugins/typescript/index.js → engine/plugins/node/runtime.js (move + rewrite)
```

### Task Breakdown

| # | Task | Effort | Priority |
|---|------|--------|----------|
| **Core Protocol** | | | |
| 1 | Define message types (Go structs + JS objects) | Small | Must |
| 2 | Rewrite `runtime.js` — bidirectional JSON-RPC + bitcode proxy | Large | Must |
| 3 | Rewrite `manager.go` Execute() — message loop, bridge request handling | Large | Must |
| 4 | Implement `bridge_handler.go` — route 50+ bridge methods to bridge.Context | Large | Must |
| 5 | Update `script.go` — create bridge.Context, pass to runner | Medium | Must |
| 6 | Update `app.go` — wire BridgeFactory, rename StartNodePool | Small | Must |
| **Process Management** | | | |
| 7 | Implement `pool.go` — ProcessPool with configurable size | Medium | Must |
| 8 | Implement process crash recovery + auto-restart with backoff | Medium | Must |
| 9 | Implement process recycling (max_executions, max_memory_mb) | Medium | Should |
| 10 | Add RuntimeConfig (pool_size, max_memory, etc.) to config.go | Small | Must |
| **TypeScript & Module** | | | |
| 11 | Add esbuild transpilation for `.ts` files in runtime.js | Medium | Must |
| 12 | Bundle esbuild in `plugins/node/package.json` | Small | Must |
| 13 | Implement `Module.createRequire()` for per-module resolution | Small | Must |
| 14 | Add `bitcode module install-deps` CLI command | Medium | Must |
| **Compatibility** | | | |
| 15 | Add `definePlugin` shim for backward compatibility | Small | Must |
| 16 | Map `runtime: "typescript"` → `"node"` in detectRuntime | Small | Must |
| 17 | Intercept `console.log/warn/error` → route to engine logger via bridge | Small | Must |
| 18 | Implement base64 encoding/decoding for binary data in JSON-RPC | Small | Must |
| **Transaction** | | | |
| 19 | Implement transaction handling (tx.begin/commit/rollback in Go + JS) | Medium | Must |
| **Migration** | | | |
| 20 | Update 6 TypeScript scripts in samples/erp to new style | Medium | Must |
| **Tests** | | | |
| 21 | Tests: bidirectional RPC (send execute, handle bridge requests, receive result) | Large | Must |
| 22 | Tests: all 20 bridge namespaces via RPC (model, db, http, cache, etc.) | Large | Must |
| 23 | Tests: per-module require resolution | Medium | Must |
| 24 | Tests: backward compatibility (definePlugin, old ctx pattern) | Small | Must |
| 25 | Tests: transaction via RPC (begin, bridge calls with txId, commit/rollback) | Medium | Must |
| 26 | Tests: error propagation (BridgeError codes through RPC) | Medium | Must |
| 27 | Tests: process pool (round-robin, queue when full, drain) | Medium | Must |
| 28 | Tests: process crash recovery + auto-restart | Medium | Must |
| 29 | Tests: TypeScript transpilation via esbuild | Small | Must |
| 30 | Tests: binary data (base64 encode/decode through RPC) | Small | Must |
| 31 | Tests: console.log interception | Small | Should |
| 32 | Tests: concurrent executions across pool | Medium | Should |
| 33 | Tests: process recycling (max_executions trigger) | Small | Should |
| 34 | Tests: Node.js not installed (graceful degradation) | Small | Should |
| 35 | Tests: script timeout (30s vm timeout) | Small | Should |
