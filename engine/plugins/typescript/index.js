const readline = require('readline');
const path = require('path');
const vm = require('vm');
const fs = require('fs');

const rl = readline.createInterface({ input: process.stdin, output: process.stdout, terminal: false });

rl.on('line', async (line) => {
  let request;
  try {
    request = JSON.parse(line);
  } catch (e) {
    process.stdout.write(JSON.stringify({ error: 'invalid JSON', id: 0 }) + '\n');
    return;
  }

  const { method, params, id } = request;

  if (method === 'execute') {
    try {
      const scriptPath = params.script;
      const scriptParams = params.params || {};

      let result;
      if (fs.existsSync(scriptPath)) {
        const code = fs.readFileSync(scriptPath, 'utf-8');
        const ctx = {
          db: { query: async (sql) => ({ rows: [] }), create: async (table, data) => data },
          http: {
            get: async (url) => ({ status: 200, data: {} }),
            post: async (url, body) => ({ status: 200, data: {} }),
          },
          log: (...args) => console.error('[plugin]', ...args),
        };

        const sandbox = { module: { exports: {} }, exports: {}, require, console, ctx, params: scriptParams };
        vm.runInNewContext(code, sandbox, { filename: scriptPath });

        const plugin = sandbox.module.exports.default || sandbox.module.exports;
        if (typeof plugin === 'function') {
          result = await plugin(ctx, scriptParams);
        } else if (plugin && typeof plugin.execute === 'function') {
          result = await plugin.execute(ctx, scriptParams);
        } else {
          result = { executed: true, script: scriptPath };
        }
      } else {
        result = { warning: 'script not found', script: scriptPath };
      }

      process.stdout.write(JSON.stringify({ result, id }) + '\n');
    } catch (e) {
      process.stdout.write(JSON.stringify({ error: e.message, id }) + '\n');
    }
  } else {
    process.stdout.write(JSON.stringify({ error: `unknown method: ${method}`, id }) + '\n');
  }
});

process.stderr.write('[plugin:typescript] ready\n');
