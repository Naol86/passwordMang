package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestAsFindsWrappedDomainError(t *testing.T) {
	cause := errors.New("row missing")
	domainErr := NotFound("folder_not_found", "Folder not found.").Wrap(cause)
	wrapped := fmt.Errorf("load folder: %w", domainErr)

	got, ok := As(wrapped)
	if !ok {
		t.Fatal("expected a domain error in the chain")
	}
	if got.Kind != KindNotFound || got.Code != "folder_not_found" {
		t.Fatalf("got kind %v code %q", got.Kind, got.Code)
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("expected the cause to stay reachable with errors.Is")
	}
}

func TestAsReturnsFalseForPlainErrors(t *testing.T) {
	if _, ok := As(errors.New("boom")); ok {
		t.Fatal("plain error must not be treated as a domain error")
	}
	if IsKind(nil, KindInvalid) {
		t.Fatal("nil must not match any kind")
	}
}

func TestIsKind(t *testing.T) {
	err := fmt.Errorf("check: %w", Forbidden("not_allowed", "Not allowed."))
	if !IsKind(err, KindForbidden) {
		t.Fatal("expected forbidden kind")
	}
	if IsKind(err, KindNotFound) {
		t.Fatal("did not expect not-found kind")
	}
}
