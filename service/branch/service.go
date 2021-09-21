package branch

import (
	"context"

	"github.com/beldeveloper/app-lego/model"
)

// Service defines the branches service interface.
type Service interface {
	Sync(ctx context.Context, r model.Repository, branches []model.Branch) ([]model.Branch, error)
	FindEnqueued(ctx context.Context) (model.Branch, error)
	UpdateStatus(ctx context.Context, b model.Branch) error
	LoadDockerCompose(ctx context.Context, b model.Branch) (model.DockerCompose, error)
	SaveDockerCompose(ctx context.Context, b model.Branch, data model.DockerCompose) error
}
