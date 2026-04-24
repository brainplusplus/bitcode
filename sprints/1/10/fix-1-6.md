di model, buat mekanisme auto increment buat primary key :
1. auto increment
2. composite key
3. UUID, ada opsi UUIDv7, UUIDv4 dan UUID yang generate dari satu atau lebih column dengan namespace, semisal di table customers, bisa generate UUID dari namespace "customers" dan value format dari beberapa column, maupun session,system,settings, dan bahkan bisa substring (ada beberapa built in feature, kamu atur aja kira2 builtin feature functionnya yang butuh dalam skala ERP itu apa saja, oia salah satunya juga ada sequence number utk built in functionnya) dll "{data.nik}-{data.customer_type}-{time.now}-{time.year}-{session.user_id}-{sequence(6)}-{setting.xxx}-{substring(data.nik,0,3)}", kalau belum ada fungsi untuk generate UUID dengan namespace dan value dengan format
4. natural key
Natural key menggunakan nilai bisnis yang sudah ada sebagai identifier, misalnya NIK KTP, NPWP, atau kode produk. Berbeda dengan surrogate key (auto-increment/UUID) yang murni teknis, natural key memiliki makna kontekstual. Risikonya adalah nilai bisa berubah atau duplikat jika tidak dikontrol ketat.
5. String / VARCHAR (Naming Series). ini mirip sama yang generate UUID dari namespace dan value, bedanya tidak berupa generate UUID, tapi murni dari generate valuenya, semisal  "{data.nik}-{data.customer_type}-{time.now}-{time.year}-{session.user_id}-{sequence(6)}-{setting.xxx}-{substring(data.nik,0,3)}"
6. manual input

oia, utk opsi 1,3,5 itu otomatis di form CRUDnya gak muncul ya saat new/create dan update

jadi butuh table sejenis sequence_numbers gitu, ini contoh aja ya ada sebuah aplikasi konsep sequence numbernya begini :

{
        "id": "4690281b-09bb-7f86-6a6a-b8f071aee7b8",
        "code": "INV_STOCK_MOVE_HISTORY_ID",
        "company_id": "XXXX",
        "name": "Inventory Stock Move History ID",
        "table_name": "inv_item_stock_move_histories",
        "field_name": "code",
        "module_id": "49384f29-6595-8bab-4ff1-689e71d23401",
        "module_row": {
            "id": "49384f29-6595-8bab-4ff1-689e71d23401",
            "code": "INV",
            "name": "Inventory"
        },
        "group_rows": [
            {
                "code": "default",
                "numbering_default": "MH/2025/00001",
                "prefix": "MH/{{now.format('y')}}/",
                "suffix": "",
                "sequence_length": 6,
                "sequence_key": "MH/{{now.format('y')}}/",
                "is_default": true,
                "sequences": [
                    {
                        "key": "MH/{{now.format('y')}}/",
                        "step_sequence": 1,
                        "next_sequence": 1
                    },
                    {
                        "key": "MH/2025/",
                        "step_sequence": 1,
                        "next_sequence": 12
                    },
                    {
                        "key": "MH/2026/",
                        "step_sequence": 1,
                        "next_sequence": 3
                    }
                ]
            }
        ],
        "is_default": true,
        "is_active": true
    }
    
    jadi pointnya sequencesnya bisa reset, semisal ganti bulan, atau tahun berdasarkan key gitu. inget ya ini referensi saja, bikin yang lebih bagus dan bisa menghandle banyak kasus


oia untuk UUID yg generate dari format tertentu (point no 3) maupun varchar naming series, bisa digunakan di field tertentu ya selain primary key. semisal case di atas kan opsinya ada 2 yg saya tahu :

1. PK nya UUID yg generate dari field_name/column code, dimana field_name/column code itu disetting untuk generate  "{data.nik}-{data.customer_type}-{time.now}-{time.year}-{session.user_id}-{sequence(6)}-{setting.xxx}-{substring(data.nik,0,3)}"

lalu otomatis PK nya cukup proses dari column code yang sudah di format

2. PK nya UUID yg generate dari format  "{data.nik}-{data.customer_type}-{time.now}-{time.year}-{session.user_id}-{sequence(6)}-{setting.xxx}-{substring(data.nik,0,3)}". semisal gak ada field_name/column code


dan fungsi untuk generate sequence number harus mengatasi race condition

saya butuh masukkan jika memang masih ada yg kurang

dan inget ya kalau sudah selesai update dokumentasi terkait, dan juga update sample, serta commit dan push
