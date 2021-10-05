package variable

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the interface for variables service.
type Service interface {
	ListFromSources(ctx context.Context, v model.VariablesSources) ([]model.Variable, error)
	ListCustom(ctx context.Context, data []byte) ([]model.Variable, error)
	ListForRepository(ctx context.Context, r model.Repository) ([]model.Variable, error)
	ListForBranch(ctx context.Context, b model.Branch) ([]model.Variable, error)
	ListForDeployment(ctx context.Context, d model.Deployment) ([]model.Variable, error)
	ListStatic(ctx context.Context) ([]model.Variable, error)
	Replace(ctx context.Context, data []byte, variables []model.Variable) ([]byte, error)
}
