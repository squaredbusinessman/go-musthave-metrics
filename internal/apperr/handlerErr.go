package apperr

import (
	"errors"
	"net/http"
)

type errorResponse struct {
	status  int
	message string
}

var serviceErrorResponses = map[error]errorResponse{
	ErrBadMetricValue:    {status: http.StatusBadRequest, message: "bad metric value"},
	ErrUnknownMetricType: {status: http.StatusBadRequest, message: "unknown metrics type"},
	ErrMetricNotFound:    {status: http.StatusNotFound},
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
