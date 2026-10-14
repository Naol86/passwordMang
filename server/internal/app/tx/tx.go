// Package tx declares the transaction port that use cases run their work through.
package tx

import (
	"context"

	"github.com/google/uuid"
)

// Runner executes fn in one database transaction and hands it repositories (R)
// bound to that transaction. fn's error rolls the transaction back.
type Runner[R any] interface {
	// Run is for work outside any organization, such as account sign-up.
	Run(ctx context.Context, fn func(ctx context.Context, repos R) error) error
	// RunInOrg scopes the transaction to one organization so row-level
	// security only exposes that organization's rows (ORG-04).
	RunInOrg(ctx context.Context, orgID uuid.UUID, fn func(ctx context.Context, repos R) error) error
}
