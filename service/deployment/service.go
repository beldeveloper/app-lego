package deployment

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the deployment service interface.
type Service interface {
	FindAll(ctx context.Context) ([]model.Deployment, error)
	FindForAutoRebuild(ctx context.Context, b model.Branch) ([]model.Deployment, error)
	FindByID(ctx context.Context, id uint64) (model.Deployment, error)
	Add(ctx context.Context, d model.Deployment) (model.Deployment, error)
	Update(ctx context.Context, d model.Deployment) (model.Deployment, error)
}
