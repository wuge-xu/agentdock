package httptransport

import "net/http"

type healthResponse struct {
	Status string `json:"status"`
}

type HealthHandler struct{}

func NewHealthHandler() HealthHandler {
	return HealthHandler{}
}

func (HealthHandler) Live(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writeJSON(
		writer,
		http.StatusOK,
		healthResponse{
			Status: "alive",
		},
	)
}
