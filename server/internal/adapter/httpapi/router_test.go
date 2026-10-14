package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"passwordmanage/server/internal/adapter/httpapi/apigen"
	"passwordmanage/server/internal/domain/errs"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func newTestRouter(t *testing.T, db Pinger) *gin.Engine {
	t.Helper()
	r, err := NewRouter(Deps{Log: discardLog, DB: db, Version: "test", Metrics: prometheus.NewRegistry()})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func serve(h http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), method, path, body))
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) apigen.ErrorDetail {
	t.Helper()
	var body apigen.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not the error shape: %v: %s", err, rec.Body.String())
	}
	return body.Error
}

func TestHealthzSetsRequestIDAndSecurityHeaders(t *testing.T) {
	rec := serve(newTestRouter(t, fakePinger{}), http.MethodGet, "/healthz", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get(headerRequestID) == "" {
		t.Error("missing request ID header")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("missing security headers: %v", rec.Header())
	}
}

func TestReadyzReportsDatabaseState(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   apigen.StatusStatus
	}{
		{name: "database up", wantStatus: http.StatusOK, wantBody: apigen.StatusStatusOk},
		{name: "database down", pingErr: errors.New("refused"), wantStatus: http.StatusServiceUnavailable, wantBody: apigen.StatusStatusUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(newTestRouter(t, fakePinger{err: tc.pingErr}), http.MethodGet, "/readyz", nil)
			var body apigen.Status
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.wantStatus || body.Status != tc.wantBody {
				t.Fatalf("got %d %q, want %d %q", rec.Code, body.Status, tc.wantStatus, tc.wantBody)
			}
		})
	}
}

func TestMetaReturnsVersion(t *testing.T) {
	rec := serve(newTestRouter(t, fakePinger{}), http.MethodGet, "/api/v1/meta", nil)
	var body apigen.Meta
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Version != "test" || body.ApiVersion != apiVersion {
		t.Fatalf("got %+v", body)
	}
}

func TestUnknownRouteUsesErrorShape(t *testing.T) {
	rec := serve(newTestRouter(t, fakePinger{}), http.MethodGet, "/api/v1/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	detail := decodeError(t, rec)
	if detail.Code != codeRouteNotFound {
		t.Errorf("code = %q", detail.Code)
	}
	if detail.RequestId == "" || detail.RequestId != rec.Header().Get(headerRequestID) {
		t.Errorf("requestId %q should match header %q", detail.RequestId, rec.Header().Get(headerRequestID))
	}
}

// errorEngine runs handler behind the same middleware the router uses for errors.
func errorEngine(handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(requestID(), recovery(discardLog), bodyLimit(16))
	r.POST("/test", handler)
	return r
}

func TestWriteErrorMapsDomainKinds(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{errs.Invalid("bad_name", "Bad name.", errs.FieldError{Field: "name", Message: "required"}), http.StatusBadRequest, "bad_name"},
		{errs.Unauthenticated("session_expired", "Sign in again."), http.StatusUnauthorized, "session_expired"},
		{errs.Forbidden("not_allowed", "Not allowed."), http.StatusForbidden, "not_allowed"},
		{errs.NotFound("item_not_found", "Not found."), http.StatusNotFound, "item_not_found"},
		{errs.Conflict("revision_conflict", "Changed elsewhere."), http.StatusConflict, "revision_conflict"},
		{errs.RateLimited("too_many_attempts", "Slow down."), http.StatusTooManyRequests, "too_many_attempts"},
	}
	for _, tc := range tests {
		t.Run(tc.wantCode, func(t *testing.T) {
			rec := serve(errorEngine(func(c *gin.Context) { writeError(c, discardLog, tc.err) }), http.MethodPost, "/test", nil)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := decodeError(t, rec).Code; got != tc.wantCode {
				t.Fatalf("code = %q, want %q", got, tc.wantCode)
			}
		})
	}
}

func TestWriteErrorIncludesFieldErrors(t *testing.T) {
	err := errs.Invalid("invalid_input", "Invalid.", errs.FieldError{Field: "limit", Message: "too big"})
	rec := serve(errorEngine(func(c *gin.Context) { writeError(c, discardLog, err) }), http.MethodPost, "/test", nil)
	detail := decodeError(t, rec)
	if detail.Fields == nil || len(*detail.Fields) != 1 || (*detail.Fields)[0].Field != "limit" {
		t.Fatalf("fields = %+v", detail.Fields)
	}
}

func TestWriteErrorHidesUnexpectedErrors(t *testing.T) {
	rec := serve(errorEngine(func(c *gin.Context) {
		writeError(c, discardLog, errors.New("pq: password authentication failed for user pm"))
	}), http.MethodPost, "/test", nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "password authentication") {
		t.Fatal("internal error details leaked to the client")
	}
	if decodeError(t, rec).Code != codeInternal {
		t.Fatal("expected internal_error code")
	}
}

func TestPanicBecomesInternalError(t *testing.T) {
	rec := serve(errorEngine(func(*gin.Context) { panic("boom") }), http.MethodPost, "/test", nil)
	if rec.Code != http.StatusInternalServerError || decodeError(t, rec).Code != codeInternal {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestBodyLimitReturns413(t *testing.T) {
	rec := serve(errorEngine(func(c *gin.Context) {
		if _, err := c.GetRawData(); err != nil {
			writeError(c, discardLog, invalidRequest(err))
		}
	}), http.MethodPost, "/test", strings.NewReader(strings.Repeat("x", 64)))

	if rec.Code != http.StatusRequestEntityTooLarge || decodeError(t, rec).Code != codeBodyTooLarge {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}
