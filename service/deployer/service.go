package deployer

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the deployer service interface.
type Service interface {
	Run(ctx context.Context) error
	AutoRebuild(ctx context.Context, b model.Branch) error
}
