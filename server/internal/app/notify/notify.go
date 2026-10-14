// Package notify declares how use cases queue outgoing email.
package notify

import "context"

// Email is stored in the job queue until sent, so it must never contain secrets
// such as invite tokens or reset links; jobs that need one generate it at send time.
type Email struct {
	To      string
	Subject string
	Text    string
}

// Outbox queues email inside the caller's transaction, so nothing is sent if it rolls back.
type Outbox interface {
	QueueEmail(ctx context.Context, email Email) error
}
