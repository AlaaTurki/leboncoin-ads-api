package usecase

import (
	"context"
	"fmt"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	adsdomain "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

type DeleteAdFunc func(ctx context.Context, id adsdomain.AdID, ownerID adsdomain.UserID) error

func NewDeleteAdFunc(adDAO adsdao.DAO) DeleteAdFunc {
	return func(ctx context.Context, id adsdomain.AdID, ownerID adsdomain.UserID) error {
		if err := adDAO.DeleteAd(ctx, id, ownerID); err != nil {
			return fmt.Errorf("unable to delete ad %s: %w", id, err)
		}
		return nil
	}
}
