package controller

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the controller interface.
type Service interface {
	AddRepository(context.Context, model.FormAddRepository) (model.Repository, error)
	DownloadRepositoryJob(context.Context)
	SyncRepositoryJob(context.Context)
	BuildBranchJob(context.Context)
}
