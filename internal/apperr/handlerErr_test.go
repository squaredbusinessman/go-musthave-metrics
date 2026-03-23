package apperr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteServiceError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "bad metric value",
			err:        ErrBadMetricValue,
			wantStatus: http.StatusBadRequest,
			wantBody:   "bad metric value\n",
		},
		{
			name:       "metric not found",
			err:        ErrMetricNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   "Not Found\n",
		},
		{
			name:       "wrapped unknown metric type",
			err:        errors.Join(errors.New("wrapped"), ErrUnknownMetricType),
			wantStatus: http.StatusBadRequest,
			wantBody:   "unknown metrics type\n",
		},
		{
			name:       "unexpected error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			WriteServiceError(recorder, tt.err)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if recorder.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), tt.wantBody)
			}
		})
	}
}
