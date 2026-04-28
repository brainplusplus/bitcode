package bridge

import "gorm.io/gorm"

type dbBridge struct {
	db *gorm.DB
}

func newDBBridge(db *gorm.DB) *dbBridge {
	return &dbBridge{db: db}
}

func (d *dbBridge) Query(sql string, args ...any) ([]map[string]any, error) {
	var results []map[string]any
	tx := d.db.Raw(sql, args...).Scan(&results)
	if tx.Error != nil {
		return nil, NewError(ErrInternalError, tx.Error.Error())
	}
	return results, nil
}

func (d *dbBridge) Execute(sql string, args ...any) (*ExecDBResult, error) {
	tx := d.db.Exec(sql, args...)
	if tx.Error != nil {
		return nil, NewError(ErrInternalError, tx.Error.Error())
	}
	return &ExecDBResult{RowsAffected: tx.RowsAffected}, nil
}

func (d *dbBridge) withTx(tx *gorm.DB) *dbBridge {
	return &dbBridge{db: tx}
}
