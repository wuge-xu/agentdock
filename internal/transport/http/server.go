package httptransport

import (
	"net/http"

	"github.com/wuge-xu/agentdock/internal/config"
)

func NewServer(
	httpConfig config.HTTPConfig,
	handler http.Handler,
) *http.Server {
	return &http.Server{
		Addr:              httpConfig.Address,
		Handler:           handler,
		ReadHeaderTimeout: httpConfig.ReadHeaderTimeout,
		ReadTimeout:       httpConfig.ReadTimeout,
		WriteTimeout:      httpConfig.WriteTimeout,
		IdleTimeout:       httpConfig.IdleTimeout,
	}
}
