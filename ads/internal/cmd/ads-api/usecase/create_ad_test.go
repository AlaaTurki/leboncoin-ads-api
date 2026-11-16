package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	testmock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestCreateAd(t *testing.T) {
	ctx := context.Background()

	ucReq := usecase.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    "user-1",
	}
	daoReq := adsdao.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    "user-1",
	}

	t.Run("nominal", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwnerForQuota", ctx, ucReq.OwnerID).Return(nil)
		m.On("CountAdsSince", ctx, ucReq.OwnerID, testmock.Anything).Return(0, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(7), nil)
		m.On("RecordAdCreation", ctx, ucReq.OwnerID).Return(nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		require.NoError(t, err)
		assert.Equal(t, domain.AdID(7), got)
	})

	t.Run("rejects a sixth ad in the rolling window", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwnerForQuota", ctx, ucReq.OwnerID).Return(nil)
		m.On("CountAdsSince", ctx, ucReq.OwnerID, testmock.Anything).Return(5, nil)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		assert.ErrorIs(t, err, adsdao.ErrQuotaExceeded)
	})

	t.Run("error from dao", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwnerForQuota", ctx, ucReq.OwnerID).Return(nil)
		m.On("CountAdsSince", ctx, ucReq.OwnerID, testmock.Anything).Return(0, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(0), dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		require.Error(t, err)
		assert.ErrorIs(t, err, dbErr)
	})
}
