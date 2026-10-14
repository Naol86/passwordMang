// Package httpapi is the HTTP adapter: Gin routing, middleware, and mapping
// between HTTP and use cases. It holds no business rules.
package httpapi

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"passwordmanage/server/internal/adapter/httpapi/apigen"
	"passwordmanage/server/internal/domain/errs"
)

type Deps struct {
	Log            *slog.Logger
	DB             Pinger
	Version        string
	TrustedProxies []string
	Metrics        prometheus.Registerer
}

// strictServer combines every feature's handlers into the generated interface.
type strictServer struct {
	systemHandlers
}

func NewRouter(d Deps) (*gin.Engine, error) {
	// Debug mode only prints a banner and the route table; our own middleware logs requests.
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(
		requestID(),
		recovery(d.Log),
		newMetrics(d.Metrics).middleware(),
		requestLogger(d.Log),
		securityHeaders(),
		bodyLimit(maxBodyBytes),
	)
	r.NoRoute(func(c *gin.Context) {
		writeError(c, d.Log, errs.NotFound(codeRouteNotFound, "No such endpoint."))
	})

	server := strictServer{systemHandlers: systemHandlers{db: d.DB, version: d.Version}}
	handler := apigen.NewStrictHandlerWithOptions(server, nil, apigen.StrictGinServerOptions{
		RequestErrorHandlerFunc:  func(c *gin.Context, err error) { writeError(c, d.Log, invalidRequest(err)) },
		HandlerErrorFunc:         func(c *gin.Context, err error) { writeError(c, d.Log, err) },
		ResponseErrorHandlerFunc: func(c *gin.Context, err error) { writeError(c, d.Log, err) },
	})
	apigen.RegisterHandlersWithOptions(r, handler, apigen.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, _ int) { writeError(c, d.Log, invalidRequest(err)) },
	})
	return r, nil
}
