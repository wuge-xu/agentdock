package httptransport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

type RequestIDGenerator func() (string, error)

func RequestIDMiddleware(
	generator RequestIDGenerator,
) func(http.Handler) http.Handler {
	if generator == nil {
		generator = generateRequestID
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				requestID := strings.TrimSpace(
					request.Header.Get(RequestIDHeader),
				)

				if requestID == "" {
					generatedID, err := generator()
					if err != nil {
						writeError(
							writer,
							http.StatusInternalServerError,
							"request_id_generation_failed",
							"failed to generate request ID",
							"",
						)
						return
					}

					requestID = generatedID
				}

				writer.Header().Set(
					RequestIDHeader,
					requestID,
				)

				contextWithRequestID := context.WithValue(
					request.Context(),
					requestIDContextKey{},
					requestID,
				)

				next.ServeHTTP(
					writer,
					request.WithContext(contextWithRequestID),
				)
			},
		)
	}
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(
		requestIDContextKey{},
	).(string)
	if !ok {
		return ""
	}

	return requestID
}

func generateRequestID() (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}
