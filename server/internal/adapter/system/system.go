// Package system provides the real clock and the operating system's secure randomness.
package system

import (
	"crypto/rand"
	"time"
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC() }

type Random struct{}

func (Random) Bytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
