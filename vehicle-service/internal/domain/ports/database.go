package ports

import (
	"context"
)

type IDatabaseService interface {
	GetList()
	Insert(ctx context.Context, items any)
	Delete()
	Update()
	Ping()
	Close()
}
