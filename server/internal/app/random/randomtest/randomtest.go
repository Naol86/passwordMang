// Package randomtest provides a deterministic random source for tests. Never use it outside tests.
package randomtest

import "sync"

// Sequence returns bytes 0, 1, 2, ... wrapping at 255, so every call yields different output.
type Sequence struct {
	mu   sync.Mutex
	next byte
}

func (s *Sequence) Bytes(n int) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, n)
	for i := range out {
		out[i] = s.next
		s.next++
	}
	return out, nil
}
