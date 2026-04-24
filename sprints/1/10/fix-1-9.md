

## 9. Security & Infrastructure

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 57 | Two-Factor Auth (2FA) | ❌ | M | — | Only username/password. Need TOTP library + 2FA setup flow + verification middleware. |
| 58 | Data Encryption | ⚠️ | M | Password hashing (bcrypt) + JWT signing. HTTPS via reverse proxy. | No field-level encryption for sensitive data in database. |
| 59 | Backup & Restore | ❌ | M | — | No backup/restore. Need db dump command (sqlite: copy, pg: pg_dump, mysql: mysqldump) + restore + scheduling. |
| 60 | Rate Limiting | ❌ | S | — | No API rate limiting. Need rate limiter middleware (gofiber/limiter or token bucket). |
| 61 | CSRF & XSS Protection | ⚠️ | S | XSS: Go html/template auto-escapes. API uses JWT (stateless, no CSRF needed). | CSRF token needed for SSR form submissions. |

oia sama tambahkan fitur impersonate user lain buat admin (ini juga masuk audit log saat impersonate). dan di audit log ada informasi impersonated_by siapa kalau ada aksi tertentu yang dilakukan karena impersonate

selesaikan itu ya, dan kalau fase implement, jalankan sendiri tanpa sub agent

jika ada yang butuh di improve agar sempurna, kasih tau aja ya
atau ada yang ingin di konfirmasi juga berkabar ya

dan juga jangan lupa setelah selesai semua update semua docs-docs terkait, lalu commit dan push, file ini juga termasuk di commit
