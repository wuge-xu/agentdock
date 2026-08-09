package httptransport

import (
	"log/slog"
	"net/http"
	"time"
)

type TimeSource func() time.Time

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (recorder *statusRecorder) WriteHeader(statusCode int) {
	if recorder.statusCode != 0 {
		return
	}

	recorder.statusCode = statusCode
	recorder.ResponseWriter.WriteHeader(statusCode)
}

func (recorder *statusRecorder) Write(data []byte) (int, error) {
	if recorder.statusCode == 0 {
		recorder.WriteHeader(http.StatusOK)
	}

	return recorder.ResponseWriter.Write(data)
}

func (recorder *statusRecorder) StatusCode() int {
	if recorder.statusCode == 0 {
		return http.StatusOK
	}

	return recorder.statusCode
}

func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

func AccessLogMiddleware(
	logger *slog.Logger,
	now TimeSource,
) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	if now == nil {
		now = time.Now
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				startedAt := now()

				recorder := &statusRecorder{
					ResponseWriter: writer,
				}

				next.ServeHTTP(recorder, request)

				duration := now().Sub(startedAt)
				if duration < 0 {
					duration = 0
				}

				logger.Info(
					"http request completed",
					slog.String(
						"request_id",
						RequestIDFromContext(request.Context()),
					),
					slog.String(
						"method",
						request.Method,
					),
					slog.String(
						"path",
						request.URL.Path,
					),
					slog.Int(
						"status_code",
						recorder.StatusCode(),
					),
					slog.Int64(
						"duration_ms",
						duration.Milliseconds(),
					),
				)
			},
		)
	}
}
