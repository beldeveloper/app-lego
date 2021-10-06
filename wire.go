//+build wireinject

package main

import (
	"github.com/beldeveloper/app-lego/controller"
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/builder"
	"github.com/beldeveloper/app-lego/service/deployer"
	"github.com/beldeveloper/app-lego/service/deployment"
	"github.com/beldeveloper/app-lego/service/marshaller"
	"github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/validation"
	"github.com/beldeveloper/app-lego/service/variable"
	"github.com/beldeveloper/app-lego/service/vcs"
	"github.com/google/wire"
)

func InitializeController() (controller.Service, error) {
	wire.Build(
		repository.NewPostgres,
		branch.NewPostgres,
		deployment.NewPostgres,
		os.NewOS,
		variable.NewVariable,
		marshaller.NewYaml,
		validation.NewValidation,
		vcs.NewGit,
		builder.NewBuilder,
		deployer.NewDeployer,
		controller.NewController,
		postgresConn,
		postgresSchema,
		workDir,
	)
	return controller.Controller{}, nil
}
