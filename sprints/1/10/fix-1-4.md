Berikut adalah daftar lengkap fitur yang perlu ada dalam platform Low-Code ERP seperti Frappe, Odoo, dan NocoBase, disajikan dalam format tabel yang terstruktur per kategori.

***

## 🗂️ Kategori Fitur Platform Low-Code ERP

### 1. Core Framework & Data Modeling

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 1 | **DocType / Schema Builder** | Visual builder untuk mendefinisikan model data (tabel, field, relasi) tanpa kode | Frappe  [github](https://github.com/frappe/frappe) |
| 2 | **Field Types** | Dukungan tipe field: Text, Number, Date, File, Link, Select, Table, Formula, dll | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |
| 3 | **Relasi Antar Model** | One-to-Many, Many-to-Many, Has One dengan foreign key otomatis | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |
| 4 | **Child Table / Inline Table** | Sub-tabel di dalam form (misal: item baris di invoice) | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 5 | **Virtual Fields / Formula Fields** | Field yang dihitung otomatis berdasarkan nilai field lain | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 6 | **Data Versioning** | Menyimpan riwayat versi perubahan setiap record | Odoo  [linktly](https://www.linktly.com/operations-software/odoo-erp-suite-review/) |
| 7 | **Multi-Source Data** | Koneksi ke database eksternal (PostgreSQL, MySQL, API, dll) | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/security/audit-logger/) |

***

### 2. Permission & Access Control

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 8 | **Role Management** | Buat dan kelola role (Admin, Manager, Staff, dll) | Frappe  [github](https://github.com/frappe/frappe) |
| 9 | **Permission Matrix** | Atur hak Create, Read, Update, Delete (CRUD) per role per DocType | Frappe  [frappeframework](https://frappeframework.com/homepage) |
| 10 | **Record Rules / Row-Level Security** | Batasi akses user hanya pada record tertentu (misal: hanya data cabangnya sendiri) | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 11 | **Field-Level Permission** | Sembunyikan atau readonly field tertentu berdasarkan role | Frappe  [frappeframework](https://frappeframework.com/homepage) |
| 12 | **Menu Access Control** | Atur menu mana yang bisa diakses role tertentu | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |
| 13 | **UI Visibility Rules** | Tampilkan/sembunyikan tombol, tab, section berdasarkan kondisi/role | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |
| 14 | **IP Whitelist / Session Policy** | Batasi akses dari IP tertentu atau atur durasi sesi | NocoBase  [docs.nocobase](https://docs.nocobase.com/v1/handbook/security/) |
| 15 | **Plugin Permission** | Atur akses ke fitur/plugin tertentu per role | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |

***

### 3. Audit Log & Monitoring

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 16 | **Audit Log** | Catat semua aktivitas user (create, update, delete, export, login) | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/security/audit-logger/) |
| 17 | **Record Activity Timeline** | Tampilkan riwayat perubahan di setiap record (siapa, kapan, apa yang diubah) | Frappe  [github](https://github.com/frappe/frappe) |
| 18 | **Login History** | Log waktu login/logout beserta IP dan User-Agent | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/security/audit-logger/) |
| 19 | **API Request Log** | Catat semua request API masuk/keluar | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/security/audit-logger/) |
| 20 | **Data Change Diff** | Tampilkan perbedaan nilai before/after saat record diubah | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |
| 21 | **Export/Import Log** | Catat siapa yang mengekspor/mengimpor data dan kapan | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/security/audit-logger/) |

***

### 4. Workflow & Automation

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 22 | **Workflow Builder (Visual)** | Drag-and-drop builder untuk membuat alur kerja (approval, state machine) | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 23 | **Approval Chain** | Multi-level approval dengan kondisi dinamis | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 24 | **Trigger & Action Rules** | Jalankan aksi otomatis berdasarkan event (on create, on update, on status change) | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 25 | **Scheduled Tasks / Cron** | Jadwalkan eksekusi otomatis (misal: kirim laporan tiap minggu) | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 26 | **Email / Notification Automation** | Kirim email/notifikasi otomatis berdasarkan workflow | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 27 | **Assignment Rules** | Assign record ke user/tim secara otomatis berdasarkan aturan | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 28 | **Webhook** | Kirim data ke sistem eksternal saat event terjadi | Frappe  [github](https://github.com/frappe/frappe) |
| 29 | **Server Script / Business Logic** | Tulis skrip Python/JS untuk logika bisnis kustom | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |

***

### 5. Form & UI Builder

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 30 | **Form Builder (Drag-and-Drop)** | Rancang layout form secara visual | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 31 | **Conditional Field Logic** | Tampilkan/sembunyikan field berdasarkan nilai field lain | Frappe  [github](https://github.com/frappe/frappe) |
| 32 | **Custom Validation Rules** | Validasi input dengan aturan kustom (regex, range, required-if) | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 33 | **Multi-Step Form / Wizard** | Form bertahap untuk proses panjang | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 34 | **Print Format / Template** | Buat template cetak PDF kustom (invoice, PO, dll) | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 35 | **Web Form (Public Form)** | Form publik yang bisa diakses tanpa login (untuk survey, lead, dll) | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 36 | **Kanban / List / Calendar View** | Berbagai mode tampilan data selain tabel | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |
| 37 | **Dashboard Builder** | Buat dashboard dengan widget KPI, chart, table | Frappe  [github](https://github.com/frappe/frappe) NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |

***

### 6. Reporting & Analytics

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 38 | **Report Builder** | Buat laporan tabular tanpa kode dengan filter dan group-by | Frappe  [github](https://github.com/frappe/frappe) |
| 39 | **Query Report (SQL/Script)** | Laporan berbasis SQL atau skrip untuk kasus kompleks | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |
| 40 | **Chart Builder** | Buat visualisasi data (bar, line, pie) dari query/model | Frappe  [frappe](https://frappe.io) |
| 41 | **Export Data (CSV/Excel/PDF)** | Ekspor data dari list atau report ke format file | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/security/audit-logger/) |
| 42 | **Pivot Table** | Analisis data multi-dimensi secara interaktif | Odoo  [linktly](https://www.linktly.com/operations-software/odoo-erp-suite-review/) |
| 43 | **Scheduled Report** | Kirim laporan otomatis via email secara terjadwal | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |

***

### 7. Integrasi & API

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 44 | **REST API Auto-Generated** | Setiap model otomatis memiliki endpoint REST API | Frappe  [github](https://github.com/frappe/frappe) |
| 45 | **OAuth2 / SSO** | Login via Google, Microsoft, LDAP, atau provider SSO lainnya | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |
| 46 | **API Key Management** | Buat dan kelola API key untuk integrasi eksternal | Frappe  [github](https://github.com/frappe/frappe) |
| 47 | **Data Import / Export** | Import massal via CSV/Excel dengan mapping field | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/security/audit-logger/) |
| 48 | **Third-Party Connector** | Konektor siap pakai ke sistem populer (Slack, WhatsApp, payment gateway) | Odoo  [linktly](https://www.linktly.com/operations-software/odoo-erp-suite-review/) |
| 49 | **GraphQL Support** | Endpoint GraphQL untuk query data yang fleksibel | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |

***

### 8. Konfigurasi & Customization

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 50 | **Custom App / Module** | Buat modul baru dari nol (HR, CRM, Inventory, dll) | Odoo  [sunarctechnologies](https://sunarctechnologies.com/blog/odoos-low-code-capabilities-accelerating-development-without-sacrificing-customization/) |
| 51 | **Workspace / Menu Builder** | Kustomisasi halaman beranda dan navigasi per role | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 52 | **Branding / White Label** | Ubah logo, warna, dan nama aplikasi | Odoo  [linktly](https://www.linktly.com/operations-software/odoo-erp-suite-review/) |
| 53 | **Multi-Language / i18n** | Dukungan banyak bahasa dengan terjemahan kustom | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |
| 54 | **Multi-Currency** | Transaksi dalam berbagai mata uang dengan konversi otomatis | Odoo  [linktly](https://www.linktly.com/operations-software/odoo-erp-suite-review/) |
| 55 | **Multi-Company / Multi-Branch** | Satu instansi untuk beberapa entitas perusahaan | Odoo  [linktly](https://www.linktly.com/operations-software/odoo-erp-suite-review/) |
| 56 | **Plugin / Extension System** | Arsitektur plugin untuk menambah fitur secara modular | NocoBase  [v2.docs.nocobase](https://v2.docs.nocobase.com/plugins/) |

***

### 9. Keamanan & Infrastruktur

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 57 | **Two-Factor Authentication (2FA)** | Lapisan keamanan tambahan saat login | NocoBase  [docs.nocobase](https://docs.nocobase.com/v1/handbook/security/) |
| 58 | **Data Encryption** | Enkripsi data sensitif di database dan saat transit (HTTPS) | NocoBase  [docs.nocobase](https://docs.nocobase.com/v1/handbook/security/) |
| 59 | **Backup & Restore** | Jadwalkan backup otomatis dan kemampuan restore | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |
| 60 | **Rate Limiting** | Batasi jumlah request API untuk mencegah abuse | NocoBase  [docs.nocobase](https://docs.nocobase.com/v1/handbook/security/) |
| 61 | **CSRF & XSS Protection** | Proteksi keamanan web standar | Frappe  [red-gate](https://www.red-gate.com/simple-talk/development/web/an-introduction-to-frappe-framework-features-and-benefits/) |
| 62 | **Soft Delete / Recycle Bin** | Record yang dihapus masuk ke tempat sampah sebelum permanent delete | Frappe  [github](https://github.com/frappe/frappe) |

***

### 10. Kolaborasi & Komunikasi

| # | Fitur | Deskripsi | Referensi |
|---|---|---|---|
| 63 | **Comment & Mention** | Komentar di record dengan mention user lain | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 64 | **Activity Feed** | Feed aktivitas real-time di level record maupun global | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 65 | **Email Inbox Integration** | Kirim dan terima email langsung dari dalam record | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 66 | **Task / To-Do per Record** | Tambahkan task terkait langsung pada setiap record | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |
| 67 | **In-App Notification** | Notifikasi real-time di dalam aplikasi | Frappe  [frappe](https://frappe.io/framework/low-code-no-code) |

***


dari fitur-fitur tersebut, mana saja yang sudah ada, dan yang belum ada?

buat laporan lengkap di fix-1-4-report.md