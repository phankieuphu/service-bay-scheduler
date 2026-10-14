package database_provider

import (
	"context"

	"gorm.io/gorm"
)

type txCtxKey struct{}

// TxManager is the GORM-backed implementation of [ports.TxManager].
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// RunInTx implements [ports.TxManager]. Repositories built on top of
// [DBFromContext] automatically pick up the transaction started here.
func (m *TxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txCtxKey{}, tx))
	})
}

// DBFromContext returns the *gorm.DB to run a query against: the active
// transaction if ctx was produced by TxManager.RunInTx, otherwise
// fallback scoped to ctx.
func DBFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok {
		return tx
	}
	return fallback.WithContext(ctx)
}
