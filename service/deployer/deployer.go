package deployer

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/deployment"
	"github.com/beldeveloper/app-lego/service/marshaller"
	appOs "github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/variable"
	"github.com/beldeveloper/go-errors-context"
	"io/ioutil"
	"log"
	"os"
	"strings"
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
		scriptsDir:       string(workDir + "/" + model.ScriptsDir),
		branchesDir:      string(workDir + "/" + model.BranchesDir),
		configDir:        string(workDir + "/" + model.ConfigDir),
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
	scriptsDir       string
	branchesDir      string
	configDir        string
}

// Run watches for the deployments state, builds, rebuilds, closes them.
func (s Deployer) Run(ctx context.Context) error {
	repositories, err := s.repositories.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: find repositories"})
	}
	repositoriesMap := make(map[uint64]model.Repository)
	for _, r := range repositories {
		repositoriesMap[r.ID] = r
	}
	deployments, err := s.deployments.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: find deployments"})
	}
	branches, err := s.branches.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: find branches"})
	}
	branchesMap := make(map[uint64]model.Branch)
	for _, b := range branches {
		branchesMap[b.ID] = b
	}
	dockerCompose, err := s.basicComposeCfg()
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: get basic compose cfg"})
	}
	var applyChanges bool
	preDeployMap := make(map[string][]model.Cmd)
	postDeployMap := make(map[string][]model.Cmd)
	for i, d := range deployments {
		d := d
		switch d.Status {
		case model.DeploymentStatusReady, model.DeploymentStatusEnqueued:
			err = s.prepare(ctx, &d, repositoriesMap, branchesMap, preDeployMap, postDeployMap, &dockerCompose)
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
			d := d
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
	err = s.runDeploymentsCommands(ctx, deployments, preDeployMap)
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
		return errors.WrapContext(err, errors.Context{Path: "service.deployer.Run: up services"})
	}
	err = s.runDeploymentsCommands(ctx, deployments, postDeployMap)
	if err != nil {
		return err
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
		d := d
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
	rm map[uint64]model.Repository,
	bm map[uint64]model.Branch,
	preDeploy map[string][]model.Cmd,
	postDeploy map[string][]model.Cmd,
	dc *model.DockerCompose,
) error {
	var b model.Branch
	var ok bool
	for i, db := range d.Branches {
		db := db
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
		for _, b := range d.Branches {
			b := b
			dBranch := bm[b.ID]
			dRepo := rm[dBranch.RepositoryID]
			variables = append(variables, model.Variable{
				Name:  fmt.Sprintf("%s_BRANCH_TMP_DIR", strings.ToUpper(dRepo.Alias)),
				Value: fmt.Sprintf("%s/%d", s.branchesDir, dBranch.ID),
			})
			variables = append(variables, model.Variable{
				Name:  fmt.Sprintf("%s_BRANCH_ID", strings.ToUpper(dRepo.Alias)),
				Value: fmt.Sprintf("%d", dBranch.ID),
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
		var composeData model.BranchComposeData
		err = s.dockerMarshaller.Unmarshal(bdcData, &composeData)
		if err != nil {
			return errors.WrapContext(err, errors.Context{
				Path:   "service.deployer.prepare: unmarshal compose cfg",
				Params: errors.Params{"deployment": d.ID, "branch": b.ID},
			})
		}
		preDeploy[fmt.Sprintf("%d_%d", d.ID, b.ID)] = composeData.PreDeploy
		postDeploy[fmt.Sprintf("%d_%d", d.ID, b.ID)] = composeData.PostDeploy
		d.Branches[i].Hash = b.Hash
		for bdcServiceName, bdcService := range composeData.ComposeServices {
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

func (s Deployer) basicComposeCfg() (model.DockerCompose, error) {
	var cfg model.DockerCompose
	data, err := ioutil.ReadFile(s.configDir + "/docker-compose.yml")
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path: "service.deployer.basicComposeCfg: read template",
		})
	}
	err = s.dockerMarshaller.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path: "service.deployer.basicComposeCfg: unmarshal template",
		})
	}
	return cfg, nil
}

func (s Deployer) runDeploymentsCommands(ctx context.Context, deployments []model.Deployment, commandsMap map[string][]model.Cmd) error {
	var err error
	for _, d := range deployments {
		d := d
		if d.Status != model.DeploymentStatusBuilding {
			continue
		}
		for _, b := range d.Branches {
			b := b
			for _, cmd := range commandsMap[fmt.Sprintf("%d_%d", d.ID, b.ID)] {
				cmd.Log = true
				if cmd.Dir == "" {
					cmd.Dir = fmt.Sprintf("%s/%d", s.branchesDir, b.ID)
				} else if strings.HasPrefix(cmd.Dir, ".") {
					cmd.Dir = fmt.Sprintf("%s/%d/%s", s.branchesDir, b.ID, cmd.Dir)
				}
				_, err = s.os.RunCmd(ctx, cmd)
				if err != nil {
					return errors.WrapContext(err, errors.Context{
						Path: "service.deployer.runDeploymentsCommands",
						Params: errors.Params{
							"deploymentId": d.ID,
							"branchId":     b.ID,
							"cmd":          cmd.Name,
							"args":         cmd.Args,
							"env":          cmd.Env,
							"dir":          cmd.Dir,
						},
					})
				}
			}
		}
	}
	return nil
}
