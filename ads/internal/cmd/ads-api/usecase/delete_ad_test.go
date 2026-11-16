package usecase_test

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"testing"
)

func TestDeleteAd(t *testing.T) {
	ctx := context.Background()
	t.Run("deletes only for the requesting owner", func(t *testing.T) {
		m := mock.New(t)
		m.On("DeleteAd", ctx, domain.AdID(9), domain.UserID("owner")).Return(nil)
		require.NoError(t, usecase.NewDeleteAdFunc(m)(ctx, 9, "owner"))
	})
	t.Run("preserves not found", func(t *testing.T) {
		m := mock.New(t)
		m.On("DeleteAd", ctx, domain.AdID(9), domain.UserID("owner")).Return(adsdao.ErrNotFound)
		assert.ErrorIs(t, usecase.NewDeleteAdFunc(m)(ctx, 9, "owner"), adsdao.ErrNotFound)
	})
	t.Run("wraps store error", func(t *testing.T) {
		m := mock.New(t)
		boom := errors.New("db down")
		m.On("DeleteAd", ctx, domain.AdID(9), domain.UserID("owner")).Return(boom)
		assert.ErrorIs(t, usecase.NewDeleteAdFunc(m)(ctx, 9, "owner"), boom)
	})
}
