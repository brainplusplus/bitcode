# Agents & Cron

Agents handle background work — reacting to events and running scheduled jobs.

## Definition

```json
{
  "name": "order_agent",
  "triggers": [
    {
      "event": "order.confirmed",
      "action": "send_confirmation",
      "script": "scripts/send_email.ts"
    }
  ],
  "cron": [
    {
      "schedule": "0 9 * * *",
      "action": "daily_report",
      "script": "scripts/daily_report.ts"
    }
  ],
  "retry": { "max": 3, "backoff": "exponential" }
}
```

## Event Triggers

When a process emits an event via `{ "type": "emit", "event": "order.confirmed" }`, all agents subscribed to that event are notified.

The agent runs the specified script with the event data as parameters.

## Cron Jobs

Cron expressions follow standard format:

```
┌───────── minute (0-59)
│ ┌─────── hour (0-23)
│ │ ┌───── day of month (1-31)
│ │ │ ┌─── month (1-12)
│ │ │ │ ┌─ day of week (0-6, Sun=0)
│ │ │ │ │
* * * * *
```

Examples:
- `0 9 * * *` — Every day at 9:00 AM
- `0 9 * * 1` — Every Monday at 9:00 AM
- `*/5 * * * *` — Every 5 minutes
- `0 0 1 * *` — First day of every month

## Retry

```json
"retry": { "max": 3, "backoff": "exponential" }
```

Failed scripts are retried up to `max` times with exponential backoff.

## Event Flow

```
Process step: { "type": "emit", "event": "order.confirmed" }
  → Event Bus publishes "order.confirmed"
  → Agent worker receives event
  → Runs script "scripts/send_email.ts"
  → On failure: retry with backoff
```
