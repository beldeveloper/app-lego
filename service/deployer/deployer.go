package deployer

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/deployment"
	appOs "github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/variable"
	"gopkg.in/yaml.v2"
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
	workDir string,
) Deployer {
	return Deployer{
		repositories: repositories,
		branches:     branches,
		deployments:  deployments,
		os:           os,
		variables:    variables,
		workDir:      workDir,
	}
}

// Deployer implements the deployer service.
type Deployer struct {
	repositories repository.Service
	branches     branch.Service
	deployments  deployment.Service
	os           appOs.Service
	variables    variable.Service
	workDir      string
}

// Run watches for the deployments state, builds, rebuilds, closes them.
func (s Deployer) Run(ctx context.Context) error {
	repositories, err := s.repositories.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("service.deployer.Run: find repositories: %w", err)
	}
	repositoriesMap := make(map[uint64]model.Repository)
	for _, r := range repositories {
		repositoriesMap[r.ID] = r
	}
	deployments, err := s.deployments.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("service.deployer.Run: find deployments: %w", err)
	}
	branches, err := s.branches.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("service.deployer.Run: find branches: %w", err)
	}
	branchesMap := make(map[uint64]model.Branch)
	for _, b := range branches {
		branchesMap[b.ID] = b
	}
	dockerCompose := model.DockerCompose{
		Version:  model.DockerComposeVersion,
		Services: make(map[string]model.DockerComposeService),
	}
	var applyChanges bool
	for i, d := range deployments {
		switch d.Status {
		case model.DeploymentStatusReady, model.DeploymentStatusEnqueued, model.DeploymentStatusPendingRebuild:
			err = s.prepare(ctx, &d, repositoriesMap, branchesMap, &dockerCompose)
			if err != nil {
				log.Println(err)
			}
			if deployments[i].Status == d.Status {
				continue
			}
			deployments[i], err = s.deployments.Update(ctx, d)
			if err != nil {
				log.Printf("service.deployer.Run: update deployment #%d status to %s: %v\n", d.ID, d.Status, err)
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
			switch d.Status {
			case model.DeploymentStatusBuilding, model.DeploymentStatusRebuilding:
				if success {
					d.Status = model.DeploymentStatusReady
				} else {
					d.Status = model.DeploymentStatusFailed
				}
			case model.DeploymentStatusPendingClose:
				if success {
					d.Status = model.DeploymentStatusClosed
				} else {
					continue
				}
			default:
				continue
			}
			d, err = s.deployments.Update(ctx, d)
			if err != nil {
				log.Printf("service.deployer.Run: update deployment #%d status to %s: %v\n", d.ID, d.Status, err)
				continue
			}
			log.Printf("Deployment #%d is marked as %s\n", d.ID, d.Status)
		}
	}()
	err = s.updateDockerCompose(dockerCompose)
	if err != nil {
		return err
	}
	_, err = s.os.RunCmd(ctx, model.Cmd{
		Name: "docker-compose",
		Args: []string{"up", "-d", "--remove-orphans"},
		Dir:  s.workDir,
		Log:  true,
	})
	if err != nil {
		return fmt.Errorf("service.deployer.Run: up services: %w", err)
	}
	log.Println("Docker-compose configuration is updated")
	success = true
	return nil
}

func (s Deployer) prepare(
	ctx context.Context,
	d *model.Deployment,
	rm map[uint64]model.Repository,
	bm map[uint64]model.Branch,
	dc *model.DockerCompose,
) error {
	var b model.Branch
	var r model.Repository
	var ok bool
	for i, db := range d.Branches {
		b, ok = bm[db.ID]
		if !ok {
			d.Status = model.DeploymentStatusFailed
			return fmt.Errorf("service.deployer.prepare: deployment #%d points to deleted branch #%d", d.ID, db.ID)
		}
		r, ok = rm[b.RepositoryID]
		if !ok {
			d.Status = model.DeploymentStatusFailed
			return fmt.Errorf("service.deployer.prepare: deployment #%d points to deleted repository #%d", d.ID, b.RepositoryID)
		}
		bdcData, err := s.branches.LoadComposeData(ctx, b)
		if err != nil {
			d.Status = model.DeploymentStatusFailed
			return fmt.Errorf("service.deployer.prepare: deployment #%d: load docker compose cfg for branch #%d", d.ID, b.ID)
		}
		bdcData, err = s.variables.Replace(ctx, bdcData, model.Variables{
			Repository: r,
			Branch:     b,
			Deployment: *d,
		})
		if err != nil {
			d.Status = model.DeploymentStatusFailed
			return fmt.Errorf("service.deployer.prepare: deployment #%d: put variables to docker compose cfg for branch #%d", d.ID, b.ID)
		}
		var bdc model.DockerCompose
		err = yaml.Unmarshal(bdcData, &bdc)
		if err != nil {
			return fmt.Errorf("service.deployer.prepare: deployment #%d: unmarshal compose cfg for branch #%d: %w", d.ID, b.ID, err)
		}
		d.Branches[i].Hash = b.Hash
		for bdcServiceName, bdcService := range bdc.Services {
			dc.Services[bdcServiceName] = bdcService
		}
	}
	switch d.Status {
	case model.DeploymentStatusEnqueued:
		d.Status = model.DeploymentStatusBuilding
	case model.DeploymentStatusPendingRebuild:
		d.Status = model.DeploymentStatusRebuilding
	case model.DeploymentStatusReady:
	default:
		return fmt.Errorf("service.deployer.prepare: deployment #%d: unexpected status: %s", d.ID, d.Status)
	}
	return nil
}

func (s Deployer) updateDockerCompose(dc model.DockerCompose) error {
	dcData, err := yaml.Marshal(dc)
	if err != nil {
		return fmt.Errorf("service.deployer.updateDockerCompose: marshal: %w", err)
	}
	f, err := os.OpenFile(s.workDir+"/docker-compose.yml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("service.deployer.updateDockerCompose: open file: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Printf("service.deployer.updateDockerCompose: close file: %v\n", err)
		}
	}()
	_, err = f.Write(dcData)
	if err != nil {
		return fmt.Errorf("service.deployer.updateDockerCompose: write: %w", err)
	}
	return nil
}
