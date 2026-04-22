# Plugins

Plugins let you write custom logic in TypeScript or Python when JSON processes aren't enough.

## How It Works

```
Engine (Go) ◄──JSON-RPC over stdin/stdout──► Plugin Process (Node.js)
```

1. Engine spawns a plugin process
2. Communication via stdin/stdout using JSON-RPC
3. Engine sends execute request with script path and parameters
4. Plugin runs the script and returns result

## TypeScript Plugin

```typescript
import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    const lead = params.input;
    
    // Access database
    // const result = await ctx.db.query('SELECT * FROM users');
    
    // Call external API
    // await ctx.http.post('https://api.example.com/notify', { ... });
    
    console.log(`Deal won: ${lead.name}`);
    return { success: true };
  }
});
```

## Using in Process

```json
{
  "type": "script",
  "runtime": "typescript",
  "script": "scripts/on_deal_won.ts"
}
```

## Context Available to Scripts

| Property | Description |
|----------|-------------|
| `params.input` | Process input data |
| `params.variables` | Process variables |
| `params.result` | Previous step result |
| `params.user_id` | Current user ID |

## Plugin Manager

The plugin manager handles:
- Spawning plugin processes
- Health monitoring
- Restart on crash
- Request/response via JSON-RPC
- Timeout handling
