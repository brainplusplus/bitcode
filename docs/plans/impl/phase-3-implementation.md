# Phase 3 Implementation Plan: Fix Python Child Process

**Estimated effort**: 6-8 days
**Prerequisites**: Phase 1 (bridge interfaces), Phase 1.5 (tenant behavior), Phase 2 (JSON-RPC protocol — reuse)
**Test command**: `go test ./internal/runtime/plugin/...` + manual test with sample scripts

---

## Implementation Order

```
Stream 1: Python Runtime Rewrite (Day 1-3)
  ↓
Stream 2: Threading Model (Router + Main) (Day 3-4)
  ↓
Stream 3: Bridge Proxy (Sync Calls) (Day 4-5)
  ↓
Stream 4: venv per-Module + pip (Day 5-6)
  ↓
Stream 5: Integration & Tests (Day 6-8)
```

---

## Stream 1: Python Runtime Rewrite

**File**: `plugins/python/` → rewrite entirely

- `runtime.py` — Main entry: starts JSON-RPC server (reuse protocol from Phase 2)
- `bridge.py` — Proxy: `bitcode.model("contact").get(id)` → JSON-RPC call to Go → wait for response
- `loader.py` — Script loader with importlib

**Key difference from Node.js**: Python bridge calls are **synchronous** (blocking). The JSON-RPC call blocks the Python thread until Go responds. This is natural for Python developers.

## Stream 2: Threading Model

```
Main thread: JSON-RPC router (receives execute requests from Go)
  ↓ spawns
Worker thread: executes script (bridge calls block this thread, not main)
```

This allows multiple scripts to execute concurrently in the same Python process.

## Stream 3: Bridge Proxy

```python
class BitcodeModel:
    def __init__(self, model_name, rpc_client):
        self._model = model_name
        self._rpc = rpc_client
    
    def get(self, id, **opts):
        return self._rpc.call("bitcode.model.get", {
            "model": self._model, "id": id, **opts
        })
    
    def search(self, **opts):
        return self._rpc.call("bitcode.model.search", {
            "model": self._model, **opts
        })
```

## Stream 4: venv per-Module

- Each module can have `requirements.txt`
- Engine creates venv per module: `python -m venv .venv`
- Engine runs `pip install -r requirements.txt` in venv
- Scripts execute within module's venv

## Definition of Done

- [ ] Bidirectional JSON-RPC works (Go ↔ Python)
- [ ] Reuses same pool infrastructure from Phase 2
- [ ] All 6 Python sample scripts execute with real bridge calls
- [ ] Sync bridge calls work (Python blocks until Go responds)
- [ ] Threading model: multiple scripts concurrent in one process
- [ ] venv per-module works
- [ ] pip install per-module works
- [ ] Crash recovery + timeout
