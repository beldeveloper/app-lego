package vcs

import (
	"context"

	"github.com/beldeveloper/app-lego/model"
)

// Service defines the VCS service interface.
type Service interface {
	DownloadRepository(ctx context.Context, r model.Repository) error
	Branches(ctx context.Context, r model.Repository) ([]model.Branch, error)
	SwitchBranch(ctx context.Context, r model.Repository, b model.Branch) error
	ReadConfiguration(ctx context.Context, r model.Repository, b model.Branch) (model.BranchCfg, error)
}
