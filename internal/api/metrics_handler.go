package api

import (
	"encoding/json"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/metrics"
)

func MetricsHandler(registry *metrics.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(registry.Snapshot())
	}
}
