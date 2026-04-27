Berikut versi **ASCII UI** dari halaman **Odoo Group (Access Rights / Groups Settings)** + contoh isi tiap tab biar kebayang struktur datanya.

---

# 🧩 MAIN FORM (GROUP)

```
+----------------------------------------------------------------------------------+
| [New]  Groups > Technical / A warning can be set on a partner (Account)     1/76 |
+----------------------------------------------------------------------------------+

Application : Technical
Name        : A warning can be set on a partner (Account)
Share Group : [ ]

Tabs:
[ Users ] [ Inherited ] [ Menus ] [ Views ] [ Access Rights ] [ Record Rules ] [ Notes ]
```

---

# 👥 TAB: Users

```
+------------------------------------------------------+
| Users                                                |
+------------------------------------------------------+
| Name                     | Login         | Active     |
|------------------------------------------------------|
| Administrator            | admin         | ✔          |
| Demo User                | demo          | ✔          |
| Finance Manager          | finance_mgr   | ✔          |
+------------------------------------------------------+
| + Add a line                                        |
+------------------------------------------------------+
```

---

# 🔗 TAB: Inherited (Group turunan)

```
+------------------------------------------------------+
| Inherited Groups                                     |
+------------------------------------------------------+
| Group Name                                           |
|------------------------------------------------------|
| User: Internal User                                  |
| Accounting / Accountant                              |
| Sales / User                                         |
+------------------------------------------------------+
| + Add a line                                        |
+------------------------------------------------------+
```

👉 Artinya group ini otomatis mewarisi permission dari group lain.

---

# 📂 TAB: Menus

```
+------------------------------------------------------+
| Menus                                                |
+------------------------------------------------------+
| Menu Name                     | Parent                |
|------------------------------------------------------|
| Accounting                    | -                     |
| Customers                     | Accounting            |
| Vendors                       | Accounting            |
| Reports                       | Accounting            |
+------------------------------------------------------+
| + Add a line                                        |
+------------------------------------------------------+
```

👉 Menu yang bisa diakses oleh group ini.

---

# 🧾 TAB: Views

```
+------------------------------------------------------+
| Views                                                |
+------------------------------------------------------+
| View Name                | Model        | Type        |
|------------------------------------------------------|
| res.partner.form         | res.partner  | Form        |
| res.partner.tree         | res.partner  | List        |
| account.move.form        | account.move | Form        |
+------------------------------------------------------+
| + Add a line                                        |
+------------------------------------------------------+
```

👉 Biasanya jarang dipakai langsung kecuali custom.

---

# 🔐 TAB: Access Rights (INI PALING PENTING)

```
+----------------------------------------------------------------------------------+
| Access Rights                                                                    |
+----------------------------------------------------------------------------------+
| Name                | Model          | Read | Write | Create | Delete             |
|----------------------------------------------------------------------------------|
| Partner Access      | res.partner    |  ✔   |   ✔   |   ✔    |   ✖                |
| Invoice Access      | account.move   |  ✔   |   ✔   |   ✔    |   ✔                |
| Product Access      | product.product|  ✔   |   ✖   |   ✖    |   ✖                |
+----------------------------------------------------------------------------------+
| + Add a line                                                                    |
+----------------------------------------------------------------------------------+
```

👉 Ini mapping ke `ir.model.access`

---

# 📏 TAB: Record Rules (Row-level security)

```
+----------------------------------------------------------------------------------+
| Record Rules                                                                     |
+----------------------------------------------------------------------------------+
| Name                  | Model        | Domain                          | Perms   |
|----------------------------------------------------------------------------------|
| Own Customers Only    | res.partner  | [('user_id','=',user.id)]       | R/W/C   |
| Company Restriction   | account.move | [('company_id','=',company_id)] | R       |
+----------------------------------------------------------------------------------+
| + Add a line                                                                    |
+----------------------------------------------------------------------------------+
```

👉 Ini filter data (level SQL/domain), beda dengan access rights.

---

# 📝 TAB: Notes

```
+------------------------------------------------------+
| Notes                                                |
+------------------------------------------------------+
| This group is used to manage warning on partner      |
| accounts in accounting module.                       |
|                                                     |
| Only assigned to finance-related users.             |
+------------------------------------------------------+
```

---

# 🔥 BONUS: GAMBAR BESAR RELASI

```
Group
 ├── Users
 ├── Inherited Groups
 ├── Access Rights (Model Level)
 ├── Record Rules (Row Level)
 ├── Menus (UI Access)
 └── Views (Optional UI control)
```

---
