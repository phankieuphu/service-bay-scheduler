package ports

import "context"

// TxManager runs fn inside a single database transaction so writes made
// through multiple repositories (e.g. the vehicle row and its outbox
// row) commit or roll back together. A repository that must participate
// reads the active transaction back out of ctx itself; see
// [database_provider.DBFromContext].
type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
