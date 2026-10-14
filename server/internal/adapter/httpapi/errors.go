package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"passwordmanage/server/internal/adapter/httpapi/apigen"
	"passwordmanage/server/internal/domain/errs"
)

const (
	codeInternal      = "internal_error"
	codeInvalidInput  = "invalid_input"
	codeBodyTooLarge  = "body_too_large"
	codeRouteNotFound = "route_not_found"

	messageInternal = "Something went wrong. Quote the request ID if you report it."
)

var statusByKind = map[errs.Kind]int{
	errs.KindInvalid:         http.StatusBadRequest,
	errs.KindUnauthenticated: http.StatusUnauthorized,
	errs.KindForbidden:       http.StatusForbidden,
	errs.KindNotFound:        http.StatusNotFound,
	errs.KindConflict:        http.StatusConflict,
	errs.KindTooLarge:        http.StatusRequestEntityTooLarge,
	errs.KindRateLimited:     http.StatusTooManyRequests,
}

// writeError sends the standard error body. Only domain errors reveal their
// code and message; anything else becomes a generic 500 and is logged.
func writeError(c *gin.Context, log *slog.Logger, err error) {
	requestID := requestIDFrom(c)

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		err = errs.TooLarge(codeBodyTooLarge, "Request body is too large.").Wrap(err)
	}

	domainErr, ok := errs.As(err)
	status, known := http.StatusInternalServerError, false
	if ok {
		status, known = statusByKind[domainErr.Kind]
	}
	if !known {
		log.ErrorContext(c.Request.Context(), "request failed", "request_id", requestID, "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, apigen.Error{Error: apigen.ErrorDetail{
			Code: codeInternal, Message: messageInternal, RequestId: requestID,
		}})
		return
	}

	detail := apigen.ErrorDetail{Code: domainErr.Code, Message: domainErr.Message, RequestId: requestID}
	if len(domainErr.Fields) > 0 {
		fields := make([]apigen.FieldError, len(domainErr.Fields))
		for i, f := range domainErr.Fields {
			fields[i] = apigen.FieldError{Field: f.Field, Message: f.Message}
		}
		detail.Fields = &fields
	}
	c.AbortWithStatusJSON(status, apigen.Error{Error: detail})
}

// invalidRequest wraps decoding and parameter errors from the generated code.
func invalidRequest(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return err
	}
	return errs.Invalid(codeInvalidInput, "The request is not valid.").Wrap(err)
}
