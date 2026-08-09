package httptransport

import (
	"encoding/json"
	"net/http"
)

const contentTypeJSON = "application/json; charset=utf-8"

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func writeJSON(
	writer http.ResponseWriter,
	statusCode int,
	payload any,
) {
	writer.Header().Set("Content-Type", contentTypeJSON)
	writer.WriteHeader(statusCode)

	_ = json.NewEncoder(writer).Encode(payload)
}

func writeError(
	writer http.ResponseWriter,
	statusCode int,
	code string,
	message string,
	requestID string,
) {
	writeJSON(
		writer,
		statusCode,
		errorEnvelope{
			Error: errorBody{
				Code:      code,
				Message:   message,
				RequestID: requestID,
			},
		},
	)
}
