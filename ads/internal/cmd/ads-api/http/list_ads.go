package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

type listAdsItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Price     float64   `json:"price"`
	PhotoURL  string    `json:"photo_url"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type listAdsResponse struct {
	Ads []listAdsItem `json:"ads"`
}

func adToListAdsItem(a domain.Ad) listAdsItem {
	return listAdsItem{
		ID:        int64(a.ID),
		Title:     a.Title,
		Price:     float64(a.Price) / 100,
		PhotoURL:  a.PhotoURL,
		UserID:    a.UserID.String(),
		CreatedAt: a.CreatedAt.UTC(),
	}
}

func handlerListAds(listAds usecase.ListAdsFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		req := usecase.ListAdsRequest{
			Title: q.Get("title"),
			Limit: 50,
		}
		if raw := q.Get("limit"); raw != "" {
			v, err := strconv.Atoi(raw)
			if err != nil || v < 1 || v > 100 {
				http.Error(rw, "limit must be between 1 and 100", http.StatusBadRequest)
				return
			}
			req.Limit = v
		}
		if raw := q.Get("offset"); raw != "" {
			v, err := strconv.Atoi(raw)
			if err != nil || v < 0 {
				http.Error(rw, "offset must be a non-negative integer", http.StatusBadRequest)
				return
			}
			req.Offset = v
		}

		if raw := q.Get("user_id"); raw != "" {
			ownerID := domain.UserID(raw)
			req.OwnerID = ownerID
		}

		if raw := q.Get("min_price"); raw != "" {
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			cents, ok := priceToCents(v)
			if !ok {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			req.MinPriceCents = cents
		}

		if raw := q.Get("max_price"); raw != "" {
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			cents, ok := priceToCents(v)
			if !ok {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			req.MaxPriceCents = cents
		}

		ads, err := listAds(r.Context(), req)
		if err != nil {
			slog.ErrorContext(r.Context(), "unable to list ads", "error", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		items := make([]listAdsItem, 0, len(ads))
		for _, a := range ads {
			items = append(items, adToListAdsItem(a))
		}

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(listAdsResponse{Ads: items})
	}
}
