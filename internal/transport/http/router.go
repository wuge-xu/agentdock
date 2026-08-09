package httptransport

import (
	"log/slog"
	"net/http"
)

func NewRouter(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	healthHandler := NewHealthHandler()

	mux.HandleFunc(
		"GET /health/live",
		healthHandler.Live,
	)

	mux.HandleFunc(
		"/health/live",
		func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			writeError(
				writer,
				http.StatusMethodNotAllowed,
				"method_not_allowed",
				"method not allowed",
				RequestIDFromContext(request.Context()),
			)
		},
	)

	mux.HandleFunc(
		"/",
		func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			writeError(
				writer,
				http.StatusNotFound,
				"route_not_found",
				"route not found",
				RequestIDFromContext(request.Context()),
			)
		},
	)

	handler := AccessLogMiddleware(
		logger,
		nil,
	)(mux)

	return RequestIDMiddleware(nil)(handler)
}
