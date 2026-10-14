// Package random declares the randomness port for tokens and IDs, so tests can be deterministic.
package random

type Source interface {
	// Bytes returns n cryptographically secure random bytes.
	Bytes(n int) ([]byte, error)
}
