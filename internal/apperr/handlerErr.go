package apperr

import (
	"errors"
	"net/http"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
)

type errorResponse struct {
	status  int
	message string
}

var serviceErrorResponses = map[error]errorResponse{
	service.ErrBadMetricValue:    {status: http.StatusBadRequest, message: "bad metric value"},
	service.ErrUnknownMetricType: {status: http.StatusBadRequest, message: "unknown metrics type"},
	service.ErrMetricNotFound:    {status: http.StatusNotFound},
}

func WriteServiceError(w http.ResponseWriter, err error) {
	for target, resp := range serviceErrorResponses {
		if errors.Is(err, target) {
			msg := resp.message
			if msg == "" {
				msg = http.StatusText(resp.status)
			}
			http.Error(w, msg, resp.status)
			return
		}
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
