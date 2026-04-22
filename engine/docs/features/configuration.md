# Configuration

## Priority Order

```
Defaults → bitcode.toml → bitcode.yaml → .env file → OS environment variables
```

Each layer overrides the previous. OS environment variables always win.

| Priority | Source | Description |
|----------|--------|-------------|
| 1 (lowest) | Built-in defaults | SQLite, memory cache, port 8080 |
| 2 | `bitcode.toml` | Preferred config file format |
| 3 | `bitcode.yaml` | Alternative config file (if no .toml found) |
| 4 | `.env` file | Key=value pairs, good for secrets |
| 5 (highest) | OS env vars | `PORT=9090 ./engine` — always wins |

## Config Options

| Key | TOML | YAML | .env / OS env | Default | Description |
|-----|------|------|---------------|---------|-------------|
| Server port | `port` | `port` | `PORT` | `8080` | HTTP server port |
| Module directory | `module_dir` | `module_dir` | `MODULE_DIR` | `modules` | Path to modules |
| JWT secret | `jwt_secret` | `jwt_secret` | `JWT_SECRET` | `change-me...` | Token signing key |
| DB driver | `[database] driver` | `database.driver` | `DB_DRIVER` | `sqlite` | `sqlite`, `postgres`, `mysql` |
| DB host | `[database] host` | `database.host` | `DB_HOST` | `localhost` | Database host |
| DB port | `[database] port` | `database.port` | `DB_PORT` | `5432` | Database port |
| DB user | `[database] user` | `database.user` | `DB_USER` | `bitcode` | Database user |
| DB password | `[database] password` | `database.password` | `DB_PASSWORD` | `bitcode` | Database password |
| DB name | `[database] name` | `database.name` | `DB_NAME` | `bitcode` | Database name |
| DB SSL mode | `[database] sslmode` | `database.sslmode` | `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| SQLite path | `[database] sqlite_path` | `database.sqlite_path` | `DB_SQLITE_PATH` | `bitcode.db` | SQLite file path |
| Cache driver | `[cache] driver` | `cache.driver` | `CACHE_DRIVER` | `memory` | `memory`, `redis` |
| Redis URL | `[cache] redis_url` | `cache.redis_url` | `REDIS_URL` | - | Redis connection URL |
| Tenant enabled | `[tenant] enabled` | `tenant.enabled` | `TENANT_ENABLED` | `false` | Enable multi-tenancy |
| Tenant strategy | `[tenant] strategy` | `tenant.strategy` | `TENANT_STRATEGY` | `header` | `header`, `subdomain`, `path` |
| Tenant header | `[tenant] header` | `tenant.header` | `TENANT_HEADER` | `X-Tenant-ID` | Header name for tenant |

---

## Format: TOML (Recommended)

File: `bitcode.toml`

```toml
name = "my-erp"
version = "1.0.0"
port = 8080
module_dir = "modules"
jwt_secret = "your-secret-key-at-least-32-chars!"

[database]
driver = "sqlite"
sqlite_path = "data.db"

# PostgreSQL example:
# driver = "postgres"
# host = "localhost"
# port = 5432
# user = "bitcode"
# password = "secret"
# name = "myapp"
# sslmode = "disable"

# MySQL example:
# driver = "mysql"
# host = "localhost"
# port = 3306
# user = "root"
# password = "secret"
# name = "myapp"

[cache]
driver = "memory"

# Redis example:
# driver = "redis"
# redis_url = "redis://localhost:6379"

[tenant]
enabled = false
strategy = "header"
header = "X-Tenant-ID"
```

---

## Format: YAML

File: `bitcode.yaml`

```yaml
name: my-erp
version: 1.0.0
port: 8080
module_dir: modules
jwt_secret: your-secret-key-at-least-32-chars!

database:
  driver: sqlite
  sqlite_path: data.db
  # driver: postgres
  # host: localhost
  # port: 5432
  # user: bitcode
  # password: secret
  # name: myapp

cache:
  driver: memory
  # driver: redis
  # redis_url: redis://localhost:6379

tenant:
  enabled: false
  strategy: header
  header: X-Tenant-ID
```

---

## Format: .env File

File: `.env`

```env
# Server
PORT=8080
MODULE_DIR=modules
JWT_SECRET=your-secret-key-at-least-32-chars!

# Database
DB_DRIVER=sqlite
DB_SQLITE_PATH=data.db
# DB_DRIVER=postgres
# DB_HOST=localhost
# DB_PORT=5432
# DB_USER=bitcode
# DB_PASSWORD=secret
# DB_NAME=myapp
# DB_SSLMODE=disable

# Cache
CACHE_DRIVER=memory
# CACHE_DRIVER=redis
# REDIS_URL=redis://localhost:6379

# Multi-tenancy
TENANT_ENABLED=false
TENANT_STRATEGY=header
TENANT_HEADER=X-Tenant-ID
```

The `.env` file supports:
- Comments with `#`
- Blank lines
- Quoted values: `JWT_SECRET="my secret"` or `JWT_SECRET='my secret'`
- No export prefix needed

Also checks `.env.local` as fallback.

---

## Format: OS Environment Variables

```bash
# Linux / macOS
PORT=9090 DB_DRIVER=postgres DB_HOST=db.example.com ./engine

# Windows (cmd)
set PORT=9090
set DB_DRIVER=postgres
engine.exe

# Windows (PowerShell)
$env:PORT = "9090"
$env:DB_DRIVER = "postgres"
.\engine.exe

# Docker
docker run -e PORT=9090 -e DB_DRIVER=postgres bitcode/engine
```

---

## Examples

### Development (SQLite, defaults)

Just run — no config needed:
```bash
./engine
```
Uses SQLite at `bitcode.db`, port 8080, memory cache.

### Development with TOML

```toml
# bitcode.toml
port = 3000
jwt_secret = "dev-secret-not-for-production!!"

[database]
driver = "sqlite"
sqlite_path = "dev.db"
```

### Production with PostgreSQL + Redis

```toml
# bitcode.toml
port = 8080
jwt_secret = "super-secret-production-key-32ch!"

[database]
driver = "postgres"
host = "db.internal"
port = 5432
user = "app"
password = "from-vault"
name = "erp_prod"
sslmode = "require"

[cache]
driver = "redis"
redis_url = "redis://redis.internal:6379"

[tenant]
enabled = true
strategy = "subdomain"
```

### Docker Compose (env vars)

```yaml
services:
  engine:
    image: bitcode/engine
    environment:
      - PORT=8080
      - DB_DRIVER=postgres
      - DB_HOST=postgres
      - DB_USER=bitcode
      - DB_PASSWORD=bitcode
      - DB_NAME=bitcode
      - CACHE_DRIVER=redis
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
```

### Mixed: TOML + .env for secrets

```toml
# bitcode.toml — committed to git
port = 8080
module_dir = "modules"

[database]
driver = "postgres"
host = "localhost"
name = "myapp"
```

```env
# .env — NOT committed to git (in .gitignore)
DB_USER=admin
DB_PASSWORD=super-secret
JWT_SECRET=production-jwt-key-32-characters!!
```

This way, non-secret config is in TOML (version controlled), and secrets are in `.env` (gitignored).

---

## File Detection

The engine auto-detects config files in the current working directory:

1. `bitcode.toml` — checked first (preferred)
2. `bitcode.yaml` or `bitcode.yml` — checked if no TOML found
3. `.env` or `.env.local` — always checked (supplements config file)

You can also specify a config file explicitly:

```bash
./engine --config /path/to/custom.toml
# or
CONFIG_FILE=/path/to/custom.yaml ./engine
```
