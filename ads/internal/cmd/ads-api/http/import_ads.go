package http

import (
	"encoding/json"
	"errors"
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

func handlerImportAds(createAd usecase.CreateAdFunc, atomicImport ...usecase.ImportAdsFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var body importAdsBody
		if err := decodeBody(r, &body); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		ownerID := domain.UserID(middleware.UserIDFromRequest(r))
		requests := make([]usecase.CreateAdRequest, 0, len(body.Ads))
		for _, item := range body.Ads {
			cents, ok := priceToCents(item.Price)
			if !ok {
				http.Error(rw, "invalid price", http.StatusBadRequest)
				return
			}
			requests = append(requests, usecase.CreateAdRequest{Title: item.Title, PriceCents: cents, PhotoURL: item.PhotoURL, OwnerID: ownerID})
		}
		if len(atomicImport) > 0 && atomicImport[0] != nil {
			key := r.Header.Get("Idempotency-Key")
			if key == "" || len(key) > 200 {
				http.Error(rw, "Idempotency-Key header must contain 1 to 200 characters", http.StatusBadRequest)
				return
			}
			created, err := atomicImport[0](r.Context(), key, requests)
			if err != nil {
				if errors.Is(err, adsdao.ErrIdempotencyConflict) {
					http.Error(rw, err.Error(), http.StatusConflict)
					return
				}
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
			return
		}
		ids := make([]int64, 0, len(body.Ads))

		for _, req := range requests {
			adID, err := createAd(r.Context(), req)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			ids = append(ids, int64(adID))
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(rw).Encode(importAdsResponse{IDs: ids})
	}
}
