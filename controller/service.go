package controller

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the controller interface.
type Service interface {
	Repositories(context.Context) ([]model.Repository, error)
	AddRepository(context.Context, model.FormAddRepository) (model.Repository, error)
	Branches(context.Context) ([]model.Branch, error)
	Deployments(context.Context) ([]model.Deployment, error)
	AddDeployment(context.Context, model.FormAddDeployment) (model.Deployment, error)
	RebuildDeployment(context.Context, uint64) (model.Deployment, error)
	CloseDeployment(context.Context, uint64) error
	DownloadRepositoryJob(context.Context)
	SyncRepositoryJob(context.Context)
	BuildBranchJob(context.Context)
	WatchDeploymentsJob(context.Context)
}
