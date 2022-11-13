//go:build wireinject
// +build wireinject

package main

import (
	"github.com/beldeveloper/app-lego/internal/app/http"
	"github.com/beldeveloper/app-lego/internal/app/postgres"
	"github.com/beldeveloper/app-lego/internal/app/svc"
	"github.com/google/wire"
)

func initializeContainer() (container, error) {
	wire.Build(
		postgres.NewRepository,
		postgres.NewBranch,
		postgres.NewDeployment,
		svc.NewRepository,
		svc.NewBranch,
		svc.NewDeployment,
		svc.NewGit,
		svc.NewHook,
		http.NewHandler,
		http.NewRouter,
		newContainer,
		newWatcher,
		newPostgresConn,
		reposDir,
		newAccessKey,
		newHookConn,
	)
	return container{}, nil
}
