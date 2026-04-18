package httpapi

import (
	"net/http"
	"time"

	"jackpotTask/internal/http/handlers"
	"jackpotTask/internal/http/middleware"
)

func NewRouter(stats *handlers.StatsHandler, authToken string, timeout time.Duration) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /gross_gaming_rev", stats.GrossGamingRevenue)
	mux.HandleFunc("GET /daily_wager_volume", stats.DailyWagerVolume)
	mux.HandleFunc("GET /user/{user_id}/wager_percentile", stats.UserWagerPercentile)

	handler := middleware.Auth(authToken)(mux)
	return handlers.WithTimeout(handler, timeout)
}

