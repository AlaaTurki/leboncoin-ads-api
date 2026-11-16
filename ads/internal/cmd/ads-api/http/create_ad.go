package http

import (
	"encoding/json"
	"errors"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"math"
	"net/http"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

type createAdBody struct {
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	PhotoURL string  `json:"photo_url"`
}

type createAdResponse struct {
	ID int64 `json:"id"`
}

// handlerCreateAd serves POST /ads.
func handlerCreateAd(createAd usecase.CreateAdFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var body createAdBody
		if err := decodeBody(r, &body); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		cents, ok := priceToCents(body.Price)
		if !ok {
			http.Error(rw, "price must be a non-negative amount with at most two decimals", http.StatusBadRequest)
			return
		}
		adID, err := createAd(r.Context(), usecase.CreateAdRequest{
			Title: body.Title, PriceCents: cents, PhotoURL: body.PhotoURL,
			OwnerID: middleware.UserIDFromRequest(r),
		})
		if err != nil {
			if errors.Is(err, adsdao.ErrQuotaExceeded) {
				http.Error(rw, err.Error(), http.StatusTooManyRequests)
				return
			}
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(rw).Encode(createAdResponse{ID: int64(adID)})
	}
}

func priceToCents(price float64) (int64, bool) {
	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 || price >= float64(math.MaxInt64)/100 {
		return 0, false
	}
	scaled := price * 100
	rounded := math.Round(scaled)
	if math.Abs(scaled-rounded) > 1e-7 {
		return 0, false
	}
	return int64(rounded), true
}
