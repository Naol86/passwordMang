// Package jobs runs background work on River, a PostgreSQL-backed queue, so no extra broker is needed.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"passwordmanage/server/internal/app/notify"
)

const (
	defaultQueueWorkers = 10
	// Finished jobs are deleted quickly so their arguments do not linger in the database.
	finishedJobRetention = time.Hour
)

// EmailSender delivers one email; the mail adapter implements it.
type EmailSender interface {
	Send(ctx context.Context, email notify.Email) error
}

type Queue struct {
	client *river.Client[pgx.Tx]
}

// Migrate creates or updates River's own tables.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("create job migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("apply job migrations: %w", err)
	}
	return nil
}

func New(pool *pgxpool.Pool, sender EmailSender, log *slog.Logger) (*Queue, error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &sendEmailWorker{sender: sender})

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:                      map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: defaultQueueWorkers}},
		Workers:                     workers,
		Logger:                      log,
		CompletedJobRetentionPeriod: finishedJobRetention,
		CancelledJobRetentionPeriod: finishedJobRetention,
		DiscardedJobRetentionPeriod: finishedJobRetention,
	})
	if err != nil {
		return nil, fmt.Errorf("create job queue: %w", err)
	}
	return &Queue{client: client}, nil
}

func (q *Queue) Start(ctx context.Context) error { return q.client.Start(ctx) }

func (q *Queue) Stop(ctx context.Context) error { return q.client.Stop(ctx) }

// Outbox returns a notify.Outbox that enqueues inside t, so email is only sent if t commits.
func (q *Queue) Outbox(t pgx.Tx) notify.Outbox {
	return txOutbox{client: q.client, tx: t}
}
