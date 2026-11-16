package usecase

import (
	"context"
	"fmt"
	"time"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

type (
	// CreateAdFunc creates an ad owned by the calling user.
	CreateAdFunc func(ctx context.Context, req CreateAdRequest) (domain.AdID, error)

	// CreateAdRequest describes the ad to create. OwnerID is the authenticated user: a caller can never create an ad on behalf of someone else.
	CreateAdRequest struct {
		Title      string
		PriceCents int64
		PhotoURL   string
		OwnerID    domain.UserID
	}

	createAd struct {
		adDAO adsdao.DAO
	}
)

// NewCreateAdFunc returns a CreateAdFunc backed by the given DAO.
func NewCreateAdFunc(adDAO adsdao.DAO) CreateAdFunc {
	return createAd{adDAO: adDAO}.createAd
}

func (uc createAd) createAd(ctx context.Context, req CreateAdRequest) (domain.AdID, error) {
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("unable to begin ad creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := tx.LockOwnerForQuota(ctx, req.OwnerID); err != nil {
		return 0, fmt.Errorf("unable to check ad quota: %w", err)
	}
	count, err := tx.CountAdsSince(ctx, req.OwnerID, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		return 0, fmt.Errorf("unable to check ad quota: %w", err)
	}
	if count >= 5 {
		return 0, adsdao.ErrQuotaExceeded
	}

	id, err := tx.CreateAd(ctx, toDAORequest(req))
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}
	if err := tx.RecordAdCreation(ctx, req.OwnerID); err != nil {
		return 0, fmt.Errorf("unable to record ad publication: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("unable to commit ad creation: %w", err)
	}
	return id, nil
}

// ImportAdsFunc atomically creates or replays a batch using the supplied idempotency key.
type ImportAdsFunc func(ctx context.Context, idempotencyKey string, requests []CreateAdRequest) ([]domain.AdID, error)

// NewImportAdsFunc returns an idempotent bulk importer backed by the given DAO.
func NewImportAdsFunc(adDAO adsdao.DAO) ImportAdsFunc {
	return func(ctx context.Context, key string, requests []CreateAdRequest) ([]domain.AdID, error) {
		daoRequests := make([]adsdao.CreateAdRequest, 0, len(requests))
		for _, req := range requests {
			daoRequests = append(daoRequests, toDAORequest(req))
		}
		ids, err := adDAO.ImportAds(ctx, key, daoRequests)
		if err != nil {
			return nil, fmt.Errorf("unable to import ads: %w", err)
		}
		return ids, nil
	}
}

func toDAORequest(req CreateAdRequest) adsdao.CreateAdRequest {
	return adsdao.CreateAdRequest{Title: req.Title, PriceCents: req.PriceCents, PhotoURL: req.PhotoURL, OwnerID: req.OwnerID}
}
