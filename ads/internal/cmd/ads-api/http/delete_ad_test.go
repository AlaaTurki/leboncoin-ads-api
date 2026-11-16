package http

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerDeleteAd(t *testing.T) {
	var gotID domain.AdID
	var gotOwner domain.UserID
	mux := http.NewServeMux()
	mux.Handle("DELETE /ads/{adID}", middleware.RequireAuth(handlerDeleteAd(func(_ context.Context, id domain.AdID, owner domain.UserID) error {
		gotID = id
		gotOwner = owner
		return nil
	})))
	server := httptest.NewServer(mux)
	defer server.Close()
	req, err := http.NewRequest(http.MethodDelete, server.URL+"/ads/12", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "user-7")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, domain.AdID(12), gotID)
	assert.Equal(t, domain.UserID("user-7"), gotOwner)
}

func TestHandlerDeleteAdRequiresAuthentication(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("DELETE /ads/{adID}", middleware.RequireAuth(handlerDeleteAd(func(context.Context, domain.AdID, domain.UserID) error {
		t.Fatal("delete called without auth")
		return nil
	})))
	req := httptest.NewRequest(http.MethodDelete, "/ads/12", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

var _ usecase.DeleteAdFunc = func(context.Context, domain.AdID, domain.UserID) error { return nil }
