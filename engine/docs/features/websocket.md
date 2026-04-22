# WebSocket

Real-time updates via WebSocket. Domain events are automatically broadcast to connected clients.

## Connect

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('Connected');
  ws.send(JSON.stringify({ type: 'subscribe', channel: 'order.confirmed' }));
  ws.send(JSON.stringify({ type: 'subscribe', channel: '*' }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log(msg.type, msg.channel, msg.data);
};
```

## Query Parameters

| Param | Description |
|-------|-------------|
| `user_id` | Associate connection with a user |
| `tenant_id` | Associate connection with a tenant (for multi-tenancy) |

Example: `ws://localhost:8080/ws?user_id=abc&tenant_id=company-a`

## Client Messages

| Type | Description |
|------|-------------|
| `subscribe` | Subscribe to a channel: `{ "type": "subscribe", "channel": "order.confirmed" }` |
| `unsubscribe` | Unsubscribe: `{ "type": "unsubscribe", "channel": "order.confirmed" }` |
| `ping` | Keep-alive: `{ "type": "ping" }` → responds with `{ "type": "pong" }` |

## Server Messages

| Type | Description |
|------|-------------|
| `event` | Domain event: `{ "type": "event", "channel": "order.confirmed", "data": {...} }` |
| `subscribed` | Confirmation: `{ "type": "subscribed", "channel": "order.confirmed" }` |
| `pong` | Keep-alive response |

## How Events Flow

```
Process step: { "type": "emit", "event": "order.confirmed" }
  → Event Bus publishes "order.confirmed"
  → WebSocket Hub receives event (subscribed to all events)
  → Broadcasts to all clients subscribed to "order.confirmed" or "*"
```

## Channel Patterns

- `order.confirmed` — specific event
- `lead.*` — not yet supported (exact match only)
- `*` — all events

## Multi-tenancy

When `tenant_id` is set on the WebSocket connection, events can be scoped to that tenant using `BroadcastToTenant()`.
