package repository

import (
	"context"

	"github.com/beldeveloper/app-lego/model"
)

// Service defines the repository service interface.
type Service interface {
	FindAll(ctx context.Context) ([]model.Repository, error)
	FindByID(ctx context.Context, id uint64) (model.Repository, error)
	FindPending(ctx context.Context) (model.Repository, error)
	FindOutdated(ctx context.Context) (model.Repository, error)
	Add(ctx context.Context, r model.Repository) (model.Repository, error)
	Update(ctx context.Context, r model.Repository) (model.Repository, error)
	LoadSecrets(ctx context.Context, r model.Repository) ([]model.Variable, error)
	SaveSecrets(ctx context.Context, r model.Repository, secrets []model.Variable) error
}
