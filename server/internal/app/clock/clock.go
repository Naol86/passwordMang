// Package clock declares the time port so use cases never call time.Now directly.
package clock

import "time"

type Clock interface {
	Now() time.Time
}
