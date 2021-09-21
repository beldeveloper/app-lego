package branch

import (
	"context"

	"github.com/beldeveloper/app-lego/model"
)

// Service defines the branches service interface.
type Service interface {
	Sync(ctx context.Context, r model.Repository, branches []model.Branch) ([]model.Branch, error)
	FindAll(ctx context.Context) ([]model.Branch, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]model.Branch, error)
	FindByID(ctx context.Context, id uint64) (model.Branch, error)
	FindEnqueued(ctx context.Context) (model.Branch, error)
	UpdateStatus(ctx context.Context, b model.Branch) error
	LoadComposeData(ctx context.Context, b model.Branch) ([]byte, error)
	SaveComposeData(ctx context.Context, b model.Branch, data []byte) error
}
