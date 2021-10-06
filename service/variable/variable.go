package variable

import (
	"bytes"
	"context"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/marshaller"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/go-errors-context"
	"strconv"
	"strings"
)

// NewVariable creates a new instance of the variables service.
func NewVariable(marshaller marshaller.Service, repository repository.Service, customFilesDir string) Variable {
	return Variable{
		marshaller:     marshaller,
		repository:     repository,
		customFilesDir: customFilesDir,
	}
}

// Variable implements the variables service.
type Variable struct {
	marshaller     marshaller.Service
	repository     repository.Service
	customFilesDir string
}

// ListFromSources returns the list of all available variables and their values according to the specific sources.
func (s Variable) ListFromSources(ctx context.Context, v model.VariablesSources) (list []model.Variable, err error) {
	sources := []func() ([]model.Variable, error){
		func() ([]model.Variable, error) {
			return s.ListStatic(ctx)
		},
	}
	if v.Repository.ID > 0 {
		sources = append(sources, func() ([]model.Variable, error) {
			return s.ListForRepository(ctx, v.Repository)
		})
	}
	if v.Branch.ID > 0 {
		sources = append(sources, func() ([]model.Variable, error) {
			return s.ListForBranch(ctx, v.Branch)
		})
	}
	if v.Deployment.ID > 0 {
		sources = append(sources, func() ([]model.Variable, error) {
			return s.ListForDeployment(ctx, v.Deployment)
		})
	}
	if v.CustomData != nil {
		sources = append(sources, func() ([]model.Variable, error) {
			return s.ListCustom(ctx, v.CustomData)
		})
	}
	var sourceList []model.Variable
	for _, source := range sources {
		sourceList, err = source()
		if err != nil {
			return
		}
		list = append(list, sourceList...)
	}
	return
}

// ListCustom parses the configuration data and returns the custom variables.
func (s Variable) ListCustom(ctx context.Context, data []byte) ([]model.Variable, error) {
	var cfg struct {
		Variables []string `yaml:"variables" json:"variables" xml:"variables" bson:"variables"`
	}
	err := s.marshaller.Unmarshal(data, &cfg)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{Path: "service.variable.ListCustom: unmarshal"})
	}
	list := make([]model.Variable, len(cfg.Variables))
	for i, v := range cfg.Variables {
		parts := strings.SplitN(v, "=", 2)
		if len(v) < 2 {
			continue
		}
		list[i] = model.Variable{
			Type:  model.VariableTypeCustom,
			Name:  parts[0],
			Value: parts[1],
		}
	}
	return list, nil
}

func (s Variable) ListForRepository(ctx context.Context, r model.Repository) ([]model.Variable, error) {
	secrets, err := s.repository.LoadSecrets(ctx, r)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.variable.ListForRepository: LoadSecrets",
			Params: errors.Params{"repository": r.ID},
		})
	}
	variables := []model.Variable{
		{
			Type:  model.VariableTypeBuilding,
			Name:  "REPOSITORY_ID",
			Value: strconv.Itoa(int(r.ID)),
		},
		{
			Type:  model.VariableTypeBuilding,
			Name:  "REPOSITORY_TYPE",
			Value: r.Type,
		},
		{
			Type:  model.VariableTypeBuilding,
			Name:  "REPOSITORY_NAME",
			Value: r.Name,
		},
		{
			Type:  model.VariableTypeBuilding,
			Name:  "REPOSITORY_ALIAS",
			Value: r.Alias,
		},
	}
	return append(variables, secrets...), nil
}

func (s Variable) ListForBranch(ctx context.Context, b model.Branch) ([]model.Variable, error) {
	return []model.Variable{
		{
			Type:  model.VariableTypeBuilding,
			Name:  "BRANCH_ID",
			Value: strconv.Itoa(int(b.ID)),
		},
		{
			Type:  model.VariableTypeBuilding,
			Name:  "BRANCH_TYPE",
			Value: b.Type,
		},
		{
			Type:  model.VariableTypeBuilding,
			Name:  "BRANCH_NAME",
			Value: b.Name,
		},
		{
			Type:  model.VariableTypeBuilding,
			Name:  "BRANCH_HASH",
			Value: b.Hash,
		},
	}, nil
}

func (s Variable) ListForDeployment(ctx context.Context, d model.Deployment) ([]model.Variable, error) {
	return []model.Variable{
		{
			Type:  model.VariableTypeBuilding,
			Name:  "DEPLOYMENT_ID",
			Value: strconv.Itoa(int(d.ID)),
		},
	}, nil
}

func (s Variable) ListStatic(ctx context.Context) ([]model.Variable, error) {
	return []model.Variable{
		{
			Type:  model.VariableTypeBuilding,
			Name:  "CUSTOM_FILES_DIR",
			Value: s.customFilesDir,
		},
	}, nil
}

// Replace puts the variables values to the configuration.
func (s Variable) Replace(ctx context.Context, data []byte, variables []model.Variable) ([]byte, error) {
	for _, v := range variables {
		data = bytes.ReplaceAll(data, []byte("{"+v.Name+"}"), []byte(v.Value))
	}
	return data, nil
}
