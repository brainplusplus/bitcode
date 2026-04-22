seperti halnya low code untuk erp, butuh component2 siap pake

Berikut adalah daftar lengkap dan detail komponen yang dibutuhkan untuk membangun low-code ERP seperti ERPNext dan Odoo.

***

# Komponen Low-Code ERP: ERPNext & Odoo

Membangun platform low-code ERP membutuhkan setidaknya tiga lapisan komponen utama: **Form Fields**, **Layout/Structure**, dan **View Types** (termasuk chart dan tabel). Berikut rinciannya.

***

## Form Field Types

Ini adalah komponen inti untuk input data. Berikut semua tipe field berdasarkan kategori:

### Input Teks & Angka
| Field Type | Deskripsi | ERP |
|---|---|---|
| **Data / Char** | Input teks singkat (maks ~140 karakter), bisa divalidasi sebagai Email, URL, Phone, Name | ERPNext & Odoo |
| **Small Text** | Teks pendek, multi-baris tanpa formatting | ERPNext |
| **Text / Long Text** | Teks panjang tanpa batas karakter | ERPNext & Odoo |
| **Text Editor** | WYSIWYG rich-text editor (bold, italic, list, dll) | ERPNext & Odoo |
| **Markdown Editor** | Input Markdown dengan preview HTML | ERPNext |
| **HTML** | Render HTML statis dari options | ERPNext |
| **Code** | Code editor dengan syntax highlighting (Python, JS, dll) | ERPNext |
| **Password** | Input terenkripsi untuk menyimpan password/secret | ERPNext & Odoo |
| **Integer / Int** | Angka bulat tanpa desimal | ERPNext & Odoo |
| **Float** | Angka desimal hingga 9 digit | ERPNext & Odoo |
| **Currency** | Angka desimal dengan simbol mata uang, hingga 6 desimal | ERPNext & Odoo |
| **Percent** | Input persentase (0–100) | ERPNext & Odoo |

### Pilihan & Relasi
| Field Type | Deskripsi | ERP |
|---|---|---|
| **Select / Dropdown** | Pilihan tunggal dari daftar opsi statis | ERPNext & Odoo |
| **Check / Checkbox** | Input boolean (true/false) | ERPNext & Odoo |
| **Toggle** | Switch on/off visual (tanpa masuk mode edit) | Odoo |
| **Radio Button** | Pilihan tunggal, tampil sebagai radio group | Odoo |
| **Link** | Relasi ke DocType/model lain (foreign key) | ERPNext & Odoo (`Many2one`) |
| **Dynamic Link** | Link ke DocType mana pun secara dinamis | ERPNext |
| **Table MultiSelect** | Kombinasi Link + pilihan multiple, tampil sebagai tag | ERPNext |
| **Many2many / Tags** | Pilihan banyak, tampil sebagai tag pills berwarna | Odoo |
| **Many2many Checkboxes** | Pilihan banyak via checkbox list | Odoo |

### Tanggal & Waktu
| Field Type | Deskripsi |
|---|---|
| **Date** | Date picker |
| **Time** | Time picker |
| **DateTime** | Gabungan date + time picker |
| **Duration** | Input rentang waktu (hari, jam, menit, detik) |

### File & Media
| Field Type | Deskripsi |
|---|---|
| **Attach / Binary** | Upload file umum (PDF, Excel, ZIP, dll) |
| **Attach Image / Image** | Upload khusus gambar (JPEG, PNG), ditampilkan langsung |
| **PDF Viewer** | Upload & tampilkan PDF inline di form |
| **Signature** | Canvas untuk tanda tangan digital |

### Special & Advanced
| Field Type | Deskripsi |
|---|---|
| **Barcode** | Input nomor barcode dan generate gambar barcode |
| **Color Picker** | Pilih warna via color picker atau input hex |
| **Geolocation / Map** | Peta interaktif (Leaflet), bisa gambar polygon, line, point |
| **Rating** | Input rating bintang (3–10 bintang, mendukung setengah bintang) |
| **JSON** | Input JSON mentah dengan syntax highlighting |
| **Read Only** | Field hanya-baca, nilai dari kalkulasi/fetch |

***

## Layout & Struktur Form

Komponen non-data yang mengatur tataletak form: [docs.frappe](https://docs.frappe.io/framework/user/en/basics/doctypes/fieldtypes)

- **Section Break** — Membagi form menjadi beberapa seksi horizontal (dengan/tanpa judul dan deskripsi)
- **Column Break** — Membagi seksi menjadi kolom (maks 2 kolom di ERPNext, fleksibel di Odoo)
- **Tab Break** — Membagi form menjadi beberapa tab navigasi
- **HTML Block** — Sisipkan HTML/konten statis di antara field
- **Button Field** — Tombol untuk trigger aksi spesifik (misalnya "Submit", "Recalculate")

***

## Child Table / Inline Table

Komponen paling khas di ERP — tabel baris-baris data di dalam satu form: [docs.frappe](https://docs.frappe.io/framework/user/en/basics/doctypes/fieldtypes)

- **Table (Child DocType)** — Tabel editable dengan banyak kolom, tombol "Add Row", dan "Delete Row". Contoh: Item Lines di Sales Order, Purchase Order
- **Table MultiSelect** — Versi ringkas, tidak ada tombol Add Row, tampil sebagai multi-tag

Fitur dalam child table:
- Kolom bisa berisi field type apa pun (Data, Currency, Link, Select, dll)
- Bisa set formula/kalkulasi otomatis per baris
- Support drag & drop reorder (via `handle` widget di Odoo)
- Subtotal dan summary row

***

## View Types (Bukan Form)

Selain form, ERP modern butuh berbagai jenis tampilan data:

| View | Fungsi |
|---|---|
| **List View** | Daftar record tabular dengan sorting, filtering, grouping |
| **Kanban View** | Board kolom berdasarkan status/stage (drag & drop) |
| **Calendar View** | Tampilkan record berdasarkan field tanggal |
| **Gantt Chart** | Timeline project management |
| **Map View** | Tampilkan record yang punya field geolocation di peta |
| **Report Builder** | Query report dengan filter, kolom kustom, totaling |
| **Dashboard** | Kumpulan chart + KPI card dalam satu halaman |
| **Tree View** | Hierarki parent-child (chart of accounts, BOM) |
| **Activity View** | Timeline log aktivitas dan komunikasi |

***

## Chart & Visualisasi (Dashboard Components)

Komponen chart yang digunakan di dashboard ERP: [docs.frappe](https://docs.frappe.io/framework/user/en/basics/doctypes/fieldtypes)

- **Number Card / KPI Card** — Angka besar dengan label, tren naik/turun
- **Bar Chart** — Batang vertikal/horizontal (perbandingan)
- **Line Chart** — Tren waktu (penjualan bulanan, cash flow)
- **Pie / Donut Chart** — Proporsi (distribusi produk, pembayaran)
- **Area Chart** — Line chart dengan area terisi
- **Heatmap** — Grid intensitas warna (aktivitas per hari/bulan)
- **Funnel Chart** — Pipeline sales/lead conversion
- **Gauge / Percentage Circle** — Persentase pencapaian target
- **Progress Bar** — Bar horizontal untuk persentase (Odoo widget `progressbar`)
- **Pivot Table** — Tabel silang agregasi data
- **Scorecard** — Komparasi metrik vs target

***

## Widget Tambahan (Odoo-Specific)

Odoo memiliki sistem widget berbasis `widget="..."` di XML view: [cybrosys](https://www.cybrosys.com/odoo/odoo-books/odoo-18-development/views/widgets/)

- `statusbar` — Visualisasi alur status sebagai panah/segmented bar
- `priority` / `stars` — Bintang prioritas (Low/Normal/High/Urgent)
- `handle` — Drag & drop icon untuk reorder di list view
- `phone` / `email` / `url` — Render sebagai link yang bisa diklik/dial
- `badge` — Tampilkan nilai sebagai pill read-only berwarna
- `copy_clipboard` — Tombol copy ke clipboard
- `emoji` — Emoji picker
- `ace` — Code editor ace (SQL, XML, Python)
- `domain` — Widget builder untuk domain filter (kondisi)
- `many2many_tags` — Many2many sebagai tag pills berwarna

***

## Fitur Form Behavior (Low-Code Logic)

Selain tipe komponen, Anda juga perlu mengimplementasikan logika form: [manualpt.angolaerp.co](https://manualpt.angolaerp.co.ao/docs/user/manual/en/customize-erpnext/custom-field)

- **Depends On** — Tampil/sembunyi field berdasarkan kondisi (expression)
- **Read Only If** — Field jadi read-only berdasarkan kondisi
- **Mandatory If** — Field jadi wajib berdasarkan kondisi
- **Fetch From** — Auto-populate nilai dari linked record
- **Default Value** — Nilai awal saat form baru dibuat
- **Formula/Computed Field** — Hitung otomatis dari field lain (misalnya `qty * price`)
- **Naming Series** — Auto-generate nomor dokumen (SINV-2024-0001)
- **Workflow/Status** — State machine untuk approval flow
- **Permissions** — Role-based akses per field atau per dokumen

================================

Berikut daftar lengkap komponen struktural dan interaktif selain field type, terorganisir berdasarkan kategorinya dari ERPNext (Frappe) dan Odoo.

***

# Komponen Struktural & Layout Lainnya di ERP

## Form Structure Components

Ini adalah "kerangka" sebuah form — bukan input data, tapi yang mengatur tampilan dan alur navigasi.

### Odoo — Structural Components

Odoo membagi komponen form menjadi dua layer: **Structural** (kerangka) dan **Semantic** (interaktif): [tutorialopenerp.wordpress](https://tutorialopenerp.wordpress.com/2025/07/15/odoo-18-technical-documentation-part-8/)

| Komponen | Tag XML | Fungsi |
|---|---|---|
| **Sheet** | `<sheet>` | Wrapper utama form yang membuat layout responsif di layar kecil |
| **Header** | `<header>` | Bar di atas form untuk workflow buttons dan status bar |
| **Notebook** | `<notebook>` | Container tab — berisi banyak `<page>` |
| **Page (Tab)** | `<page string="...">` | Satu tab di dalam notebook; bisa invisible berdasarkan kondisi |
| **Group** | `<group col="4">` | Layout field dalam kolom-kolom; `col` menentukan jumlah kolom (default 2) |
| **Nested Group** | `<group>` dalam `<group>` | Membuat 2 grup berdampingan (kiri-kanan) dalam 1 baris |
| **Separator** | `<separator string="...">` | Garis pemisah vertikal antar field dalam group, dengan opsional judul |
| **Newline** | `<newline/>` | Paksa field berikutnya mulai di baris baru dalam group |
| **Div: Button Box** | `<div name="button_box">` | Area khusus di pojok kanan atas form untuk **Smart Buttons** |
| **Div: Title** | `<div class="oe_title">` | Wrapper judul dokumen (biasanya h1 + field name) |
| **Label** | `<label for="...">` | Label manual untuk field yang berada di luar group |

### ERPNext (Frappe) — Layout Fields

Frappe menggunakan field khusus sebagai pemisah layout, bukan tag HTML: [frappe](https://frappe.io/blog/engineering/fixing-long-forms)

| Komponen | Field Type | Fungsi |
|---|---|---|
| **Section Break** | `Section Break` | Memulai seksi baru (full-width), bisa punya judul & deskripsi |
| **Collapsible Section** | `Section Break` + `collapsible: true` | Seksi yang bisa dilipat/dibuka, dengan opsi `Collapsible Depends On` untuk kondisi otomatis |
| **Column Break** | `Column Break` | Membagi seksi menjadi 2 kolom (kiri & kanan) |
| **Tab Break** | `Tab Break` | Membuat navigasi tab di dalam form |
| **Fold** | `Fold` *(deprecated)* | Pemisah lama yang menyembunyikan konten di bawahnya; digantikan collapsible sections  [frappe](https://frappe.io/blog/engineering/fixing-long-forms) |

***

## Navigasi & Aksi di Form

### Smart Buttons (Odoo)

Stat buttons / smart buttons adalah tombol kecil di pojok kanan atas form yang menampilkan jumlah record terkait. Contoh: "3 Invoices", "5 Delivery Orders" pada Sales Order. [github](https://github.com/Desdaemon/odoo-lsp/blob/main/odoo.rng)

```xml
<div name="button_box" class="oe_button_box">
  <button class="oe_stat_button" icon="fa-truck" type="object" name="action_view_delivery">
    <span class="o_stat_text">Deliveries</span>
  </button>
</div>
```

### Status Bar (Odoo)

Visualisasi pipeline workflow sebagai panah berurutan di bagian atas form: [tutorialopenerp.wordpress](https://tutorialopenerp.wordpress.com/2025/07/15/odoo-18-technical-documentation-part-8/)

```xml
<header>
  <button name="action_confirm" type="object" string="Confirm" class="btn-primary"/>
  <field name="state" widget="statusbar" statusbar_visible="draft,sent,sale,done"/>
</header>
```

### Action Buttons (ERPNext)

Di ERPNext, tombol aksi muncul di header form secara otomatis berdasarkan workflow. Custom button bisa ditambahkan via **Client Script**.

***

## Chatter & Komunikasi

Komponen sosial/komunikasi yang khas di ERP modern:

| Komponen | Platform | Fungsi |
|---|---|---|
| **Chatter** (`oe_chatter`) | Odoo | Panel kanan/bawah form: kirim pesan, log note, lihat aktivitas, followers, riwayat perubahan field |
| **Activity Widget** | Odoo | Bagian dari chatter untuk jadwal & tracking aktivitas (call, meeting, email, to-do) |
| **Thread / Comment** | ERPNext | Kolom komentar & email thread di bawah setiap form |
| **Assignment / Followers** | Keduanya | Assign user ke dokumen, notifikasi otomatis |
| **Timeline / Log** | Keduanya | Audit trail perubahan nilai field (siapa ubah apa & kapan) |

***

## List View Components

Komponen dalam tampilan daftar (bukan form): [tutorialopenerp.wordpress](https://tutorialopenerp.wordpress.com/2025/07/15/odoo-18-technical-documentation-part-8/)

| Komponen | Fungsi |
|---|---|
| **Control Row** | Baris di bawah child table dengan tombol kustom: "Add a Product", "Add a Section", "Add a Note" |
| **Header (List)** | Tombol aksi yang muncul di atas list saat user memilih/centang record |
| **Groupby Header** | Header baris group saat list di-group, bisa punya field & button tambahan |
| **Decoration** | Warna kondisional per baris: `decoration-success`, `danger`, `warning`, `info`, `muted`, `bf` (bold), `it` (italic) |
| **Column Optional** | Kolom yang bisa di-show/hide oleh user via icon di pojok kanan header tabel |
| **Sum / Avg Footer** | Baris total/rata-rata di bawah kolom numerik |
| **Progress Bar Column** | Kolom yang tampil sebagai bar progress (`widget="progressbar"`) |

***

## Search & Filter Components

Komponen di area pencarian/filter: [tutorialopenerp.wordpress](https://tutorialopenerp.wordpress.com/2025/07/15/odoo-18-technical-documentation-part-8/)

| Komponen | Fungsi |
|---|---|
| **Search Box** | Input pencarian utama dengan dropdown suggestion |
| **Filter Button** | Preset filter yang bisa diklik (contoh: "My Records", "Overdue") |
| **Group By Button** | Pengelompokan data (contoh: Group by Customer, by Month) |
| **Separator** | Garis pemisah antar filter dalam dropdown |
| **Search Panel** | Panel filter di sebelah kiri list/kanban; tampilkan kategori & jumlah record |
| **Favorites** | Simpan kombinasi filter + group by sebagai favorit |

***

## Kanban Card Components

Komponen di dalam kartu kanban: [tutorialopenerp.wordpress](https://tutorialopenerp.wordpress.com/2025/07/15/odoo-18-technical-documentation-part-8/)

| Komponen | Fungsi |
|---|---|
| **Card Template** | Template QWeb (`t-name="card"`) yang menentukan isi kartu |
| **Menu Template** | Dropdown `⋮` di sudut kartu: Edit, Delete, Arsip, pilih warna |
| **Kanban Progressbar** | Bar progress di atas setiap kolom group berdasarkan status |
| **Kanban Color Picker** | Widget pilih warna kartu (disimpan ke field `color` Integer) |
| **Footer Section** | Bagian bawah kartu: avatar user, tanggal, badge status |
| **Quick Create** | Form mini munculkan langsung di kolom kanban (tanpa buka form penuh) |

***

## Dialog & Overlay Components

| Komponen | Platform | Fungsi |
|---|---|---|
| **Modal / Dialog** | Keduanya | Pop-up form untuk input cepat, konfirmasi, atau detail record |
| **Wizard** | Odoo | Multi-step dialog dengan tombol Next/Back/Finish untuk proses kompleks |
| **Quick Entry** | ERPNext | Dialog mini untuk buat record baru dari field Link |
| **Confirm Dialog** | Keduanya | Pop-up konfirmasi sebelum aksi destruktif (delete, submit) — via atribut `confirm="..."` pada button |
| **Toast Notification** | Keduanya | Notifikasi kecil di pojok layar untuk feedback sukses/error/warning |

***

## Print & Export Components

| Komponen | Fungsi |
|---|---|
| **Print Format** | Template PDF/HTML untuk cetak dokumen (invoice, PO, delivery slip) |
| **Letter Head** | Header & footer perusahaan pada print |
| **Export XLSX** | Export list view ke Excel (bisa di-disable per view) |
| **Report Builder** | Query report dengan kolom kustom, filter, subtotal, chart |
| **Dashboard Shortcuts** | Link ke report/view dari halaman dashboard |

=================================

nah, karena referensinya odoo dan erpnext, serta masing2 ada kurang dan lebihnya

maka rangkum ya, kalau bisa pengennya punya component selengkap mungkin

dan paling *PENTING*, saya pengennya itu component bisa di pake di json maupun di html (semacam webcomponent), baik custom component nanti juga harus mendukung json dan html. 

buat document design dan implementation plan nya dulu ya sebelum eksekusi