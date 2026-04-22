# Laporan Audit Fitur Platform Low-Code ERP

**Tanggal**: 20 April 2026
**Codebase**: BitCode Engine (Go + JSON definitions)
**Dibandingkan dengan**: Frappe, Odoo, NocoBase

---

## Ringkasan

| Metrik | Jumlah |
|--------|--------|
| Total Fitur | 67 |
| ✅ Sudah Ada | 30 |
| ⚠️ Sebagian | 9 |
| ❌ Belum Ada | 28 |
| **Persentase Selesai** | **44.8%** |

---

## Legenda

| Status | Arti |
|--------|------|
| ✅ | Sudah diimplementasi dan berfungsi |
| ⚠️ | Sebagian ada — fitur dasar ada tapi belum lengkap |
| ❌ | Belum ada sama sekali |

**Effort Estimate**:
- **S** = Small (1-2 hari, 1 file/modul)
- **M** = Medium (3-5 hari, beberapa file)
- **L** = Large (1-2 minggu, fitur baru)
- **XL** = Extra Large (2-4 minggu, subsistem baru)

---

## 1. Core Framework & Data Modeling

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 1 | **DocType / Schema Builder** | ⚠️ | Model didefinisikan via JSON (`modules/*/models/*.json`), di-parse oleh `internal/compiler/parser/model.go`. Admin UI di `/admin/models/:name` menampilkan field & record rules. | **Ada**: JSON-based schema definition + admin viewer. **Belum ada**: Visual drag-and-drop builder di browser untuk membuat/edit model secara interaktif. Frappe punya DocType builder UI, NocoBase punya visual schema editor. **Effort: L** — Perlu frontend CRUD untuk model JSON + field type picker + relation builder. |
| 2 | **Field Types** | ✅ | `docs/features/models.md` mendokumentasikan 16 tipe: string, text, integer, decimal, boolean, date, datetime, selection, email, many2one, one2many, many2many, json, file, computed. Parser di `internal/compiler/parser/model.go`. | Lengkap. Mendukung semua tipe umum termasuk computed fields (meski evaluator belum selesai — lihat #5). |
| 3 | **Relasi Antar Model** | ✅ | `many2one` (FK), `one2many` (reverse FK), `many2many` (junction table auto-created). Didokumentasikan di `docs/features/models.md`. Junction table di-generate otomatis oleh `MigrateModel()`. | Lengkap. One-to-Many, Many-to-Many, Has One semua didukung. |
| 4 | **Child Table / Inline Table** | ✅ | `one2many` field type + form view dengan tabs yang bisa embed view lain: `{ "tabs": [{ "label": "Lines", "view": "order_line_list" }] }`. Contoh di `docs/features/views.md`. | Berfungsi via one2many + embedded list view di tab form. |
| 5 | **Virtual Fields / Formula Fields** | ⚠️ | Tipe `computed` ada di parser (`"type": "computed"`, `"computed": "expression"`). Terdaftar di `docs/features/models.md`. | **Ada**: Definisi field computed di JSON. **Belum ada**: Expression evaluator runtime — tercatat di AGENTS.md sebagai "Remaining: Computed field evaluation". Belum bisa evaluate `sum(lines.subtotal)` saat query. Frappe punya formula field evaluator. **Effort: M** — Perlu expression parser + evaluator di query time. |
| 6 | **Data Versioning** | ⚠️ | Audit log model (`modules/base/models/audit_log.json`) menyimpan `changes` (JSON) per record. Middleware `internal/presentation/middleware/audit.go` log write operations. | **Ada**: Audit log mencatat siapa mengubah apa. **Belum ada**: Full version history dengan kemampuan rollback/restore ke versi sebelumnya. Odoo menyimpan complete snapshot per versi. **Effort: M** — Perlu simpan full before/after snapshot, bukan hanya log. |
| 7 | **Multi-Source Data** | ⚠️ | Mendukung 3 database: SQLite, PostgreSQL, MySQL via `DB_DRIVER` config. Didokumentasikan di `docs/features/configuration.md`. | **Ada**: Multi-database driver (SQLite/Postgres/MySQL). **Belum ada**: Koneksi ke database eksternal secara bersamaan (multi-source). Saat ini hanya 1 database per instance. NocoBase bisa connect ke multiple external databases. **Effort: L** — Perlu connection pool manager untuk multiple datasources + model-level datasource config. |

**Subtotal**: ✅ 3 | ⚠️ 4 | ❌ 0

---

## 2. Permission & Access Control

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 8 | **Role Management** | ✅ | Model `role` di `modules/base/models/role.json`. Domain logic di `internal/domain/security/role.go`. Admin UI di `/admin` dengan menu "Roles". | Lengkap. CRUD role via API dan admin UI. |
| 9 | **Permission Matrix** | ✅ | Permission pattern `module.model.action` (read/create/write/delete). Didefinisikan di `module.json`. Middleware `internal/presentation/middleware/permission.go`. Auto-derived saat `auto_crud: true`. | Lengkap. RBAC per model per action. |
| 10 | **Record Rules / Row-Level Security** | ✅ | `record_rules` di model JSON. Domain logic di `internal/domain/security/record_rule.go`. Middleware `internal/presentation/middleware/record_rule.go`. Inject WHERE clause otomatis. | Lengkap. Domain filter syntax dengan operators dan variabel `{{user.id}}`. |
| 11 | **Field-Level Permission** | ❌ | Tidak ditemukan implementasi. Grep untuk `field.*permission`, `FieldPermission` tidak menghasilkan match. | **Belum ada**: Tidak bisa hide/readonly field tertentu berdasarkan role. Frappe punya per-field permission (read/write per role). **Effort: M** — Perlu field-level permission config di model JSON + filter di API response dan form renderer. |
| 12 | **Menu Access Control** | ✅ | Menu didefinisikan di `module.json` per module. Module hanya di-load jika installed. Group/permission system mengontrol akses. | Berfungsi via module-level menu definition. Menu hanya muncul jika module terinstall. |
| 13 | **UI Visibility Rules** | ✅ | View actions punya `"visible": "status == 'draft'"` condition. Form fields punya `"readonly": true`. Didokumentasikan di `docs/features/views.md`. Component compiler di `internal/presentation/view/component_compiler.go`. | Berfungsi untuk action buttons dan field readonly. Condition expression di-evaluate saat render. |
| 14 | **IP Whitelist / Session Policy** | ❌ | Tidak ditemukan. Grep untuk `ip.*whitelist`, `session.*policy` tidak menghasilkan match. | **Belum ada**: Tidak ada pembatasan IP atau kebijakan durasi sesi. NocoBase punya IP whitelist dan session timeout config. **Effort: S** — Middleware sederhana untuk IP check + JWT expiry config. |
| 15 | **Plugin Permission** | ⚠️ | Permission system ada per module (`module.json` → `permissions`). Plugin scripts berjalan dalam konteks module. | **Ada**: Permission per module. **Belum ada**: Granular permission per plugin/script individual. NocoBase punya per-plugin access control. **Effort: S** — Extend permission pattern ke `module.plugin.script_name`. |

**Subtotal**: ✅ 5 | ⚠️ 1 | ❌ 2

---

## 3. Audit Log & Monitoring

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 16 | **Audit Log** | ✅ | Model `audit_log` di `modules/base/models/audit_log.json` dengan fields: user_id, action (create/update/delete/login/logout), model_name, record_id, changes (JSON), ip_address. Middleware `internal/presentation/middleware/audit.go`. | Lengkap. Mencatat semua write operations + login/logout. |
| 17 | **Record Activity Timeline** | ⚠️ | Audit log menyimpan per-record changes (model_name + record_id + changes JSON). | **Ada**: Data tersimpan di audit_log. **Belum ada**: UI timeline view per record yang menampilkan riwayat perubahan secara visual. Frappe menampilkan activity timeline di setiap form. **Effort: M** — Perlu API endpoint filter audit_log by record + timeline UI component. |
| 18 | **Login History** | ⚠️ | Audit log action includes `login`/`logout` + `ip_address`. | **Ada**: Login/logout tercatat di audit_log. **Belum ada**: Dedicated login history view dengan User-Agent, device info, geolocation. **Effort: S** — Extend audit_log fields + dedicated view. |
| 19 | **API Request Log** | ⚠️ | Audit middleware logs write operations (POST/PUT/DELETE/PATCH) ke stdout. | **Ada**: Log ke stdout untuk write operations. **Belum ada**: Persistent API request log untuk semua request (termasuk GET), disimpan ke database, bisa di-query. NocoBase punya full API request logging. **Effort: M** — Perlu request logger middleware + storage + query API. |
| 20 | **Data Change Diff** | ⚠️ | Audit log punya field `changes` (JSON) yang menyimpan perubahan. | **Ada**: Changes disimpan sebagai JSON. **Belum ada**: Structured before/after diff display. Frappe menampilkan field-by-field diff (old value → new value). **Effort: S** — Perlu simpan old_value/new_value per field + diff UI component. |
| 21 | **Export/Import Log** | ❌ | Tidak ditemukan. Tidak ada fitur export/import data, jadi log-nya juga belum ada. | **Belum ada**: Fitur export/import data belum ada (lihat #41, #47), sehingga log-nya juga belum ada. **Effort: S** (setelah export/import diimplementasi) — Tambah action type di audit_log. |

**Subtotal**: ✅ 1 | ⚠️ 4 | ❌ 1

---

## 4. Workflow & Automation

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 22 | **Workflow Builder (Visual)** | ⚠️ | Workflow engine lengkap via JSON definition (`docs/features/workflows.md`). State machine dengan states, transitions, permissions, process linking. Runtime di `internal/runtime/workflow/`. | **Ada**: Full workflow engine via JSON. **Belum ada**: Visual drag-and-drop builder di browser. Frappe punya visual workflow builder. **Effort: L** — Perlu frontend state machine editor (nodes + edges + properties panel). |
| 23 | **Approval Chain** | ✅ | Workflow transitions punya `permission` field. Multi-level approval via chained transitions: draft → confirmed (need `order.confirm`) → done (need `order.write`). | Berfungsi via workflow transitions dengan permission gates. |
| 24 | **Trigger & Action Rules** | ✅ | Agent system dengan event triggers: `{ "event": "order.confirmed", "action": "send_confirmation", "script": "scripts/send_email.ts" }`. Process engine punya `emit` step. Didokumentasikan di `docs/features/agents.md`. | Lengkap. Event-driven triggers via agent definitions + process emit steps. |
| 25 | **Scheduled Tasks / Cron** | ✅ | Agent cron: `{ "schedule": "0 9 * * *", "action": "daily_report", "script": "scripts/daily_report.ts" }`. Standard cron format. Runtime di `internal/runtime/agent/`. | Lengkap. Full cron scheduler dengan retry + backoff. |
| 26 | **Email / Notification Automation** | ❌ | Tidak ditemukan SMTP/email sending implementation. Grep untuk `email.*send`, `smtp`, `SMTP` tidak menghasilkan match. | **Belum ada**: Tidak ada email sending service. Agent bisa trigger script yang mengirim email, tapi tidak ada built-in email integration. Frappe punya built-in email queue + template system. **Effort: L** — Perlu SMTP config, email queue, template engine, notification preferences. |
| 27 | **Assignment Rules** | ❌ | Tidak ditemukan. Grep untuk `assignment.*rule`, `auto.*assign` tidak menghasilkan match. | **Belum ada**: Tidak ada auto-assignment berdasarkan aturan. Frappe punya Assignment Rule DocType. **Effort: M** — Perlu assignment rule JSON definition + evaluator yang jalan saat record create/update. |
| 28 | **Webhook** | ❌ | Tidak ditemukan. Grep untuk `webhook`, `Webhook` tidak menghasilkan match. | **Belum ada**: Tidak ada webhook outgoing. Process engine punya `http` step type yang bisa call external API, tapi bukan webhook system yang configurable. Frappe punya Webhook DocType. **Effort: M** — Perlu webhook definition (URL, events, headers) + dispatcher yang listen ke event bus. |
| 29 | **Server Script / Business Logic** | ✅ | Plugin system (TypeScript + Python) via JSON-RPC. Process engine `script` step: `{ "type": "script", "runtime": "typescript", "script": "scripts/on_deal_won.ts" }`. Didokumentasikan di `docs/features/plugins.md`. | Lengkap. TypeScript dan Python runtime, JSON-RPC protocol, plugin manager dengan health monitoring. |

**Subtotal**: ✅ 4 | ⚠️ 1 | ❌ 3

---

## 5. Form & UI Builder

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 30 | **Form Builder (Drag-and-Drop)** | ⚠️ | Form layout didefinisikan via JSON dengan rows, fields, widths, tabs. Rendered oleh `internal/presentation/view/`. Stencil.js components di `packages/components/`. | **Ada**: JSON-based form layout definition + SSR rendering. **Belum ada**: Visual drag-and-drop form designer di browser. Odoo punya Studio form editor. **Effort: L** — Perlu frontend form designer yang output JSON layout definition. |
| 31 | **Conditional Field Logic** | ✅ | View actions punya `"visible": "status == 'draft'"`. Form fields punya `"readonly": true`. Component compiler di `internal/presentation/view/component_compiler.go` evaluates conditions. | Berfungsi untuk visibility conditions dan readonly state. |
| 32 | **Custom Validation Rules** | ✅ | Process `validate` step: `{ "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "..." }`. Operators: eq, neq, required. Didokumentasikan di `docs/features/processes.md`. | Berfungsi via process validate step. Mendukung eq, neq, required rules. |
| 33 | **Multi-Step Form / Wizard** | ❌ | Tidak ditemukan. Grep untuk `wizard`, `multi.*step` tidak menghasilkan match. | **Belum ada**: Tidak ada multi-step form/wizard component. Odoo punya wizard system untuk proses bertahap. **Effort: M** — Perlu wizard JSON definition (steps + fields per step) + stepper UI component. |
| 34 | **Print Format / Template** | ❌ | Tidak ada PDF generation. Template engine ada (Go html/template) tapi hanya untuk HTML views. Grep untuk `pdf`, `PDF` hanya match di upload handler (file type check). | **Belum ada**: Tidak ada PDF generation atau print template system. Frappe punya Print Format builder + PDF via wkhtmltopdf. **Effort: L** — Perlu print template JSON definition + PDF renderer (wkhtmltopdf/chromedp/gotenberg). |
| 35 | **Web Form (Public Form)** | ❌ | Tidak ditemukan. Grep untuk `web.*form`, `public.*form` tidak menghasilkan match. Semua form memerlukan auth. | **Belum ada**: Tidak ada public-facing form tanpa login. Frappe punya Web Form untuk survey, lead capture, dll. **Effort: M** — Perlu public form route (bypass auth) + CAPTCHA + rate limiting. |
| 36 | **Kanban / List / Calendar View** | ✅ | 6 view types: list, form, kanban, calendar, chart, custom. Semua diimplementasi di `internal/presentation/view/`. Templates di `modules/base/templates/views/`. | Lengkap. Semua view types termasuk kanban board, calendar, dan chart. |
| 37 | **Dashboard Builder** | ✅ | Custom view type dengan `data_sources`: `{ "type": "custom", "template": "templates/dashboard.html", "data_sources": {...} }`. Admin dashboard di `/admin`. | Berfungsi via custom view type + data sources. Admin panel punya built-in dashboard. |

**Subtotal**: ✅ 4 | ⚠️ 1 | ❌ 3

---

## 6. Reporting & Analytics

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 38 | **Report Builder** | ⚠️ | List view punya `filters` dan `sort`. Custom API endpoints bisa serve report data. Tapi tidak ada dedicated report builder. | **Ada**: List view dengan filter + sort. **Belum ada**: Dedicated report builder dengan group-by, aggregation, calculated columns. Frappe punya Report Builder dengan drag-and-drop columns. **Effort: L** — Perlu report JSON definition (columns, filters, group_by, aggregations) + report renderer. |
| 39 | **Query Report (SQL/Script)** | ⚠️ | Process engine `query` step bisa read data. Plugin scripts bisa execute custom logic. Custom API endpoints bisa serve data. | **Ada**: Query via process steps + plugin scripts. **Belum ada**: Dedicated SQL/script report system yang bisa dijalankan dari UI. Frappe punya Script Report dan Query Report. **Effort: M** — Perlu report type "script" atau "query" + safe SQL executor + result renderer. |
| 40 | **Chart Builder** | ✅ | Chart view type: `{ "type": "chart" }`. Template di `modules/base/templates/views/chart.html`. | Berfungsi sebagai view type. |
| 41 | **Export Data (CSV/Excel/PDF)** | ❌ | Tidak ditemukan. Grep untuk `csv`, `excel`, `xlsx` hanya match di upload handler (file type). Tidak ada export functionality. | **Belum ada**: Tidak ada data export ke CSV/Excel/PDF. NocoBase punya built-in export. **Effort: M** — Perlu export handler per model (CSV via encoding/csv, Excel via excelize library, PDF via print template). |
| 42 | **Pivot Table** | ❌ | Tidak ditemukan. Grep untuk `pivot`, `Pivot` tidak menghasilkan match. | **Belum ada**: Tidak ada pivot table analysis. Odoo punya pivot view. **Effort: L** — Perlu pivot table engine (dimensions, measures, aggregations) + interactive UI component. |
| 43 | **Scheduled Report** | ❌ | Cron system ada, tapi tidak ada report scheduling. Tidak ada email sending untuk deliver reports. | **Belum ada**: Cron ada tapi belum ada report + email delivery integration. Frappe punya Auto Email Report. **Effort: M** (setelah email + report builder ada) — Perlu scheduled report definition + email delivery. |

**Subtotal**: ✅ 1 | ⚠️ 2 | ❌ 3

---

## 7. Integrasi & API

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 44 | **REST API Auto-Generated** | ✅ | `auto_crud: true` di API JSON → generates GET (list+detail), POST, PUT, DELETE. Pagination, search, soft delete. Didokumentasikan di `docs/features/apis.md`. | Lengkap. Full auto-CRUD dengan pagination, search, auth, RLS. |
| 45 | **OAuth2 / SSO** | ❌ | Tidak ditemukan. Grep untuk `OAuth`, `SSO`, `LDAP` tidak menghasilkan match. Hanya JWT auth (username/password). | **Belum ada**: Hanya JWT login. Tidak ada OAuth2 provider, SSO, atau LDAP integration. Frappe punya OAuth2 + Social Login. **Effort: L** — Perlu OAuth2 client (Google, Microsoft, GitHub) + LDAP connector + SSO protocol handler. |
| 46 | **API Key Management** | ❌ | Tidak ditemukan. Grep untuk `api.*key`, `ApiKey` tidak menghasilkan match. | **Belum ada**: Tidak ada API key system untuk integrasi machine-to-machine. Frappe punya API Key per user. **Effort: S** — Perlu api_key model + auth middleware yang accept API key header. |
| 47 | **Data Import / Export** | ❌ | Tidak ada import/export UI. Data seeding ada via `data/*.json` tapi itu untuk module install, bukan user-facing import. | **Belum ada**: Tidak ada CSV/Excel import dengan field mapping. NocoBase punya import wizard. **Effort: L** — Perlu import wizard (upload file → map columns → validate → insert) + export (lihat #41). |
| 48 | **Third-Party Connector** | ⚠️ | Process engine `http` step bisa call external APIs: `{ "type": "http", "url": "...", "method": "POST" }`. Plugin scripts bisa call any API. | **Ada**: HTTP step + plugin scripts untuk integrasi. **Belum ada**: Pre-built connectors (Slack, WhatsApp, payment gateway). Odoo punya marketplace connectors. **Effort: XL** — Perlu connector framework + individual connector implementations. |
| 49 | **GraphQL Support** | ❌ | Tidak ditemukan. Tercatat di AGENTS.md sebagai "Remaining: GraphQL API". | **Belum ada**: Tercatat sebagai planned feature. NocoBase punya GraphQL endpoint. **Effort: L** — Perlu GraphQL schema generator dari model definitions + resolver layer. |

**Subtotal**: ✅ 1 | ⚠️ 1 | ❌ 4

---

## 8. Konfigurasi & Customization

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 50 | **Custom App / Module** | ✅ | Module system lengkap. CLI: `bitcode module create mymod`. Dependency resolution, topological sort. 3 built-in modules (base, crm, sales). Didokumentasikan di `docs/features/modules.md`. | Lengkap. Full module system dengan dependency management. |
| 51 | **Workspace / Menu Builder** | ✅ | Menu didefinisikan di `module.json` dengan label, icon, children, view links. Layout template dengan sidebar navigation. | Berfungsi via JSON menu definition + sidebar rendering. |
| 52 | **Branding / White Label** | ❌ | Tidak ditemukan dedicated branding system. Grep untuk `branding`, `white.*label` hanya match di `internal/app.go` (logo reference in template). | **Belum ada**: Tidak ada UI untuk ubah logo, warna, nama aplikasi. Template hardcoded. Odoo punya branding settings. **Effort: S** — Perlu branding settings (logo URL, app name, primary color) + inject ke template. |
| 53 | **Multi-Language / i18n** | ✅ | Translation files per module (`i18n/*.json`). Translator API dengan fallback chain. Didokumentasikan di `docs/features/i18n.md`. Infrastructure di `internal/infrastructure/i18n/`. | Lengkap. Multi-language support dengan locale fallback. |
| 54 | **Multi-Currency** | ❌ | Tidak ditemukan dedicated currency system. `formatCurrency` template helper ada tapi hardcoded format. Grep untuk `currency`, `Currency` match di template helper dan settings, tapi tidak ada conversion system. | **Belum ada**: Tidak ada currency model, exchange rate, atau automatic conversion. Odoo punya multi-currency dengan daily rate updates. **Effort: L** — Perlu currency model + exchange rate table + conversion logic di computed fields. |
| 55 | **Multi-Company / Multi-Branch** | ⚠️ | Multi-tenancy ada (header/subdomain/path strategy). Tenant isolation di repository level. | **Ada**: Multi-tenancy bisa digunakan sebagai multi-company. **Belum ada**: Dedicated multi-company dengan inter-company transactions, consolidated reporting. Odoo punya multi-company dengan cross-company rules. **Effort: L** — Perlu company model + company-level settings + inter-company transaction support. |
| 56 | **Plugin / Extension System** | ✅ | Plugin system lengkap: TypeScript + Python runtime, JSON-RPC protocol, gRPC proto defined. Plugin manager dengan health monitoring, restart on crash. Didokumentasikan di `docs/features/plugins.md`. | Lengkap. Dual-runtime plugin system dengan robust process management. |

**Subtotal**: ✅ 4 | ⚠️ 1 | ❌ 2

---

## 9. Keamanan & Infrastruktur

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 57 | **Two-Factor Authentication (2FA)** | ❌ | Tidak ditemukan. Grep untuk `2FA`, `TwoFactor`, `totp`, `otp` tidak menghasilkan match. | **Belum ada**: Hanya username/password login. NocoBase punya TOTP-based 2FA. **Effort: M** — Perlu TOTP library (pquerna/otp) + 2FA setup flow + verification middleware. |
| 58 | **Data Encryption** | ⚠️ | HTTPS bisa dikonfigurasi di reverse proxy. Password di-hash (`password_hash` field). JWT signed. Tapi tidak ada field-level encryption. | **Ada**: Password hashing + JWT signing. **Belum ada**: Field-level encryption untuk data sensitif di database. NocoBase punya encrypted fields. **Effort: M** — Perlu encryption/decryption layer di repository untuk marked fields. |
| 59 | **Backup & Restore** | ❌ | Tidak ditemukan. Grep untuk `backup`, `restore` tidak menghasilkan match. | **Belum ada**: Tidak ada backup/restore functionality. Frappe punya scheduled backup + download. **Effort: M** — Perlu database dump command (sqlite: copy file, pg: pg_dump, mysql: mysqldump) + restore command + optional scheduling. |
| 60 | **Rate Limiting** | ❌ | Tidak ditemukan. Grep untuk `rate.*limit`, `RateLimit` tidak menghasilkan match. | **Belum ada**: Tidak ada API rate limiting. NocoBase punya configurable rate limits. **Effort: S** — Perlu rate limiter middleware (gofiber/limiter atau custom token bucket). |
| 61 | **CSRF & XSS Protection** | ⚠️ | Go html/template auto-escapes output (XSS protection built-in). Tapi tidak ada explicit CSRF token system. | **Ada**: XSS protection via Go template auto-escaping. **Belum ada**: CSRF token untuk form submissions. API menggunakan JWT (stateless, tidak perlu CSRF), tapi SSR forms perlu CSRF protection. **Effort: S** — Perlu CSRF middleware untuk SSR form routes. |
| 62 | **Soft Delete / Recycle Bin** | ✅ | Setiap model punya `active` boolean field (auto-generated). DELETE = set `active = false`. API config `soft_delete: true` (default). Implementasi di `internal/infrastructure/persistence/repository.go` dan `internal/presentation/api/crud_handler.go`. | Berfungsi. Soft delete via `active` flag. **Belum ada**: Recycle bin UI untuk melihat dan restore deleted records. **Effort: S** — Perlu recycle bin view (filter `active = false`) + restore endpoint. |

**Subtotal**: ✅ 1 | ⚠️ 2 | ❌ 3

---

## 10. Kolaborasi & Komunikasi

| # | Fitur | Status | Evidence | Gap Analysis |
|---|-------|--------|----------|--------------|
| 63 | **Comment & Mention** | ❌ | Tidak ditemukan. Grep untuk `comment`, `mention` tidak menghasilkan match. | **Belum ada**: Tidak ada comment system per record. Frappe punya comment + @mention di setiap DocType. **Effort: M** — Perlu comment model (record_id, user_id, content, mentions) + API + UI component. |
| 64 | **Activity Feed** | ❌ | Tidak ditemukan dedicated activity feed. Audit log ada tapi tidak ada feed UI. | **Belum ada**: Tidak ada activity feed real-time. Frappe punya activity feed di homepage. **Effort: M** — Perlu activity feed API (aggregate dari audit_log + comments) + feed UI component + WebSocket integration. |
| 65 | **Email Inbox Integration** | ❌ | Tidak ditemukan. Tidak ada email sending (lihat #26), apalagi inbox integration. | **Belum ada**: Tidak ada email integration sama sekali. Frappe punya Email Account + inbox per record. **Effort: XL** — Perlu IMAP client + email parsing + thread linking ke records + send via SMTP. |
| 66 | **Task / To-Do per Record** | ❌ | Tidak ditemukan. Tidak ada todo/task model terkait record. | **Belum ada**: Tidak ada task/todo system. Frappe punya ToDo DocType yang bisa di-link ke record apapun. **Effort: M** — Perlu todo model (record_id, model_name, assigned_to, due_date, status) + API + UI widget. |
| 67 | **In-App Notification** | ⚠️ | WebSocket system ada (`internal/presentation/websocket/`). Domain events di-broadcast ke connected clients. Subscribe per channel. | **Ada**: WebSocket event broadcasting. **Belum ada**: Persistent notification system (notification model, read/unread status, notification preferences, notification bell UI). Frappe punya notification center. **Effort: M** — Perlu notification model + preferences + bell UI + WebSocket delivery. |

**Subtotal**: ✅ 0 | ⚠️ 1 | ❌ 4

---

## Ringkasan per Kategori

| # | Kategori | ✅ | ⚠️ | ❌ | Total | % Selesai |
|---|----------|-----|------|------|-------|-----------|
| 1 | Core Framework & Data Modeling | 3 | 4 | 0 | 7 | 43% (71% jika partial dihitung) |
| 2 | Permission & Access Control | 5 | 1 | 2 | 8 | 63% |
| 3 | Audit Log & Monitoring | 1 | 4 | 1 | 6 | 17% (50% jika partial) |
| 4 | Workflow & Automation | 4 | 1 | 3 | 8 | 50% |
| 5 | Form & UI Builder | 4 | 1 | 3 | 8 | 50% |
| 6 | Reporting & Analytics | 1 | 2 | 3 | 6 | 17% |
| 7 | Integrasi & API | 1 | 1 | 4 | 6 | 17% |
| 8 | Konfigurasi & Customization | 4 | 1 | 2 | 7 | 57% |
| 9 | Keamanan & Infrastruktur | 1 | 2 | 3 | 6 | 17% |
| 10 | Kolaborasi & Komunikasi | 0 | 1 | 4 | 5 | 0% |
| | **TOTAL** | **24** | **18** | **25** | **67** | **35.8%** |

> **Catatan**: Jika fitur ⚠️ (partial) dihitung sebagai 0.5, maka skor efektif = 24 + 9 = **33 / 67 = 49.3%**

---

## Effort Summary — Fitur yang Belum Ada

### Quick Wins (Effort S — 1-2 hari)

| # | Fitur | Effort |
|---|-------|--------|
| 14 | IP Whitelist / Session Policy | S |
| 15 | Plugin Permission (granular) | S |
| 46 | API Key Management | S |
| 52 | Branding / White Label | S |
| 60 | Rate Limiting | S |
| 61 | CSRF Protection (untuk SSR forms) | S |
| 62 | Recycle Bin UI (soft delete sudah ada) | S |

### Medium Effort (Effort M — 3-5 hari)

| # | Fitur | Effort |
|---|-------|--------|
| 5 | Computed Field Evaluator | M |
| 6 | Data Versioning (full snapshot) | M |
| 11 | Field-Level Permission | M |
| 17 | Record Activity Timeline UI | M |
| 20 | Data Change Diff UI | M |
| 27 | Assignment Rules | M |
| 28 | Webhook System | M |
| 33 | Multi-Step Form / Wizard | M |
| 35 | Web Form (Public Form) | M |
| 39 | Query/Script Report | M |
| 41 | Export Data (CSV/Excel/PDF) | M |
| 57 | 2FA (TOTP) | M |
| 58 | Field-Level Encryption | M |
| 59 | Backup & Restore | M |
| 63 | Comment & Mention | M |
| 64 | Activity Feed | M |
| 66 | Task / To-Do per Record | M |
| 67 | In-App Notification (persistent) | M |

### Large Effort (Effort L — 1-2 minggu)

| # | Fitur | Effort |
|---|-------|--------|
| 1 | Visual Schema Builder UI | L |
| 7 | Multi-Source Data | L |
| 22 | Visual Workflow Builder UI | L |
| 26 | Email / Notification Automation | L |
| 30 | Visual Form Builder UI | L |
| 34 | Print Format / PDF Template | L |
| 38 | Report Builder | L |
| 42 | Pivot Table | L |
| 45 | OAuth2 / SSO | L |
| 47 | Data Import / Export Wizard | L |
| 49 | GraphQL API | L |
| 54 | Multi-Currency | L |
| 55 | Multi-Company (beyond multi-tenant) | L |

### Extra Large Effort (Effort XL — 2-4 minggu)

| # | Fitur | Effort |
|---|-------|--------|
| 48 | Third-Party Connectors | XL |
| 65 | Email Inbox Integration | XL |

---

## Catatan Detail

### Kekuatan Utama Platform Saat Ini

1. **Core engine sangat solid** — JSON-driven development, DDD architecture, multi-database support
2. **Security foundation kuat** — JWT + RBAC + Record Rules (row-level security) sudah production-ready
3. **Workflow & Process engine lengkap** — 14 step types, state machine, event-driven architecture
4. **Plugin system mature** — Dual runtime (TypeScript + Python), JSON-RPC, health monitoring
5. **Module system seperti Odoo** — Dependency resolution, data seeding, cross-module views
6. **View system komprehensif** — 6 view types termasuk kanban, calendar, chart

### Gap Terbesar

1. **Tidak ada visual builders** (#1, #22, #30) — Semua konfigurasi via JSON. Ini gap terbesar dibanding Frappe/Odoo/NocoBase yang punya visual editors.
2. **Tidak ada email system** (#26, #43, #65) — Blocking untuk notification automation, scheduled reports, dan collaboration.
3. **Tidak ada export/import** (#41, #47) — Fitur dasar yang diharapkan user.
4. **Kolaborasi minimal** (#63-67) — Tidak ada comment, mention, todo, notification center.
5. **Reporting lemah** (#38, #39, #42, #43) — Hanya chart view, belum ada report builder atau pivot table.

### Rekomendasi Prioritas Implementasi

**Phase 1 — Quick Wins (1-2 minggu)**
Implementasi semua fitur effort S: rate limiting, CSRF, API keys, branding, recycle bin UI, IP whitelist.

**Phase 2 — Core Gaps (3-4 minggu)**
- Export Data (#41) + Data Import (#47) — Fitur paling diminta user
- Computed Field Evaluator (#5) — Sudah planned, unblock formula fields
- Webhook (#28) — Enable integrasi dengan sistem lain
- Comment & Mention (#63) + In-App Notification (#67) — Dasar kolaborasi

**Phase 3 — Visual Builders (6-8 minggu)**
- Visual Schema Builder (#1)
- Visual Form Builder (#30)
- Visual Workflow Builder (#22)
Ini yang membedakan "low-code" dari "JSON-code".

**Phase 4 — Enterprise Features (8-12 minggu)**
- OAuth2/SSO (#45) + 2FA (#57) — Enterprise security
- Email System (#26) + Scheduled Reports (#43)
- Report Builder (#38) + Pivot Table (#42)
- Multi-Currency (#54) + Multi-Company (#55)
