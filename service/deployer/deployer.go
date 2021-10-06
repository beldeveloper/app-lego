package deployer

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/deployment"
	"github.com/beldeveloper/app-lego/service/marshaller"
	appOs "github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/variable"
	"github.com/beldeveloper/go-errors-context"
	"log"
	"os"
)

// NewDeployer creates a new instance of the deployer service.
func NewDeployer(
	repositories repository.Service,
	branches branch.Service,
	deployments deployment.Service,
	os appOs.Service,
	variables variable.Service,
	dockerMarshaller marshaller.Service,
	workDir model.FilePath,
) Service {
	return Deployer{
		repositories:     repositories,
		branches:         branches,
		deployments:      deployments,
		os:               os,
		variables:        variables,
		dockerMarshaller: dockerMarshaller,
		workDir:          string(workDir),
	}
}

// Deployer implements the deployer service.
type Deployer struct {
	repositories     repository.Service
	branches         branch.Service
	deployments      deployment.Service
	os               appOs.Service
	variables        variable.Service
	dockerMarshaller marshaller.Service
	workDir          string
}

// Run watches for the deployments state, builds, rebuilds, closes them.
func (s Deployer) Run(ctx context.Context) error {
	deployments, err := s.deployments.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: find repositories"})
	}
	branches, err := s.branches.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: find branches"})
	}
	branchesMap := make(map[uint64]model.Branch)
	for _, b := range branches {
		branchesMap[b.ID] = b
	}
	dockerCompose := s.basicComposeCfg()
	var applyChanges bool
	for i, d := range deployments {
		switch d.Status {
		case model.DeploymentStatusReady, model.DeploymentStatusEnqueued:
			err = s.prepare(ctx, &d, branchesMap, &dockerCompose)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "service.deployer.Run: prepare",
					Params: errors.Params{"deployment": d.ID},
				}))
			}
			if deployments[i].Status == d.Status {
				continue
			}
			deployments[i], err = s.deployments.Update(ctx, d)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "service.deployer.Run: update status",
					Params: errors.Params{"deployment": d.ID, "status": d.Status},
				}))
				continue
			}
			if d.Status != model.DeploymentStatusFailed {
				applyChanges = true
			}
		}
	}
	if !applyChanges {
		return nil
	}
	log.Println("Updating docker-compose configuration")
	var success bool
	defer func() {
		var err error
		for _, d := range deployments {
			if d.Status != model.DeploymentStatusBuilding {
				continue
			}
			if success {
				d.Status = model.DeploymentStatusReady
			} else {
				d.Status = model.DeploymentStatusFailed
			}
			d, err = s.deployments.Update(ctx, d)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "service.deployer.Run: update status",
					Params: errors.Params{"deployment": d.ID, "status": d.Status},
				}))
				continue
			}
			log.Printf("Deployment #%d is %s\n", d.ID, d.Status)
		}
	}()
	err = s.updateDockerCompose(dockerCompose)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: updateDockerCompose"})
	}
	_, err = s.os.RunCmd(ctx, model.Cmd{
		Name: "docker-compose",
		Args: []string{"up", "-d", "--remove-orphans"},
		Dir:  s.workDir,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: up services"})
	}
	log.Println("Docker-compose configuration is updated")
	success = true
	return nil
}

// AutoRebuild enqueues all appropriate deployments to be re-built.
func (s Deployer) AutoRebuild(ctx context.Context, b model.Branch) error {
	deployments, err := s.deployments.FindForAutoRebuild(ctx, b)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.deployer.AutoRebuild: find",
			Params: errors.Params{"branch": b.ID},
		})
	}
	for _, d := range deployments {
		d.Status = model.DeploymentStatusEnqueued
		d, err = s.deployments.Update(ctx, d)
		if err != nil {
			log.Println(errors.WrapContext(err, errors.Context{
				Path:   "service.deployer.AutoRebuild: update",
				Params: errors.Params{"deployment": d.ID, "status": d.Status},
			}))
			continue
		}
		log.Printf("Deployment #%d is enqueued for auto-rebuilding\n", d.ID)
	}
	return nil
}

func (s Deployer) prepare(
	ctx context.Context,
	d *model.Deployment,
	bm map[uint64]model.Branch,
	dc *model.DockerCompose,
) error {
	var b model.Branch
	var ok bool
	for i, db := range d.Branches {
		b, ok = bm[db.ID]
		if !ok {
			d.Status = model.DeploymentStatusFailed
			return errors.NewWithContext("deployment points to deleted branch", errors.Context{
				Path:   "service.deployer.prepare",
				Params: errors.Params{"deployment": d.ID, "branch": db.ID},
			})
		}
		bdcData, err := s.branches.LoadComposeData(ctx, b)
		if err != nil {
			d.Status = model.DeploymentStatusFailed
			return errors.WrapContext(err, errors.Context{
				Path:   "service.deployer.prepare: LoadComposeData",
				Params: errors.Params{"deployment": d.ID, "branch": b.ID},
			})
		}
		variables, err := s.variables.ListForDeployment(ctx, *d)
		if err != nil {
			return errors.WrapContext(err, errors.Context{
				Path:   "service.deployer.prepare: list variables",
				Params: errors.Params{"deployment": d.ID},
			})
		}
		bdcData, err = s.variables.Replace(ctx, bdcData, variables)
		if err != nil {
			d.Status = model.DeploymentStatusFailed
			return errors.WrapContext(err, errors.Context{
				Path:   "service.deployer.prepare: replace variables",
				Params: errors.Params{"deployment": d.ID, "branch": b.ID},
			})
		}
		var bdc model.DockerCompose
		err = s.dockerMarshaller.Unmarshal(bdcData, &bdc)
		if err != nil {
			return errors.WrapContext(err, errors.Context{
				Path:   "service.deployer.prepare: unmarshal compose cfg",
				Params: errors.Params{"deployment": d.ID, "branch": b.ID},
			})
		}
		d.Branches[i].Hash = b.Hash
		for bdcServiceName, bdcService := range bdc.Services {
			dc.Services[bdcServiceName] = bdcService
		}
	}
	switch d.Status {
	case model.DeploymentStatusEnqueued:
		d.Status = model.DeploymentStatusBuilding
	case model.DeploymentStatusReady:
	default:
		return errors.NewWithContext("unexpected status", errors.Context{
			Path:   "service.deployer.prepare",
			Params: errors.Params{"deployment": d.ID, "status": d.Status},
		})
	}
	return nil
}

func (s Deployer) updateDockerCompose(dc model.DockerCompose) error {
	dcData, err := s.dockerMarshaller.Marshal(dc)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.updateDockerCompose: marshal"})
	}
	f, err := os.OpenFile(s.workDir+"/docker-compose.yml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.updateDockerCompose: open file"})
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println(errors.WrapContext(err, errors.Context{Path: "service.deployer.updateDockerCompose: close file"}))
		}
	}()
	_, err = f.Write(dcData)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.updateDockerCompose: write"})
	}
	return nil
}

func (s Deployer) basicComposeCfg() model.DockerCompose {
	return model.DockerCompose{
		Version: model.DockerComposeVersion,
		Services: map[string]model.DockerComposeService{
			"traefik": {
				Image: model.TraefikImage,
				Ports: []string{"80:80", "443:443"},
				Volumes: []string{
					"/var/run/docker.sock:/var/run/docker.sock:ro",
					"./traefik.toml:/traefik.toml:ro",
					"./crt:/crt:ro",
				},
				/*Command: []string{
					"--providers.docker=true",
					"--entrypoints.web.address=:80",
				},*/
			},
		},
	}
}
