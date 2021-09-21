package service

import (
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/builder"
	"github.com/beldeveloper/app-lego/service/deployer"
	"github.com/beldeveloper/app-lego/service/deployment"
	"github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/validation"
	"github.com/beldeveloper/app-lego/service/variable"
	"github.com/beldeveloper/app-lego/service/vcs"
)

// Container keeps all services in one place.
type Container struct {
	Builder    builder.Service
	Repository repository.Service
	Branches   branch.Service
	Deployment deployment.Service
	Deployer   deployer.Service
	VCS        vcs.Service
	OS         os.Service
	Variable   variable.Service
	Validation validation.Service
}
