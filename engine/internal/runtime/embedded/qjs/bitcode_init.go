package qjs_runtime

const bitcodeInitJS = `
const bitcode = {
  model: (name) => ({
    search: (opts) => __bc_model_search(name, opts || {}),
    get: (id) => __bc_model_get(name, id),
    create: (data) => __bc_model_create(name, data),
    write: (id, data) => __bc_model_write(name, id, data),
    delete: (id) => __bc_model_delete(name, id),
    count: (opts) => __bc_model_count(name, opts || {}),
    sudo: () => ({
      search: (opts) => __bc_model_search(name, opts || {}),
      get: (id) => __bc_model_get(name, id),
      create: (data) => __bc_model_create(name, data),
      write: (id, data) => __bc_model_write(name, id, data),
      delete: (id) => __bc_model_delete(name, id),
      count: (opts) => __bc_model_count(name, opts || {}),
    }),
  }),
  db: {
    query: (sql, ...args) => __bc_db_query(sql),
    execute: (sql, ...args) => __bc_db_execute(sql),
  },
  http: {
    get: (url, opts) => __bc_http_request('GET', url, opts),
    post: (url, opts) => __bc_http_request('POST', url, opts),
    put: (url, opts) => __bc_http_request('PUT', url, opts),
    patch: (url, opts) => __bc_http_request('PATCH', url, opts),
    delete: (url, opts) => __bc_http_request('DELETE', url, opts),
  },
  cache: {
    get: (key) => __bc_cache_get(key),
    set: (key, val, opts) => __bc_cache_set(key, val, opts),
    del: (key) => __bc_cache_del(key),
  },
  fs: {
    read: (p) => __bc_fs_read(p),
    write: (p, content) => __bc_fs_write(p, content),
    exists: (p) => __bc_fs_exists(p),
    list: (p) => __bc_fs_list(p),
    mkdir: (p) => __bc_fs_mkdir(p),
    remove: (p) => __bc_fs_remove(p),
  },
  env: (key) => __bc_env(key),
  config: (key) => __bc_config(key),
  log: (level, msg, data) => __bc_log(level, msg),
  emit: (event, data) => __bc_emit(event, data),
  call: (process, input) => __bc_call(process, input),
  t: (key) => __bc_t(key),
  exec: (cmd, args, opts) => __bc_exec(cmd, args, opts),
  email: {
    send: (opts) => __bc_email_send(opts),
  },
  notify: {
    send: (opts) => __bc_notify_send(opts),
    broadcast: (channel, data) => __bc_notify_broadcast(channel, data),
  },
  crypto: {
    hash: (val) => __bc_crypto_hash(val),
    verify: (val, hash) => __bc_crypto_verify(val, hash),
  },
  audit: {
    log: (opts) => __bc_audit_log(opts),
  },
  session: typeof __bc_session !== 'undefined' ? __bc_session : {},
};

const console = {
  log: (...args) => __bc_log('info', args.join(' ')),
  warn: (...args) => __bc_log('warn', args.join(' ')),
  error: (...args) => __bc_log('error', args.join(' ')),
  debug: (...args) => __bc_log('debug', args.join(' ')),
};
`
