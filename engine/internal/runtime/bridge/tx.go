package bridge

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const defaultTxTimeout = 30 * time.Second

type txManager struct {
	db *gorm.DB
}

func newTxManager(db *gorm.DB) *txManager {
	return &txManager{db: db}
}

func (t *txManager) RunTx(parent *Context, fn func(tx *Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTxTimeout)
	defer cancel()

	return t.db.WithContext(ctx).Transaction(func(gormTx *gorm.DB) error {
		txCtx := parent.cloneWithTx(gormTx)
		err := fn(txCtx)
		if err != nil {
			return err
		}
		if ctx.Err() == context.DeadlineExceeded {
			return NewError(ErrTxTimeout, "transaction timeout exceeded")
		}
		return nil
	})
}
