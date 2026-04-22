# Workflows

Workflows define state machines for models. They control which transitions are allowed and who can perform them.

## Definition

```json
{
  "name": "order_workflow",
  "model": "order",
  "field": "status",
  "states": {
    "draft":     { "label": "Draft" },
    "confirmed": { "label": "Confirmed" },
    "done":      { "label": "Done" },
    "cancelled": { "label": "Cancelled" }
  },
  "transitions": [
    { "from": "draft",     "to": "confirmed", "action": "confirm", "permission": "order.confirm", "process": "confirm_order" },
    { "from": "confirmed", "to": "done",      "action": "complete", "permission": "order.write" },
    { "from": ["draft", "confirmed"], "to": "cancelled", "action": "cancel", "permission": "order.write" }
  ]
}
```

## How It Works

1. **States** — possible values for the status field
2. **Transitions** — allowed state changes with:
   - `from` — current state (string or array for multiple)
   - `to` — target state
   - `action` — action name (becomes API endpoint)
   - `permission` — required permission
   - `process` — optional process to run during transition

## API Integration

Link a workflow to an API:

```json
{
  "name": "order_api",
  "model": "order",
  "auto_crud": true,
  "auth": true,
  "workflow": "order_workflow",
  "actions": {
    "confirm":  { "transition": "confirm",  "permission": "order.confirm" },
    "complete": { "transition": "complete", "permission": "order.complete" }
  }
}
```

This generates `POST /api/orders/:id/confirm` and `POST /api/orders/:id/complete`.

## Transition Validation

```
POST /api/orders/:id/confirm
  → Check: current status == "draft"? (from the workflow)
  → Check: user has "order.confirm" permission?
  → Run "confirm_order" process (if defined)
  → Update status to "confirmed"
  → Emit "order.confirmed" event
```

Invalid transitions return an error:
```json
{ "error": "transition \"confirm\" not allowed from state \"done\"" }
```

## Multi-From Transitions

A transition can have multiple source states:

```json
{ "from": ["draft", "confirmed"], "to": "cancelled", "action": "cancel" }
```

This allows cancelling from either draft or confirmed state.
