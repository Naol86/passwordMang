package httpapi

import (
	"context"
	"time"

	"passwordmanage/server/internal/adapter/httpapi/apigen"
)

const (
	apiVersion   = "v1"
	readyTimeout = 2 * time.Second
)

// Pinger reports whether the database is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

type systemHandlers struct {
	db      Pinger
	version string
}

func (h systemHandlers) GetHealthz(context.Context, apigen.GetHealthzRequestObject) (apigen.GetHealthzResponseObject, error) {
	return apigen.GetHealthz200JSONResponse{Status: apigen.StatusStatusOk}, nil
}

func (h systemHandlers) GetReadyz(ctx context.Context, _ apigen.GetReadyzRequestObject) (apigen.GetReadyzResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		return apigen.GetReadyz503JSONResponse{Status: apigen.StatusStatusUnavailable}, nil
	}
	return apigen.GetReadyz200JSONResponse{Status: apigen.StatusStatusOk}, nil
}

func (h systemHandlers) GetMeta(context.Context, apigen.GetMetaRequestObject) (apigen.GetMetaResponseObject, error) {
	return apigen.GetMeta200JSONResponse{Version: h.version, ApiVersion: apiVersion}, nil
}
