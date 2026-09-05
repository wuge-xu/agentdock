package httptransport

import (
	"log/slog"
	"net/http"
	"time"
)

func NewRouter(
	logger *slog.Logger,
	database DatabasePinger,
	readinessTimeout time.Duration,
	taskHandlers ...*TaskHandler,
) http.Handler {
	mux := http.NewServeMux()

	healthHandler := NewHealthHandler()

	readinessHandler := NewReadinessHandler(
		database,
		readinessTimeout,
	)

	var taskHandler *TaskHandler

	if len(taskHandlers) > 0 {
		taskHandler = taskHandlers[0]
	}

	if taskHandler != nil {
		mux.HandleFunc(
			"POST /api/v1/tasks",
			taskHandler.Create,
		)

		mux.HandleFunc(
			"GET /api/v1/tasks/{task_id}",
			taskHandler.Get,
		)
	}

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
		"GET /health/ready",
		readinessHandler.Ready,
	)

	mux.HandleFunc(
		"/health/ready",
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
