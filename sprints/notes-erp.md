# Divisi marketing
- butuh ada manajement data/files dari PO, approval, dll yang menyangkut dengan divisi ini  (file manager dan historical file untuk data dari tiap flow approval). Butuh EDMS + Directory Project Structure

# Divisi Estimation
- butuh ada manajement data/files dari approval, dll yang menyangkut dengan divisi ini (file manager dan historical file untuk data dari tiap flow approval). Butuh EDMS + Directory Project Structure
- bisa langsung mengakses data-data harga barang dari divisi Purchasing dan harga jasa dari divisi Administration
- bisa langsung melihat harga-harga barang dari seluruh dunia untuk kebutuhan konstruksi
- bisa mengakses data-data proyek dari divisi terkait yakni divisi construction, divisi engineering, dan marketing

# Divisi Engineering
- butuh ada manajement data/files dari approval, dll yang menyangkut dengan divisi ini (file manager dan historical file untuk data dari tiap flow approval). Butuh EDMS + Directory Project Structure
- Ada project management system untuk mengatur task ke PIC terkait
- Ada mekanisme untuk timesheet, dimana PIC terkait bisa Start, Pause, Dan Finish gitu, dan bisa menyertakan alasan kenapa Pause untuk dilakukan tinjauan dan approval dari atasan
- Bisa menghitung langsung work hour dari mekanisme timesheet yang bisa Start, Pause, dan Finish, selama ini pake excel/manual, jadi tidak akurat karena kalau Pause jadi terhitung keseluruhan
- Butuh Gantt Chart 
- Bisa mengakses data-data proyek dari divisi terkait yakni divisi construction, divisi estimation, dan marketing

# Divisi Construction
- butuh ada manajement data/files dari approval, dll yang menyangkut dengan divisi ini (file manager dan historical file untuk data dari tiap flow approval). Butuh EDMS + Directory Project Structure
- Ada project management system untuk mengatur task ke PIC terkait
- Ada mekanisme untuk timesheet, dimana PIC terkait bisa Start, Pause, Dan Finish gitu, dan bisa menyertakan alasan kenapa Pause untuk dilakukan tinjauan dan approval dari atasan
- Bisa menghitung langsung work hour dari mekanisme timesheet yang bisa Start, Pause, dan Finish, selama ini pake excel/manual, jadi tidak akurat karena kalau Pause jadi terhitung keseluruhan
- Butuh Gantt Chart 
- Bisa mengakses data-data proyek dari divisi terkait yakni divisi engineering, divisi estimation, divisi marketing, dan divisi purchasing serta divisi Administration
- Ada semacam system untuk forecasting/alerting jadi proyek2 yg berjalan dapat terpantau harus order barang apa saja


# Divisi Purchasing
- ada smart comparison AI harga dari Harga-harga dari tiap vendor, yang otomatis menyeleksi harga terbaik
- ada system detection saat ada permintaan request pengadaan material dari divisi lain dalam proyek yang melebihi batas anggaran dan RAB (quantity dan price)
- ada smart tracking, dimana bisa ke track dari delivery barang, yg kemudian di terima, lalu di serahkan ke proyek
- Automatic update pricing terhadap avg rata-rata penawaran dari vendor yg pernah di tawarkan, semisal rata-rata per 3 bulan harga barang A
- Butuh vendor portal, agar vendor bisa mengirimkan secara langsung penawaran dalam system 
- butuh ada manajement data/files dari approval, dll yang menyangkut dengan divisi ini  (file manager dan historical file untuk data dari tiap flow approval). Butuh EDMS + Directory Project Structure

# Divisi Administration
- butuh ada manajement data/files dari PO, approval, dll yang menyangkut dengan divisi ini  (file manager dan historical file untuk data dari tiap flow approval). Butuh EDMS + Directory Project Structure
- Butuh vendor portal, agar vendor bisa mengirimkan secara langsung penawaran dalam system 
- Automatic update pricing terhadap avg rata-rata penawaran dari vendor yg pernah di tawarkan, semisal rata-rata per 3 bulan harga sewa/jasa supervisor

# Accounting
sesuai dengan excel, intinya bisa seperti accurate (akunting system yg mendukung PPH, dll dan aturan indonesia)

# HRGA
- ada performance appraisal dimana tiap karyawan bisa menginput sendiri target pekerjaan, nanti ada semacam bobot untuk terhitung dengan approval atasan/review
- tiap karyawan/user ada chat AI terkait hal-hal dalam ketenagakerjaan dan aturan perusahaan
- ada chat AI untuk analisa data dari divisi HR
- ada portal job untuk pelamar
- bisa membuat template dokumen, yang nantinya bisa custom, dan jika ada divisi lain butuh tinggal download dan automatic bisa tergenerate datanya sesuai data karyawan yang request, semisal template dokumen untuk cuti, dsb
- butuh ada manajement data/files dari approval, dll yang menyangkut dengan divisi ini  (file manager dan historical file untuk data dari tiap flow approval). Butuh EDMS + Directory Project Structure

# QHSE (Quality, Health, Safety & environtment)
- butuh ada manajement data/files dari approval, dll yang menyangkut dengan divisi ini  (file manager dan historical file untuk data dari tiap flow approval, maupun dokumen inspeksi). Butuh EDMS + Directory Project Structure
- tersinkronisasi dengan proyek divisi construction, dan lain-lain, jadi bisa membuat list pekerjaan dari yang butuh divisi QHSE terkait inspeksi pekerjaan, jadi list task/pekerjaan bisa di atur apakah butuh QHSE atau tidak, dan bisa memilih template dokumen yang akan diguanakan terkait ISO2
- Ada WA notifikasi jika ada pekerjaan yang sudah di inspeksi harus segera di evaluasi, akan masuk ke manager terkait, semisal manager construction, lalu manager construction akan assign tugas ke PIC, bersangkutan, jika sudah oke, akan masuk ke divisi QHSE. nah tiap proses itu ada notifikasi via WA dan email baik ke construction maanger, QHSE, atau PIC terkait
- ada offline mode dan bisa auto sync seperti halnya whatsapp, semisal di android dimana jika sedang inspeksi dan tidak ada sinyal internet tetap bisa mengerjakan inspeksi, dan jika sudah ada sinyal itnernet otomatis akan tersync ke server

# Divisi Admin/IT System ERP
- Ada pengaturan workflow approval
- Ada pengaturan user roles, permission bahkan Record rules seperti di odoo

# Notes
- EDMS + Directory Project Structure, maksudnya adalah, semisal ada project A,dengan task C di divisi engineering
maka akan automatic bikin struktur directory rapih
/Project A
  --/Engineering
    --/Task C
        --/catatan.pdf
  --/Construction
    --/Task B
        --/approval.pdf

jadi nanti bisa lebih mudah dalam memetakan file dan mencari file terkait
