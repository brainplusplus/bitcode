workflow sudah?

resources mirip laravel
views diganti pages
/app => internal, /web => public

background job
cron

repository => can used by another process
datasources
field model/form bisa dependent, misal province, city, dll

create components stencil
agent => coder tools, models tools, form tools, report tools
email template
email editor

i18n kalau bisa digabung jadi 1 utk 11 bahasa, jadi bukan id.json gitu, tapi lebih ke model.json

CRUD component stencil with export, pdf, etc support
openapi/swagger
rest api, websocket, graphql
website
i18n utk semua label di json

khusus database lain selain main (external), modelnya menggunakan namespace di jsonnya, creadential ada di .toml, dan ada opsi utk auto migrate dan auto generate model json, semisal external [database.ext1], namespace = "ext1"
mcp server buat vibe coding bitcode


=======
prefix_tables in module || done
query builder, can used process atau ts python || done
mongodb support || done
update namespace bitcode-engine menjadi bitcode-framework || done
go install di run.bat || done
ada version di field model json, buat menangani race condition || done
pdf viewer, image viewer, doc/xls/ppt viewer utk file upload atau file tertentu, youtube viewer || done
di model json, ada event2 seperti di laravel yakni before, on, after utk insert, update, delete yang bisa mengacu ke process || done
data seeder mendukung xlsx, csv maupun json, xml ala odoo dan bisa custom processing data seedernya || done
