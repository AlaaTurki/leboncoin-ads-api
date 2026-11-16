package http

import (
	"encoding/json"
	"errors"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersapi "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
	"net/http"
	"time"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

type getAdResponse struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Price     float64   `json:"price"`
	Username  string    `json:"username"`
	PhotoURL  string    `json:"photo_url"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func adToGetAdResponse(a domain.Ad) getAdResponse {
	return getAdResponse{
		ID:        int64(a.ID),
		Title:     a.Title,
		Price:     float64(a.Price) / 100,
		PhotoURL:  a.PhotoURL,
		UserID:    a.UserID.String(),
		CreatedAt: a.CreatedAt.UTC(),
	}
}

func handlerGetAd(getAd usecase.GetAdByIDFunc, userClients ...usersclient.UserClient) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		adID, err := adIDFromURL(r)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)

			return
		}

		ad, err := getAd(r.Context(), adID)
		if err != nil {
			if errors.Is(err, adsdao.ErrNotFound) {
				rw.WriteHeader(http.StatusNotFound)
			} else {
				rw.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		response := adToGetAdResponse(ad)
		if len(userClients) > 0 && userClients[0] != nil {
			user, err := userClients[0].GetUserByID(r.Context(), ad.UserID)
			if err != nil && !errors.Is(err, usersapi.ErrNotFound) {
				http.Error(rw, "unable to load ad owner", http.StatusBadGateway)
				return
			}
			if err == nil {
				response.Username = user.Nickname
			}
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(response)
	}
}
