package repository

import (
	"context"

	"github.com/beldeveloper/app-lego/model"
)

// Service defines the repository service interface.
type Service interface {
	FindByID(ctx context.Context, id uint64) (model.Repository, error)
	FindPending(ctx context.Context) (model.Repository, error)
	FindOutdated(ctx context.Context) (model.Repository, error)
	Add(ctx context.Context, r model.Repository) (model.Repository, error)
	Update(ctx context.Context, r model.Repository) (model.Repository, error)
}
