package variable

import (
	"bytes"
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/marshaller"
	"github.com/beldeveloper/app-lego/service/repository"
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

// Replace puts the variables values to the configuration.
func (s Variable) Replace(ctx context.Context, data []byte, v model.Variables) ([]byte, error) {
	var err error
	if v.Repository.ID > 0 {
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_ID}"), s.castUint64(v.Repository.ID))
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_TYPE}"), s.castString(v.Repository.Type))
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_NAME}"), s.castString(v.Repository.Name))
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_ALIAS}"), s.castString(v.Repository.Alias))
		data, err = s.replaceRepositoryVariables(ctx, v.Repository, data)
		if err != nil {
			return nil, err
		}
	}
	if v.Branch.ID > 0 {
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_ID}"), s.castUint64(v.Branch.ID))
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_TYPE}"), s.castString(v.Branch.Type))
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_NAME}"), s.castString(v.Branch.Name))
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_HASH}"), s.castString(v.Branch.Hash))
	}
	if v.Deployment.ID > 0 {
		data = bytes.ReplaceAll(data, s.castString("{DEPLOYMENT_ID}"), s.castUint64(v.Deployment.ID))
	}
	data, err = s.replaceCustomVariables(data)
	if err != nil {
		return nil, err
	}
	data = bytes.ReplaceAll(data, s.castString("{CUSTOM_FILES_DIR}"), s.castString(s.customFilesDir))
	return data, nil
}

func (s Variable) castString(v string) []byte {
	return []byte(v)
}

func (s Variable) castUint64(v uint64) []byte {
	return []byte(strconv.Itoa(int(v)))
}

func (s Variable) replaceRepositoryVariables(ctx context.Context, r model.Repository, data []byte) ([]byte, error) {
	secrets, err := s.repository.LoadSecrets(ctx, r)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		data = bytes.ReplaceAll(data, s.castString("{"+secret.Name+"}"), s.castString(secret.Value))
	}
	return data, nil
}

func (s Variable) replaceCustomVariables(data []byte) ([]byte, error) {
	var cfg struct {
		Variables []string `yaml:"variables"`
	}
	err := s.marshaller.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("service.variable.replaceCustomVariables: unmarshal: %w", err)
	}
	for _, v := range cfg.Variables {
		parts := strings.SplitN(v, "=", 2)
		if len(v) < 2 {
			continue
		}
		data = bytes.ReplaceAll(data, s.castString("{"+parts[0]+"}"), s.castString(parts[1]))
	}
	return data, nil
}
