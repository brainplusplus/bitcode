# Phase 3: Fix Python Child Process + venv

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: Phase 1 (bridge API interfaces), Phase 1.5 (multi-tenancy), Phase 2 (bidirectional JSON-RPC protocol — reused)
**Unlocks**: Phase 6C (engine enhancements), Phase 7 (module setting)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Python vs Node.js — Key Differences](#2-python-vs-nodejs--key-differences)
3. [Current State — What's Wrong](#3-current-state--whats-wrong)
4. [Architecture — Reuse Phase 2 Protocol](#4-architecture--reuse-phase-2-protocol)
5. [Python Runtime Rewrite](#5-python-runtime-rewrite)
6. [Bridge Proxy — Sync vs Async](#6-bridge-proxy--sync-vs-async)
7. [Per-Module venv Isolation](#7-per-module-venv-isolation)
8. [pip Package Management (Step-by-Step)](#8-pip-package-management-step-by-step)
9. [Script Signature Migration](#9-script-signature-migration)
10. [Critical Design Decisions](#10-critical-design-decisions)
11. [Edge Cases](#11-edge-cases)
12. [Implementation Tasks](#12-implementation-tasks)

---

## 1. Goal

Make the Python child process runtime **actually work** — scripts can call all 20 `bitcode.*` bridge methods with real results, not stubs.

### Prerequisites

- **Python 3.10+** (minimum). Recommended: Python 3.12+.
  - Python 3.9: EOL October 2025 — not supported
  - Python 3.10: supported until October 2026
  - Python 3.11: supported until October 2027
  - Python 3.12: recommended, supported until October 2028
  - Why 3.10+: match expressions (`match/case`), improved error messages, `typing` improvements
  - Python 2: **not supported**
- **pip** (bundled with Python 3.4+)
- **venv** (bundled with Python 3.3+)
- **Optional**: Python runtime is NOT required to run the engine. Same `enabled: "auto"` pattern as Node.js.

### 1.1 Runtime Configuration

```yaml
# bitcode.yaml — UNIFIED pool config (applies to ALL runtimes)
runtime:
  worker:                          # fast scripts — same for node, python, goja, qjs, yaegi
    pool_size: 4
    default_step_timeout: "30s"
    max_step_timeout: "5m"
    max_process_timeout: "10m"
    max_executions: 1000
    max_memory_mb: 0               # 0 = unlimited (default). Overridable by module/process.
    hard_max_memory_mb: 0          # 0 = no ceiling (default). NOT overridable. max_memory_mb cannot exceed this.

  background:                      # long-running scripts — same for all
    pool_size: 2
    default_step_timeout: "5m"
    max_step_timeout: "0"          # unlimited
    max_process_timeout: "0"       # unlimited
    max_executions: 100
    max_memory_mb: 0               # 0 = unlimited (default)
    hard_max_memory_mb: 0          # 0 = no ceiling (default)

  # Python-specific (only what's truly unique)
  python:
    enabled: "auto"
    command: "python3"
    min_version: "3.10.0"
```

**Why unified**: Pool config is the same for all runtimes. Developer thinks "is this fast or slow work?" — not "what's the Python memory limit vs Node.js memory limit?" Crawling in Node.js and ML in Python both need background pool with high memory. One config covers both.

### 1.2 Success Criteria

- All 6 Python scripts in `samples/erp` execute with real DB operations
- `bitcode.model("lead").search({...})` returns actual records from database
- Per-module `requirements.txt` + `.venv/` isolation works
- `import` resolves from module's venv, not system Python
- Backward compatible: old-style scripts (`def execute(params)`) still work
- Error contract: `BridgeError` raised on failures with correct codes

### 1.3 What This Phase Does NOT Do

- Does not add goja or yaegi (Phase 4-5)
- Does not change process engine or view system (Phase 6C)
- Does not build module setting (Phase 7)

---

## 2. Python vs Node.js — Key Differences

Phase 2 (Node.js) and Phase 3 (Python) share the same bidirectional JSON-RPC protocol. But Python has fundamental differences that affect implementation:

| Aspect | Node.js (Phase 2) | Python (Phase 3) |
|--------|-------------------|------------------|
| **Default execution model** | Async (event loop, Promises) | Sync (blocking by default) |
| **Bridge calls** | `await bitcode.model("lead").search({})` | `bitcode.model("lead").search({})` — blocking |
| **Async support** | Native `async/await` | `asyncio` exists but not default |
| **Dependency isolation** | `node_modules/` per module | `.venv/` per module (heavier) |
| **Package manager** | npm | pip |
| **Package file** | `package.json` | `requirements.txt` |
| **Lock file** | `package-lock.json` | `requirements.txt` IS the lock (or use `pip freeze`) |
| **Import system** | `require()` — simple path resolution | `import` — `sys.path`, `__init__.py`, complex |
| **Transpilation** | TypeScript → JS via esbuild | Not needed |
| **GIL** | No GIL — true async | GIL — one thread at a time |
| **Process pool importance** | Important for throughput | **Critical** — GIL makes single process a hard bottleneck |
| **Memory behavior** | V8 GC, generally stable | CPython can leak, fragmentation common |
| **Startup time** | ~50ms | ~100-200ms (heavier stdlib) |

### Key Implication: Bridge Calls Must Be Synchronous

In Node.js:
```javascript
// Natural — async/await is the default
const leads = await bitcode.model("lead").search({});
```

In Python:
```python
# Must be synchronous — Python scripts don't use asyncio by default
leads = bitcode.model("lead").search({})  # blocks until Go responds
```

This means the Python runtime must handle the bidirectional RPC **within a synchronous call**. The script thread blocks while waiting for Go's response. This is implemented using `threading.Event` (see §6).

---

## 3. Current State — What's Wrong

Same problems as Node.js (Phase 2 §2), plus Python-specific issues:

| # | Problem | Impact |
|---|---------|--------|
| 1 | **Bridge is nonexistent** — Python scripts only receive `params`, no `ctx`/`bitcode` at all | Scripts can only do pure computation |
| 2 | **One-way communication** — same as Node.js | Bridge methods can't work |
| 3 | **No venv** — uses system Python | Package conflicts, pollutes system |
| 4 | **No `requirements.txt` support** — no mechanism to declare/install dependencies | Manual pip install to system Python |
| 5 | **Import resolution from system** — not from module directory | Module-level packages impossible |
| 6 | **No error structure** — errors are plain strings | No `BridgeError` |
| 7 | **Module caching** — Python caches imported modules in `sys.modules` | Edited modules don't reload |
| 8 | **No version check** — could run on Python 2 | Undefined behavior |

---

## 4. Architecture — Reuse Phase 2 Protocol

> **Cross-Runtime Execution**: A single process can mix Node.js and Python steps (and goja/yaegi). Data flows via JSON between steps. See master doc §4.4 for full documentation.

The bidirectional JSON-RPC protocol from Phase 2 is **identical** for Python. Same message types, same bridge method mapping, same transaction handling.

```
Go Engine                          Python Process
    │                                    │
    │ ── { type: "execute", ... }        │
    │ ──────────────────────────────────► │
    │                                    │ Load script, create bitcode proxy
    │                                    │ script.execute(bitcode, params)
    │                                    │
    │                                    │ # Script calls bitcode.model("lead").search(...)
    │ ◄────────────────────────────────── │
    │    { type: "bridge_request", ... } │
    │                                    │
    │ Go executes bridge method          │
    │                                    │
    │ ── { type: "bridge_response", ... }│
    │ ──────────────────────────────────► │
    │                                    │ # Script receives result, continues
    │                                    │
    │ ◄────────────────────────────────── │
    │    { type: "execute_complete" }    │
```

**What's shared with Phase 2:**
- Message type definitions (Go structs in `message.go`)
- Bridge handler (Go side `bridge_handler.go` — same for all runtimes)
- Process pool pattern (`pool.go` — parameterized for any command)
- Transaction protocol (tx.begin/commit/rollback)

**What's Python-specific:**
- `runtime.py` rewrite (Python side)
- Synchronous bridge proxy (threading-based)
- venv management (instead of node_modules)
- Import path manipulation (instead of Module.createRequire)

---

## 5. Python Runtime Rewrite

### 5.1 New File: `plugins/python/runtime.py`

```python
#!/usr/bin/env python3
"""BitCode Python Runtime — Bidirectional JSON-RPC bridge."""

import sys
import json
import threading
import importlib.util
import traceback
import os

# --- Communication Layer ---

_stdout_lock = threading.Lock()
_pending_requests = {}  # id -> threading.Event + result holder
_next_bridge_id = 0
_bridge_id_lock = threading.Lock()

# Reader thread: reads messages from Go, routes to pending requests
_incoming_queue = {}  # id -> response message


def send_to_go(message):
    """Send JSON message to Go via stdout."""
    with _stdout_lock:
        sys.stdout.write(json.dumps(message) + "\n")
        sys.stdout.flush()


def bridge_call(method, params, tx_id=None):
    """
    Send bridge request to Go and BLOCK until response arrives.
    This is called from the script thread (synchronous).
    The reader thread will receive the response and signal us.
    """
    global _next_bridge_id
    with _bridge_id_lock:
        _next_bridge_id += 1
        req_id = _next_bridge_id

    event = threading.Event()
    _pending_requests[req_id] = {"event": event, "result": None, "error": None}

    msg = {"type": "bridge_request", "id": req_id, "method": method, "params": params}
    if tx_id:
        msg["txId"] = tx_id
    send_to_go(msg)

    # Block until Go responds (reader thread will set the event)
    event.wait(timeout=60)  # 60s timeout for bridge calls

    entry = _pending_requests.pop(req_id, None)
    if entry is None or not event.is_set():
        raise BridgeError("BRIDGE_TIMEOUT", f"Bridge call '{method}' timed out after 60s")

    if entry["error"]:
        err = entry["error"]
        raise BridgeError(
            err.get("code", "INTERNAL_ERROR"),
            err.get("message", "Unknown error"),
            err.get("details"),
            err.get("retryable", False),
        )

    return entry["result"]


# --- Error Type ---

class BridgeError(Exception):
    """Structured error from bridge operations."""
    def __init__(self, code, message, details=None, retryable=False):
        super().__init__(message)
        self.code = code
        self.details = details
        self.retryable = retryable


# --- Bridge Proxy (bitcode.*) ---

class ModelHandle:
    """Proxy for bitcode.model(name) operations."""

    def __init__(self, model_name, sudo=False, tenant=None, skip_val=False):
        self._model = model_name
        self._sudo = sudo
        self._tenant = tenant
        self._skip_val = skip_val

    def _params(self, **kwargs):
        p = {"model": self._model, **kwargs}
        if self._sudo:
            p["sudo"] = True
        if self._tenant:
            p["tenant"] = self._tenant
        if self._skip_val:
            p["skipValidation"] = True
        return p

    # Single record CRUD
    def search(self, opts=None):
        return bridge_call("model.search", self._params(opts=opts or {}))

    def get(self, id, opts=None):
        return bridge_call("model.get", self._params(id=id, opts=opts))

    def create(self, data):
        return bridge_call("model.create", self._params(data=data))

    def write(self, id, data):
        return bridge_call("model.write", self._params(id=id, data=data))

    def delete(self, id):
        return bridge_call("model.delete", self._params(id=id))

    def count(self, opts=None):
        return bridge_call("model.count", self._params(opts=opts or {}))

    def sum(self, field, opts=None):
        return bridge_call("model.sum", self._params(field=field, opts=opts or {}))

    def upsert(self, data, unique):
        return bridge_call("model.upsert", self._params(data=data, unique=unique))

    # Bulk operations
    def create_many(self, records):
        return bridge_call("model.createMany", self._params(records=records))

    def write_many(self, ids, data):
        return bridge_call("model.writeMany", self._params(ids=ids, data=data))

    def delete_many(self, ids):
        return bridge_call("model.deleteMany", self._params(ids=ids))

    def upsert_many(self, records, unique):
        return bridge_call("model.upsertMany", self._params(records=records, unique=unique))

    # Relation operations
    def add_relation(self, id, field, related_ids):
        return bridge_call("model.addRelation", self._params(id=id, field=field, relatedIds=related_ids))

    def remove_relation(self, id, field, related_ids):
        return bridge_call("model.removeRelation", self._params(id=id, field=field, relatedIds=related_ids))

    def set_relation(self, id, field, related_ids):
        return bridge_call("model.setRelation", self._params(id=id, field=field, relatedIds=related_ids))

    def load_relation(self, id, field):
        return bridge_call("model.loadRelation", self._params(id=id, field=field))

    # Mode switching
    def sudo(self):
        return SudoModelHandle(self._model)


class SudoModelHandle(ModelHandle):
    """ModelHandle with sudo mode — bypasses permissions."""

    def __init__(self, model_name, tenant=None, skip_val=False):
        super().__init__(model_name, sudo=True, tenant=tenant, skip_val=skip_val)

    def hard_delete(self, id):
        return bridge_call("model.hardDelete", self._params(id=id))

    def hard_delete_many(self, ids):
        return bridge_call("model.hardDeleteMany", self._params(ids=ids))

    def with_tenant(self, tenant_id):
        return SudoModelHandle(self._model, tenant=tenant_id, skip_val=self._skip_val)

    def skip_validation(self):
        return SudoModelHandle(self._model, tenant=self._tenant, skip_val=True)


class DBProxy:
    def query(self, sql, *args):
        return bridge_call("db.query", {"sql": sql, "args": list(args)})

    def execute(self, sql, *args):
        return bridge_call("db.execute", {"sql": sql, "args": list(args)})


class HTTPProxy:
    def get(self, url, **opts):
        return bridge_call("http.request", {"method": "GET", "url": url, **opts})

    def post(self, url, **opts):
        return bridge_call("http.request", {"method": "POST", "url": url, **opts})

    def put(self, url, **opts):
        return bridge_call("http.request", {"method": "PUT", "url": url, **opts})

    def patch(self, url, **opts):
        return bridge_call("http.request", {"method": "PATCH", "url": url, **opts})

    def delete(self, url, **opts):
        return bridge_call("http.request", {"method": "DELETE", "url": url, **opts})


class CacheProxy:
    def get(self, key):
        return bridge_call("cache.get", {"key": key})

    def set(self, key, value, ttl=None):
        opts = {"key": key, "value": value}
        if ttl:
            opts["ttl"] = ttl
        return bridge_call("cache.set", opts)

    def delete(self, key):
        return bridge_call("cache.del", {"key": key})


class FSProxy:
    def read(self, path):
        return bridge_call("fs.read", {"path": path})

    def write(self, path, content):
        return bridge_call("fs.write", {"path": path, "content": content})

    def exists(self, path):
        return bridge_call("fs.exists", {"path": path})

    def list(self, path):
        return bridge_call("fs.list", {"path": path})

    def mkdir(self, path):
        return bridge_call("fs.mkdir", {"path": path})

    def remove(self, path):
        return bridge_call("fs.remove", {"path": path})


class EmailProxy:
    def send(self, **opts):
        return bridge_call("email.send", opts)


class NotifyProxy:
    def send(self, **opts):
        return bridge_call("notify.send", opts)

    def broadcast(self, channel, data):
        return bridge_call("notify.broadcast", {"channel": channel, "data": data})


class StorageProxy:
    def upload(self, **opts):
        return bridge_call("storage.upload", opts)

    def url(self, id):
        return bridge_call("storage.url", {"id": id})

    def download(self, id):
        return bridge_call("storage.download", {"id": id})

    def delete(self, id):
        return bridge_call("storage.delete", {"id": id})


class SecurityProxy:
    def permissions(self, model):
        return bridge_call("security.permissions", {"model": model})

    def has_group(self, group):
        return bridge_call("security.hasGroup", {"group": group})

    def groups(self):
        return bridge_call("security.groups", {})


class AuditProxy:
    def log(self, **opts):
        return bridge_call("audit.log", opts)


class CryptoProxy:
    def encrypt(self, text):
        return bridge_call("crypto.encrypt", {"text": text})

    def decrypt(self, text):
        return bridge_call("crypto.decrypt", {"text": text})

    def hash(self, value):
        return bridge_call("crypto.hash", {"value": value})

    def verify(self, value, hash_str):
        return bridge_call("crypto.verify", {"value": value, "hash": hash_str})


class ExecutionProxy:
    def search(self, **opts):
        return bridge_call("execution.search", opts)

    def get(self, id, **opts):
        return bridge_call("execution.get", {"id": id, **opts})

    def current(self):
        return bridge_call("execution.current", {})

    def retry(self, id):
        return bridge_call("execution.retry", {"id": id})

    def cancel(self, id):
        return bridge_call("execution.cancel", {"id": id})


class BitcodeContext:
    """Main bridge context — exposed as 'bitcode' to scripts."""

    def __init__(self, session):
        self.session = session
        self.db = DBProxy()
        self.http = HTTPProxy()
        self.cache = CacheProxy()
        self.fs = FSProxy()
        self.email = EmailProxy()
        self.notify = NotifyProxy()
        self.storage = StorageProxy()
        self.security = SecurityProxy()
        self.audit = AuditProxy()
        self.crypto = CryptoProxy()
        self.execution = ExecutionProxy()

    def model(self, name):
        return ModelHandle(name)

    def env(self, key):
        return bridge_call("env.get", {"key": key})

    def config(self, key):
        return bridge_call("config.get", {"key": key})

    def log(self, level, msg, data=None):
        bridge_call("log", {"level": level, "msg": msg, "data": data})

    def emit(self, event, data=None):
        return bridge_call("emit", {"event": event, "data": data or {}})

    def call(self, process, input_data=None):
        return bridge_call("call", {"process": process, "input": input_data or {}})

    def exec(self, cmd, args=None, **opts):
        return bridge_call("exec", {"cmd": cmd, "args": args or [], **opts})

    def t(self, key):
        return bridge_call("t", {"key": key})

    def tx(self, fn):
        """Execute fn inside a database transaction."""
        result = bridge_call("tx.begin", {})
        tx_id = result["txId"]
        # TODO: create tx-scoped context that passes tx_id to all bridge calls
        try:
            ret = fn(self)  # for now, pass self — tx routing via tx_id
            bridge_call("tx.commit", {"txId": tx_id})
            return ret
        except Exception:
            bridge_call("tx.rollback", {"txId": tx_id})
            raise


# --- Script Execution ---

def execute_script(script_path, params, module_name, session, security_rules):
    """Load and execute a Python script with bitcode bridge."""
    bitcode = BitcodeContext(session)

    if not os.path.exists(script_path):
        raise FileNotFoundError(f"script not found: {script_path}")

    # Add module directory to sys.path for imports
    module_dir = os.path.join("modules", module_name) if module_name else os.path.dirname(script_path)
    venv_site = os.path.join(module_dir, ".venv", "lib", f"python{sys.version_info.major}.{sys.version_info.minor}", "site-packages")

    # Prepend module paths to sys.path (module-level packages first)
    paths_to_add = []
    if os.path.isdir(venv_site):
        paths_to_add.append(venv_site)
    if os.path.isdir(module_dir):
        paths_to_add.append(module_dir)

    for p in reversed(paths_to_add):
        if p not in sys.path:
            sys.path.insert(0, p)

    # Load script as module
    spec = importlib.util.spec_from_file_location("bitcode_script", script_path)
    if spec is None:
        raise ImportError(f"cannot load script: {script_path}")

    mod = importlib.util.module_from_spec(spec)

    # Invalidate cached module (hot reload)
    if "bitcode_script" in sys.modules:
        del sys.modules["bitcode_script"]

    spec.loader.exec_module(mod)

    # Call execute function
    if hasattr(mod, "execute"):
        func = mod.execute
        # Detect signature: execute(params) vs execute(bitcode, params)
        import inspect
        sig = inspect.signature(func)
        param_count = len(sig.parameters)

        if param_count >= 2:
            return func(bitcode, params)  # new style
        else:
            return func(params)  # legacy style
    elif hasattr(mod, "main"):
        return mod.main(params)  # alternative legacy
    else:
        return {"executed": True, "script": script_path}


# --- Message Loop (Main Thread + Reader Thread) ---

def message_router():
    """
    Single reader thread that reads ALL messages from stdin and routes them.
    
    This solves the race condition: only ONE thread reads stdin.
    Messages are routed by type:
    - "execute" → put in execute_queue for main thread
    - "bridge_response" → route to pending bridge_call() via threading.Event
    
    This is the same pattern as Node.js readline — one reader, multiple consumers.
    """
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            message = json.loads(line)
        except json.JSONDecodeError:
            continue

        msg_type = message.get("type")

        if msg_type == "bridge_response":
            # Go responded to a bridge request from script
            req_id = message.get("id")
            entry = _pending_requests.get(req_id)
            if entry:
                if "error" in message and message["error"]:
                    entry["error"] = message["error"]
                else:
                    entry["result"] = message.get("result")
                entry["event"].set()

        elif msg_type == "execute":
            # Go wants us to execute a script
            _execute_queue.put(message)


_execute_queue = __import__("queue").Queue()


def main():
    """
    Main entry point.
    
    Architecture (single process, pool model):
    - Router thread: reads ALL stdin, routes messages by type
    - Main thread: picks execute requests from queue, runs scripts
    - bridge_call(): blocks on threading.Event, unblocked by router thread
    
    Each Python process handles ONE execution at a time.
    Go's ProcessPool routes executions to available processes.
    
    Flow:
    1. Router thread reads stdin, routes "execute" to queue
    2. Main thread picks from queue, runs script
    3. Script calls bridge_call() → sends request via stdout
    4. Router thread receives bridge_response → unblocks bridge_call()
    5. Script completes → main thread sends result → picks next from queue
    """
    sys.stderr.write("[plugin:python] ready\n")
    sys.stderr.flush()

    # Start router thread (single reader for stdin)
    router = threading.Thread(target=message_router, daemon=True)
    router.start()

    # Main loop: process execute requests one at a time
    while True:
        message = _execute_queue.get()  # blocks until execute request arrives
        exec_id = message.get("id")
        params = message.get("params", {})

        try:
            result = execute_script(
                params.get("script", ""),
                params.get("params", {}),
                params.get("module", ""),
                params.get("session", {}),
                params.get("securityRules", {}),
            )
            send_to_go({"type": "execute_complete", "id": exec_id, "result": result})
        except BridgeError as e:
            send_to_go({
                "type": "execute_error", "id": exec_id,
                "error": {"code": e.code, "message": str(e), "details": e.details}
            })
        except Exception as e:
            send_to_go({
                "type": "execute_error", "id": exec_id,
                "error": {"code": "SCRIPT_ERROR", "message": str(e), "stack": traceback.format_exc()}
            })


if __name__ == "__main__":
    main()
```

---

## 6. Bridge Proxy — Sync vs Async

### 6.1 Default: Synchronous (Blocking)

Most Python scripts are synchronous. Bridge calls block the script thread:

```python
def execute(bitcode, params):
    # Each call blocks until Go responds — natural for Python
    leads = bitcode.model("lead").search({"domain": [["status", "=", "new"]]})
    for lead in leads:
        bitcode.model("lead").write(lead["id"], {"score": 85})
    return {"processed": len(leads)}
```

**How it works internally:**
1. Script thread calls `bridge_call("model.search", ...)`
2. `bridge_call` sends JSON-RPC to Go via stdout
3. `bridge_call` blocks on `threading.Event.wait()`
4. Reader thread (separate) receives response from Go via stdin
5. Reader thread sets the Event, unblocking script thread
6. Script thread gets result, continues

### 6.2 Optional: Async Support

For scripts that need concurrency (e.g., parallel HTTP calls):

```python
import asyncio

async def execute(bitcode, params):
    # Async version — parallel HTTP calls
    tasks = [
        asyncio.to_thread(bitcode.http.get, "https://api1.com"),
        asyncio.to_thread(bitcode.http.get, "https://api2.com"),
        asyncio.to_thread(bitcode.http.get, "https://api3.com"),
    ]
    results = await asyncio.gather(*tasks)
    return {"results": results}
```

**How it works**: `asyncio.to_thread()` runs each blocking `bridge_call` in a thread pool. The GIL is released during I/O wait (the `threading.Event.wait()`), so multiple bridge calls can be in-flight simultaneously.

**Detection in runtime**: `inspect.iscoroutinefunction(func)` → if True, run with `asyncio.run(func(bitcode, params))`.

```python
# In execute_script():
if inspect.iscoroutinefunction(func):
    return asyncio.run(func(bitcode, params))  # async script
else:
    return func(bitcode, params)  # sync script
```

---

## 7. Per-Module venv Isolation

### 7.1 Module Structure

```
modules/analytics/
├── module.json
├── requirements.txt       ← pip dependencies
├── .venv/                 ← virtual environment (created by bitcode CLI)
│   ├── bin/python3
│   ├── lib/python3.12/
│   │   └── site-packages/
│   │       ├── pandas/
│   │       ├── numpy/
│   │       └── scikit-learn/
│   └── pyvenv.cfg
└── scripts/
    └── analyze_data.py    ← can import pandas, numpy, etc.
```

### 7.2 How Import Resolution Works

```python
# In runtime.py execute_script():
# 1. Find module's venv site-packages
venv_site = os.path.join(module_dir, ".venv", "lib", f"python{major}.{minor}", "site-packages")

# 2. Prepend to sys.path (module-level first)
sys.path.insert(0, venv_site)
sys.path.insert(0, module_dir)

# Now when script does:
import pandas  # resolves from modules/analytics/.venv/lib/.../site-packages/pandas/
```

**Resolution order:**
1. Module's `.venv/lib/.../site-packages/` (first priority)
2. Module directory itself (for local imports)
3. System Python `site-packages` (fallback)
4. Python stdlib

### 7.3 Cross-Platform venv Paths

venv `site-packages` path differs per OS:

| OS | Path |
|----|------|
| Linux/macOS | `.venv/lib/python3.12/site-packages/` |
| Windows | `.venv/Lib/site-packages/` |

Runtime detects OS and constructs correct path:

```python
import platform
if platform.system() == "Windows":
    venv_site = os.path.join(module_dir, ".venv", "Lib", "site-packages")
else:
    venv_site = os.path.join(module_dir, ".venv", "lib",
                             f"python{sys.version_info.major}.{sys.version_info.minor}",
                             "site-packages")
```

---

## 8. pip Package Management (Step-by-Step)

### 8.1 How Developers Add pip Packages to a Module

**Step 1: Create `requirements.txt` in module directory**

```bash
# modules/analytics/requirements.txt
pandas>=2.0
numpy>=1.24
scikit-learn>=1.3
```

**Step 2: Create venv and install packages**

```bash
# Option A: bitcode CLI (recommended)
bitcode module install-deps analytics
# What it does:
# 1. cd modules/analytics
# 2. python3 -m venv .venv
# 3. .venv/bin/pip install -r requirements.txt
# 4. Done

# Option B: manual
cd modules/analytics
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

Result:
```
modules/analytics/
├── requirements.txt
├── .venv/
│   └── lib/python3.12/site-packages/
│       ├── pandas/
│       ├── numpy/
│       └── scikit-learn/
└── scripts/
    └── analyze_data.py
```

**Step 3: Use in script**

```python
# modules/analytics/scripts/analyze_data.py
import pandas as pd
from sklearn.cluster import KMeans

def execute(bitcode, params):
    leads = bitcode.model("lead").search({"domain": [["status", "=", "qualified"]]})
    df = pd.DataFrame(leads)
    kmeans = KMeans(n_clusters=3)
    df["segment"] = kmeans.fit_predict(df[["score", "revenue"]])
    
    for _, row in df.iterrows():
        bitcode.model("lead").write(row["id"], {"segment": int(row["segment"])})
    
    return {"segmented": len(leads)}
```

### 8.2 CLI Commands

```bash
# Install dependencies for one module (creates venv if needed)
bitcode module install-deps analytics
# → python3 -m venv modules/analytics/.venv (if not exists)
# → modules/analytics/.venv/bin/pip install -r modules/analytics/requirements.txt

# Install dependencies for ALL modules
bitcode module install-deps --all
# → finds all modules/*/requirements.txt, creates venv + pip install for each

# Add a package to a module
bitcode module add-package analytics pandas
# → modules/analytics/.venv/bin/pip install pandas
# → updates requirements.txt

# Remove a package
bitcode module remove-package analytics pandas
# → modules/analytics/.venv/bin/pip uninstall pandas
# → updates requirements.txt

# Freeze current packages (lock versions)
bitcode module freeze analytics
# → modules/analytics/.venv/bin/pip freeze > modules/analytics/requirements.txt

# Recreate venv from scratch
bitcode module recreate-venv analytics
# → rm -rf modules/analytics/.venv
# → python3 -m venv modules/analytics/.venv
# → pip install -r requirements.txt
```

### 8.3 What About .venv in .gitignore?

Developers should add to `.gitignore`:
```
modules/*/.venv/
```

And in CI/CD or deployment:
```bash
bitcode module install-deps --all
```

`requirements.txt` IS committed — it's the dependency declaration.

### 8.4 Edge Cases

| Edge Case | Behavior |
|-----------|----------|
| Module has no `requirements.txt` | No venv created. `import` only resolves stdlib + system packages. |
| `pip install` fails (network error) | CLI shows error. Module works without pip packages (bridge methods still available). |
| Two modules need different versions of `pandas` | Each has own `.venv/`. No conflict. |
| Script imports package not installed | Standard Python error: `ModuleNotFoundError: No module named 'pandas'` |
| Package has native extensions (e.g., `numpy`, `scipy`) | Works — pip compiles for current platform. Same as any Python project. |
| `.venv/` is very large (e.g., `torch` ~2GB) | Per-module isolation means only modules that need it pay the cost. |
| Python version mismatch (venv created with 3.11, runtime is 3.12) | May fail. `bitcode module recreate-venv` to fix. |
| Windows vs Linux venv structure | Runtime auto-detects OS for correct `site-packages` path. |

---

## 9. Script Signature Migration

### Old Style (current)

```python
def execute(params):
    leads = params.get("leads", [])
    return {"total": len(leads)}
```

### New Style

```python
def execute(bitcode, params):
    leads = bitcode.model("lead").search({"domain": [["status", "=", "new"]]})
    bitcode.log("info", f"Found {len(leads)} new leads")
    return {"total": len(leads)}
```

### Backward Compatibility

Runtime auto-detects signature:

```python
import inspect

sig = inspect.signature(func)
param_count = len(sig.parameters)

if param_count >= 2:
    result = func(bitcode, params)   # new style
else:
    result = func(params)            # legacy style
```

### Naming Convention: snake_case

Python bridge uses **snake_case** (Pythonic), not camelCase:

```python
# Python style (snake_case)
bitcode.model("lead").create_many(records)
bitcode.model("lead").write_many(ids, data)
bitcode.model("lead").delete_many(ids)
bitcode.model("lead").add_relation(id, field, ids)
bitcode.model("lead").remove_relation(id, field, ids)
bitcode.model("lead").load_relation(id, field)
bitcode.model("lead").sudo().hard_delete(id)
bitcode.model("lead").sudo().with_tenant("x")
bitcode.model("lead").sudo().skip_validation()
bitcode.security.has_group("crm.manager")

# But JSON-RPC method names stay camelCase (protocol level):
# "model.createMany", "model.writeMany", etc.
# Python proxy translates: create_many() → bridge_call("model.createMany", ...)
```

---

## 10. Critical Design Decisions

### 10.1 Threading Model

**Problem**: Python has GIL. Bridge calls need bidirectional communication. Script execution is synchronous. Both "execute" messages and "bridge_response" messages arrive on the same stdin — only one thread can read stdin.

**Solution**: Two threads per process with single stdin reader:
- **Router thread**: reads ALL stdin messages, routes by type (execute → queue, bridge_response → pending request)
- **Main thread**: picks execute requests from queue, runs scripts, blocks on bridge_call()

```
Router Thread (stdin reader)       Main Thread
    │                                  │
    │ for line in sys.stdin:           │ message = queue.get()  (blocks)
    │   if "execute":                  │
    │     queue.put(message) ─────────►│ execute_script()
    │                                  │   │
    │                                  │   │ bridge_call("model.search")
    │                                  │   │   send_to_go(request)  → stdout
    │                                  │   │   event.wait()  (blocks, releases GIL)
    │   if "bridge_response":          │   │
    │     entry["result"] = result     │   │
    │     entry["event"].set() ───────►│   │ (unblocked, has result)
    │                                  │   │ continue script...
    │                                  │   │
    │                                  │ send_to_go(execute_complete)
    │                                  │ message = queue.get()  (blocks, waits for next)
```

**Why single stdin reader**: Two threads reading from the same stdin is a race condition — one thread might read half a line. Router thread is the ONLY reader. Main thread never reads stdin directly.

**Why this works despite GIL**: `threading.Event.wait()` and `queue.Queue.get()` both release the GIL. So while main thread is blocked waiting for bridge response, router thread can run and process the incoming message from stdin.

### 10.2 Module Import Caching

**Problem**: Python caches imported modules in `sys.modules`. If developer edits a script, the old version stays cached.

**Solution**: Clear module from cache before each execution:

```python
if "bitcode_script" in sys.modules:
    del sys.modules["bitcode_script"]
```

**But**: Third-party packages (pandas, numpy) should stay cached — reimporting them is expensive. Only the script module itself is cleared.

### 10.3 Dual-Pool Architecture (Same as Node.js Phase 2)

Python's GIL makes process pool even more critical than Node.js. Same dual-pool pattern from Phase 2 §10.1.

Pool config is **unified** — same `bitcode.yaml` config applies to ALL runtimes (Node.js, Python, goja, qjs, yaegi). Developer thinks "is this fast or slow work?" — not per-language tuning.

Pool, timeout, and memory are all overridable via resolution order: **step → process → module → project → hardcoded**. See master doc §3.1d for full resolution table.

```json
// Module level: all scripts in analytics default to background pool with 4GB
{
  "name": "analytics",
  "default_pool": "background",
  "max_memory_mb": 4096,
  "default_step_timeout": "10m"
}

// Process level: this training gets 6GB and 2h timeout
{
  "name": "train_model",
  "pool": "background",
  "timeout": "2h",
  "max_memory_mb": 6144,
  "steps": [
    { "type": "script", "runtime": "python", "script": "train.py", "timeout": "90m" }
  ]
}
```

### 10.4 Timeout + Memory — Python-Specific Behavior

Resolution order is the same as Node.js (see Phase 2 §10.2). **Key difference**: Python has no `vm.timeout` equivalent. Timeout is enforced entirely from Go side by **killing the process**:

```go
select {
case res := <-resultCh:
    return res
case <-stepCtx.Done():
    // Timeout — must KILL Python process (no graceful interrupt)
    proc.kill()
    pool.replaceProcess(proc)
    return nil, &bridge.BridgeError{Code: "STEP_TIMEOUT", ...}
}
```

**Why kill is necessary**: Python has no mechanism to interrupt a running script from outside. Unlike Node.js where `vm.timeout` gracefully stops execution, Python requires process termination. This is why process pool is essential — killing one process doesn't affect others.

### 10.5 sys.path Pollution

**Problem**: Each script execution prepends module paths to `sys.path`. After 100 executions from different modules, `sys.path` has 200+ entries.

**Solution**: Save and restore `sys.path` per execution:

```python
def execute_script(script_path, params, module_name, ...):
    original_path = sys.path.copy()
    try:
        sys.path.insert(0, venv_site)
        sys.path.insert(0, module_dir)
        # ... execute script ...
    finally:
        sys.path = original_path
```

### 10.6 Binary Data

Same as Node.js (Phase 2 §9.5): base64 encoding for binary fields in JSON-RPC.

```python
import base64

def serialize_params(params):
    """Convert bytes to base64 for JSON-RPC."""
    if isinstance(params, bytes):
        return {"_type": "binary", "encoding": "base64", "data": base64.b64encode(params).decode()}
    if isinstance(params, dict):
        return {k: serialize_params(v) for k, v in params.items()}
    if isinstance(params, list):
        return [serialize_params(v) for v in params]
    return params
```

### 10.7 venv Python Version Mismatch

**Problem**: venv is tied to the Python version that created it. If server upgrades Python 3.11 → 3.12, all venvs break.

**Solution**: Detect at startup and warn.

```go
func (a *App) checkPythonVenvs() {
    for _, mod := range a.modules {
        venvPython := filepath.Join(mod.Path, ".venv", "bin", "python3")
        if !fileExists(venvPython) { continue }
        
        // Check venv Python version vs system Python version
        venvVersion := getVersion(venvPython)
        systemVersion := getVersion("python3")
        
        if venvVersion != systemVersion {
            log.Printf("[WARN] Module '%s': venv Python %s != system Python %s. Run: bitcode module recreate-venv %s",
                mod.Name, venvVersion, systemVersion, mod.Name)
        }
    }
}
```

### 10.8 Python Not Installed

Same pattern as Node.js:

```
Engine startup:
  → Check: python3 --version
  → Found Python 3.12.1 → start pool, log "[INFO] Python 3.12.1 runtime ready (pool: 4)"
  → Not found → skip, log "[INFO] Python runtime not available"
  → Found Python 3.8 → skip, log "[WARN] Python 3.8 found but minimum 3.10 required"

Script with runtime: "python" triggered:
  → BridgeError { code: "RUNTIME_NOT_AVAILABLE", message: "Python runtime not available. Install Python 3.10+" }
```

---

## 11. Edge Cases

### 11.1 Communication

| Edge Case | Behavior |
|-----------|----------|
| Python process crashes (segfault, OOM) | Go detects broken pipe, returns error, auto-restart with backoff |
| Bridge call timeout (60s) | `threading.Event.wait(timeout=60)` expires, raises `BridgeError("BRIDGE_TIMEOUT")` |
| Multiple concurrent executions | Process pool — each execution in separate process |
| Python sends malformed JSON | Go logs warning, ignores |

### 11.2 Script Execution

| Edge Case | Behavior |
|-----------|----------|
| Script has syntax error | `spec.loader.exec_module()` raises `SyntaxError`. Caught, returned as `execute_error`. |
| Script raises unhandled exception | Caught by try/except. Full traceback in error response. |
| Script runs forever (infinite loop) | No built-in timeout in Python (unlike Node.js `vm.timeout`). Rely on process-level timeout from Go side. |
| Script calls `sys.exit()` | Kills the process. Pool manager detects and restarts. |
| Script calls `os.fork()` | Works but dangerous. Not blocked (unlike yaegi). Document: don't do this. |
| Script modifies `sys.path` globally | Restored after execution (§10.4). |

### 11.3 Dependencies

| Edge Case | Behavior |
|-----------|----------|
| `import pandas` but not in venv | `ModuleNotFoundError: No module named 'pandas'` |
| Two modules need different pandas versions | Each has own `.venv/`. No conflict. |
| venv created with Python 3.11, runtime is 3.12 | May fail with ABI mismatch. `bitcode module recreate-venv` to fix. |
| Native extension fails to compile | pip shows error. Module works without that package. |

### 11.4 Performance

| Operation | Expected Latency |
|-----------|-----------------|
| Bridge RPC round-trip (Go ↔ Python via pipe) | ~0.1-0.5ms |
| `bitcode.model("lead").search({})` (10 records) | ~2-10ms (RPC + DB query) |
| Script import (first time, with pandas) | ~500-2000ms (pandas is heavy) |
| Script import (cached) | ~1-5ms |
| Process startup | ~100-200ms |

---

## 12. Implementation Tasks

### Files to Create

```
engine/plugins/python/
├── runtime.py              # Complete rewrite — bidirectional JSON-RPC + bitcode proxy

engine/internal/runtime/plugin/
├── pool.go                 # Shared with Phase 2 — ProcessPool (if not already created)
```

### Files to Modify

```
engine/internal/runtime/plugin/manager.go
  → Add StartPythonPool() (same pattern as StartNodePool)
  → Reuse bidirectional Execute() from Phase 2
  → Reuse bridge_handler.go from Phase 2

engine/internal/config.go
  → Add python runtime config (pool_size, max_memory, etc.)

engine/internal/app.go
  → Wire Python pool startup
  → Version check at startup

engine/cmd/bitcode/main.go
  → Add venv management to `bitcode module install-deps`
  → Add `bitcode module recreate-venv`
  → Add `bitcode module freeze`
```

### Files to Delete/Move

```
engine/plugins/python/runtime.py → rewrite in place
```

### What's Reused from Phase 2 (No New Code)

```
engine/internal/runtime/plugin/message.go        ← same message types
engine/internal/runtime/plugin/bridge_handler.go  ← same bridge routing
engine/internal/runtime/plugin/pool.go            ← same pool pattern (parameterized)
engine/internal/runtime/bridge/*                  ← same bridge.Context
```

**This is a key advantage of the shared protocol design** — Phase 3 is significantly less work than Phase 2 because the Go side is already done.

### Task Breakdown

| # | Task | Effort | Priority |
|---|------|--------|----------|
| **Core** | | | |
| 1 | Rewrite `runtime.py` — bidirectional JSON-RPC + bitcode proxy classes | Large | Must |
| 2 | Implement threading model (main thread + reader thread) | Medium | Must |
| 3 | Implement synchronous `bridge_call()` with `threading.Event` | Medium | Must |
| 4 | Add `StartPythonPool()` to manager.go (reuse pool pattern) | Small | Must |
| 5 | Add Python version check at startup | Small | Must |
| 6 | Add Python runtime config to config.go | Small | Must |
| 7 | Wire Python pool in app.go | Small | Must |
| **venv & Packages** | | | |
| 8 | Implement venv creation in `bitcode module install-deps` | Medium | Must |
| 9 | Implement `sys.path` manipulation for per-module venv | Small | Must |
| 10 | Implement `sys.path` save/restore per execution | Small | Must |
| 11 | Cross-platform venv path detection (Linux/macOS/Windows) | Small | Must |
| 12 | Add `bitcode module recreate-venv` CLI command | Small | Should |
| 13 | Add `bitcode module freeze` CLI command | Small | Should |
| **Compatibility** | | | |
| 14 | Auto-detect script signature (1 param vs 2 params) | Small | Must |
| 15 | Module cache clearing per execution (hot reload) | Small | Must |
| 16 | Async script detection (`inspect.iscoroutinefunction`) | Small | Should |
| 17 | Binary data base64 serialization | Small | Must |
| **Migration** | | | |
| 18 | Update 6 Python scripts in samples/erp to new style | Medium | Must |
| **Tests** | | | |
| 19 | Tests: bidirectional RPC (Python ↔ Go) | Large | Must |
| 20 | Tests: all 20 bridge namespaces via Python proxy | Large | Must |
| 21 | Tests: per-module venv isolation | Medium | Must |
| 22 | Tests: backward compatibility (old `execute(params)` signature) | Small | Must |
| 23 | Tests: snake_case → camelCase method mapping | Small | Must |
| 24 | Tests: threading model (concurrent bridge calls) | Medium | Must |
| 25 | Tests: error propagation (BridgeError through RPC) | Medium | Must |
| 26 | Tests: process pool + crash recovery | Medium | Must |
| 27 | Tests: sys.path save/restore | Small | Must |
| 28 | Tests: cross-platform venv paths | Small | Should |
| 29 | Tests: async script support | Small | Should |
| 30 | Tests: Python not installed (graceful degradation) | Small | Should |
| 31 | Tests: script timeout (process-level) | Small | Should |

### Effort Comparison: Phase 2 vs Phase 3

| Aspect | Phase 2 (Node.js) | Phase 3 (Python) |
|--------|-------------------|------------------|
| Go side changes | Large (new protocol) | Small (reuse Phase 2) |
| Runtime rewrite | Large (runtime.js) | Large (runtime.py) |
| Package management | Medium (npm) | Medium (pip + venv) |
| Total new Go code | ~1500 lines | ~200 lines |
| Total new JS/PY code | ~400 lines | ~500 lines |
| Tests | 35 | 31 |

**Phase 3 is ~40% less effort than Phase 2** because the Go side (protocol, bridge handler, pool) is already done.
