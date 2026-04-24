
## 1. Core Framework & Data Modeling

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 1 | Schema Builder | ⚠️ | L | JSON-based model definitions (`models/*.json`), parsed by `compiler/parser/model.go`. Admin UI at `/admin/models/:name` shows fields & rules. | Visual drag-and-drop builder in browser. Frappe has DocType builder UI, NocoBase has visual schema editor. |
| 5 | Computed / Formula Fields | ⚠️ | M | `computed` type defined in parser. JSON definition supported. | Runtime expression evaluator missing — can't evaluate `sum(lines.subtotal)` at query time. Listed in AGENTS.md as "Remaining". Dan juga ada computed/format/formula yg bukan dari query, nah itu ada pengaturan di jsonnya |
| 6 | Data Versioning | ⚠️ | M | Audit log model stores `changes` (JSON) per record. Middleware logs write operations. | Full before/after snapshot for rollback/restore. Currently only logs, not snapshots. Nah ini saya bingung, baiknya utk rollback atau restore menggunakan table terpisah semisal data_histories atau audit_log ? |

jika ada yang butuh di improve agar sempurna, kasih tau aja ya

selesaikan itu ya, dan kalau fase implement, jalankan sendiri tanpa sub agent

dan juga jangan lupa setelah selesai semua update semua docs-docs terkait, lalu commit dan push, sprints juga termasuk di commit
