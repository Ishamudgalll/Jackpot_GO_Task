package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"jackpotTask/internal/cache"
	"jackpotTask/internal/service"
)

type StatsHandler struct {
	service      *service.StatsService
	cache        cache.Store
	validator    *validator.Validate
	maxDateRange time.Duration
}

func NewStatsHandler(s *service.StatsService, c cache.Store, v *validator.Validate, maxDateRange time.Duration) *StatsHandler {
	return &StatsHandler{service: s, cache: c, validator: v, maxDateRange: maxDateRange}
}

func (h *StatsHandler) GrossGamingRevenue(w http.ResponseWriter, r *http.Request) {
	from, to, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	cacheKey := "ggr:" + from.Format(time.RFC3339) + ":" + to.Format(time.RFC3339)
	if payload, ok := h.cache.Get(cacheKey); ok {
		writeCachedJSON(w, payload)
		return
	}

	rows, err := h.service.GetGGR(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := map[string]any{"from": from, "to": to, "rows": rows}
	h.writeAndCache(w, cacheKey, resp)
}

func (h *StatsHandler) DailyWagerVolume(w http.ResponseWriter, r *http.Request) {
	from, to, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	cacheKey := "daily_wager:" + from.Format(time.RFC3339) + ":" + to.Format(time.RFC3339)
	if payload, ok := h.cache.Get(cacheKey); ok {
		writeCachedJSON(w, payload)
		return
	}

	rows, err := h.service.GetDailyWagerVolume(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := map[string]any{"from": from, "to": to, "rows": rows}
	h.writeAndCache(w, cacheKey, resp)
}

func (h *StatsHandler) UserWagerPercentile(w http.ResponseWriter, r *http.Request) {
	from, to, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	userIDStr := r.PathValue("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid user_id"))
		return
	}

	cacheKey := "user_percentile:" + userID.Hex() + ":" + from.Format(time.RFC3339) + ":" + to.Format(time.RFC3339)
	if payload, ok := h.cache.Get(cacheKey); ok {
		writeCachedJSON(w, payload)
		return
	}

	result, err := h.service.GetUserWagerPercentile(r.Context(), userID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrUserNoWagers) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := map[string]any{"from": from, "to": to, "result": result}
	h.writeAndCache(w, cacheKey, resp)
}

func (h *StatsHandler) parseRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	query := r.URL.Query()
	from, to, err := validateDateRange(h.validator, query.Get("from"), query.Get("to"), h.maxDateRange)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

func validateDateRange(v *validator.Validate, fromRaw, toRaw string, maxRange time.Duration) (time.Time, time.Time, error) {
	payload := struct {
		From string `validate:"required"`
		To   string `validate:"required"`
	}{From: fromRaw, To: toRaw}

	if err := v.Struct(payload); err != nil {
		return time.Time{}, time.Time{}, err
	}

	from, err := parseDate(fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from date: %w", err)
	}

	to, err := parseDate(toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to date: %w", err)
	}

	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("from must be earlier than to")
	}

	if to.Sub(from) > maxRange {
		return time.Time{}, time.Time{}, fmt.Errorf("requested date range exceeds max allowed range")
	}

	return from.UTC(), to.UTC(), nil
}

func parseDate(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD")
}

func (h *StatsHandler) writeAndCache(w http.ResponseWriter, key string, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	h.cache.Set(key, encoded)
	writeCachedJSON(w, encoded)
}

func writeCachedJSON(w http.ResponseWriter, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	_ = writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

func WithTimeout(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

