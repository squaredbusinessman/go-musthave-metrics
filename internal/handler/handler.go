package handler

import (
	"net/http"
	"strconv"
	"strings"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

func AcceptMetricsToStorage(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != `POST` {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != `text/plain` {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		}

		parts := strings.Split(strings.Trim(r.URL.Path, `/`), `/`)
		if len(parts) != 4 || parts[0] != `update` {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		metricType, name, raw := parts[1], parts[2], parts[3]
		switch metricType {
		case `gauge`:
			val, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				http.Error(w, "bad gauge value", http.StatusBadRequest)
				return
			}
			storage.SetGauge(name, models.Gauge{Value: val})
		case `counter`:
			val, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				http.Error(w, "bad counter value", http.StatusBadRequest)
				return
			}
			storage.SetCounter(name, models.Counter{Value: val})
		default:
			http.Error(w, "unknown metrics type", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
