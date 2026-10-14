package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"passwordmanage/server/internal/adapter/postgres"
	"passwordmanage/server/internal/adapter/postgres/sqlcgen"
	"passwordmanage/server/internal/testdb"
)

type repos struct {
	q  *sqlcgen.Queries
	tx pgx.Tx
}

func newRunner(t *testing.T) *postgres.TxRunner[repos] {
	pool := testdb.New(t)
	return postgres.NewTxRunner(pool, func(tx pgx.Tx) repos { return repos{q: sqlcgen.New(tx), tx: tx} })
}

func TestRunInOrgSetsScopeForThatTransactionOnly(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()
	orgID := uuid.New()

	err := runner.RunInOrg(ctx, orgID, func(ctx context.Context, r repos) error {
		got, err := r.q.CurrentOrgScope(ctx)
		if err != nil {
			return err
		}
		if got != orgID.String() {
			t.Errorf("scope inside RunInOrg = %q, want %q", got, orgID)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A later transaction, possibly on the same pooled connection, must start unscoped.
	err = runner.Run(ctx, func(ctx context.Context, r repos) error {
		got, err := r.q.CurrentOrgScope(ctx)
		if err != nil {
			return err
		}
		if got != "" {
			t.Errorf("scope leaked into next transaction: %q", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunRollsBackOnError(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()
	failure := errors.New("use case failed")

	err := runner.Run(ctx, func(ctx context.Context, r repos) error {
		if _, err := r.tx.Exec(ctx, "CREATE TABLE rollback_probe (id int)"); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("expected the use-case error, got %v", err)
	}

	err = runner.Run(ctx, func(ctx context.Context, r repos) error {
		var exists bool
		if err := r.tx.QueryRow(ctx, "SELECT to_regclass('rollback_probe') IS NOT NULL").Scan(&exists); err != nil {
			return err
		}
		if exists {
			t.Error("table from the failed transaction should not exist")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunInOrgRequiresOrgID(t *testing.T) {
	runner := newRunner(t)
	err := runner.RunInOrg(context.Background(), uuid.Nil, func(context.Context, repos) error { return nil })
	if err == nil {
		t.Fatal("expected an error for a missing organization ID")
	}
}
