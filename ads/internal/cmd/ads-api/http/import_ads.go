package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

type importAdsBody struct {
	Ads []createAdBody `json:"ads"`
}

type importAdsResponse struct {
	IDs []int64 `json:"ids"`
}

const MaxImportBatchSize = 500

func handlerImportAds(importAds usecase.ImportAdsFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(rw, r.Body, 2<<20)
		var body importAdsBody
		if err := decodeBody(r, &body); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(rw, "import request exceeds 2 MiB", http.StatusRequestEntityTooLarge)
				return
			}
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		ownerID := domain.UserID(middleware.UserIDFromRequest(r))
		if len(body.Ads) > MaxImportBatchSize {
			http.Error(rw, "import batch exceeds maximum of 500 ads", http.StatusRequestEntityTooLarge)
			return
		}
		requests := make([]usecase.CreateAdRequest, 0, len(body.Ads))
		for _, item := range body.Ads {
			if err := validateAd(item.Title, item.PhotoURL); err != nil {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			cents, ok := priceToCents(item.Price)
			if !ok {
				http.Error(rw, "invalid price", http.StatusBadRequest)
				return
			}
			requests = append(requests, usecase.CreateAdRequest{Title: item.Title, PriceCents: cents, PhotoURL: item.PhotoURL, OwnerID: ownerID})
		}
		key := r.Header.Get("Idempotency-Key")
		if key == "" || len(key) > 200 {
			http.Error(rw, "Idempotency-Key header must contain 1 to 200 characters", http.StatusBadRequest)
			return
		}
		created, err := importAds(r.Context(), key, requests)
		if err != nil {
			if errors.Is(err, adsdao.ErrIdempotencyConflict) {
				http.Error(rw, err.Error(), http.StatusConflict)
				return
			}
			slog.ErrorContext(r.Context(), "unable to import ads", "error", err)
			http.Error(rw, "unable to import ads", http.StatusInternalServerError)
			return
		}
		ids := make([]int64, 0, len(created))
		for _, id := range created {
			ids = append(ids, int64(id))
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(rw).Encode(importAdsResponse{IDs: ids})
	}
}
