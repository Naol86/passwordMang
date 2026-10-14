package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"passwordmanage/server/internal/app/notify"
)

// sendEmailArgs is stored as JSON in the job table, so it must never hold secrets (see notify.Email).
type sendEmailArgs struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

func (sendEmailArgs) Kind() string { return "send_email" }

type sendEmailWorker struct {
	river.WorkerDefaults[sendEmailArgs]
	sender EmailSender
}

func (w *sendEmailWorker) Work(ctx context.Context, job *river.Job[sendEmailArgs]) error {
	return w.sender.Send(ctx, notify.Email{To: job.Args.To, Subject: job.Args.Subject, Text: job.Args.Text})
}

type txOutbox struct {
	client *river.Client[pgx.Tx]
	tx     pgx.Tx
}

func (o txOutbox) QueueEmail(ctx context.Context, email notify.Email) error {
	_, err := o.client.InsertTx(ctx, o.tx, sendEmailArgs{To: email.To, Subject: email.Subject, Text: email.Text}, nil)
	return err
}
