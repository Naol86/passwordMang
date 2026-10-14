package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"passwordmanage/server/internal/adapter/postgres/sqlcgen"
	"passwordmanage/server/internal/app/tx"
)

// TxRunner implements tx.Runner. build turns an open transaction into the
// repositories a feature's use cases need.
type TxRunner[R any] struct {
	pool  *pgxpool.Pool
	build func(pgx.Tx) R
}

var _ tx.Runner[struct{}] = (*TxRunner[struct{}])(nil)

func NewTxRunner[R any](pool *pgxpool.Pool, build func(pgx.Tx) R) *TxRunner[R] {
	return &TxRunner[R]{pool: pool, build: build}
}

func (r *TxRunner[R]) Run(ctx context.Context, fn func(ctx context.Context, repos R) error) error {
	return pgx.BeginFunc(ctx, r.pool, func(t pgx.Tx) error {
		return fn(ctx, r.build(t))
	})
}

func (r *TxRunner[R]) RunInOrg(ctx context.Context, orgID uuid.UUID, fn func(ctx context.Context, repos R) error) error {
	if orgID == uuid.Nil {
		return errors.New("RunInOrg: organization ID is required")
	}
	return pgx.BeginFunc(ctx, r.pool, func(t pgx.Tx) error {
		// Transaction-local, so the setting never leaks to the next user of the pooled connection.
		if err := sqlcgen.New(t).SetOrgScope(ctx, orgID.String()); err != nil {
			return err
		}
		return fn(ctx, r.build(t))
	})
}
