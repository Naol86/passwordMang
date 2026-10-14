package page

import (
	"testing"
	"time"

	"passwordmanage/server/internal/domain/errs"
)

func TestNewRequestDefaultsAndBounds(t *testing.T) {
	ptr := func(n int) *int { return &n }

	tests := []struct {
		name      string
		limit     *int
		wantLimit int
		wantErr   bool
	}{
		{name: "nil uses default", limit: nil, wantLimit: DefaultLimit},
		{name: "min", limit: ptr(1), wantLimit: 1},
		{name: "max", limit: ptr(MaxLimit), wantLimit: MaxLimit},
		{name: "zero rejected", limit: ptr(0), wantErr: true},
		{name: "above max rejected", limit: ptr(MaxLimit + 1), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewRequest(tc.limit, nil)
			if tc.wantErr {
				if !errs.IsKind(err, errs.KindInvalid) {
					t.Fatalf("expected invalid error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Limit != tc.wantLimit {
				t.Fatalf("limit = %d, want %d", got.Limit, tc.wantLimit)
			}
		})
	}
}

type position struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

func TestCursorRoundTrip(t *testing.T) {
	want := position{CreatedAt: time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC), ID: "abc"}
	cursor, err := EncodeCursor(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeCursor[position](cursor)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDecodeCursorRejectsTampering(t *testing.T) {
	if _, err := DecodeCursor[position]("not*base64"); !errs.IsKind(err, errs.KindInvalid) {
		t.Fatalf("expected invalid error, got %v", err)
	}
}
