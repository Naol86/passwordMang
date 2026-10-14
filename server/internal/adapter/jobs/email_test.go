package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"passwordmanage/server/internal/app/notify"
)

type fakeSender struct {
	sent []notify.Email
	err  error
}

func (f *fakeSender) Send(_ context.Context, email notify.Email) error {
	f.sent = append(f.sent, email)
	return f.err
}

func emailJob(args sendEmailArgs) *river.Job[sendEmailArgs] {
	return &river.Job[sendEmailArgs]{JobRow: &rivertype.JobRow{}, Args: args}
}

func TestSendEmailWorkerDelivers(t *testing.T) {
	sender := &fakeSender{}
	worker := &sendEmailWorker{sender: sender}

	err := worker.Work(context.Background(), emailJob(sendEmailArgs{To: "ada@example.com", Subject: "Hi", Text: "Hello"}))
	if err != nil {
		t.Fatal(err)
	}
	want := notify.Email{To: "ada@example.com", Subject: "Hi", Text: "Hello"}
	if len(sender.sent) != 1 || sender.sent[0] != want {
		t.Fatalf("sent = %+v, want one %+v", sender.sent, want)
	}
}

func TestSendEmailWorkerReturnsErrorSoRiverRetries(t *testing.T) {
	sendErr := errors.New("smtp down")
	worker := &sendEmailWorker{sender: &fakeSender{err: sendErr}}

	err := worker.Work(context.Background(), emailJob(sendEmailArgs{To: "ada@example.com"}))
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected the send error to propagate, got %v", err)
	}
}
