package http

import (
	"errors"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
	"net/http"
)

func handlerDeleteAd(deleteAd usecase.DeleteAdFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		id, err := adIDFromURL(r)
		if err != nil {
			http.Error(rw, "invalid ad id", http.StatusBadRequest)
			return
		}
		err = deleteAd(r.Context(), id, middleware.UserIDFromRequest(r))
		if errors.Is(err, adsdao.ErrNotFound) {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(rw, "unable to delete ad", http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusNoContent)
	}
}
