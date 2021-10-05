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

// List returns the list of all available variables and their values.
func (s Variable) List(ctx context.Context, v model.Variables) (map[string]string, error) {
	list := make(map[string]string)
	if v.Repository.ID > 0 {
		list["REPOSITORY_ID"] = strconv.Itoa(int(v.Repository.ID))
		list["REPOSITORY_TYPE"] = v.Repository.Type
		list["REPOSITORY_NAME"] = v.Repository.Name
		list["REPOSITORY_ALIAS"] = v.Repository.Alias
		err := s.addRepositorySecretsToList(ctx, v.Repository, list)
		if err != nil {
			return nil, err
		}
	}
	if v.Branch.ID > 0 {
		list["BRANCH_ID"] = strconv.Itoa(int(v.Branch.ID))
		list["BRANCH_TYPE"] = v.Branch.Type
		list["BRANCH_NAME"] = v.Branch.Name
		list["BRANCH_HASH"] = v.Branch.Hash
	}
	if v.Deployment.ID > 0 {
		list["DEPLOYMENT_ID"] = strconv.Itoa(int(v.Deployment.ID))
	}
	list["CUSTOM_FILES_DIR"] = s.customFilesDir
	return list, nil
}

// ListEnv returns the list of all available variables and their values in the format of environment variables.
func (s Variable) ListEnv(ctx context.Context, v model.Variables) ([]string, error) {
	list, err := s.List(ctx, v)
	if err != nil {
		return nil, err
	}
	listEnv := make([]string, 0, len(list))
	for k, v := range list {
		listEnv = append(listEnv, fmt.Sprintf("%s=%s", k, v))
	}
	return listEnv, nil
}

// Replace puts the variables values to the configuration.
func (s Variable) Replace(ctx context.Context, data []byte, v model.Variables) ([]byte, error) {
	list, err := s.List(ctx, v)
	if err != nil {
		return nil, err
	}
	err = s.addCustomVariablesToList(data, list)
	if err != nil {
		return nil, err
	}
	for k, v := range list {
		data = bytes.ReplaceAll(data, s.castString("{"+k+"}"), s.castString(v))
	}
	return data, nil
}

func (s Variable) castString(v string) []byte {
	return []byte(v)
}

func (s Variable) addRepositorySecretsToList(ctx context.Context, r model.Repository, list map[string]string) error {
	secrets, err := s.repository.LoadSecrets(ctx, r)
	if err != nil {
		return err
	}
	for _, secret := range secrets {
		list[secret.Name] = secret.Value
	}
	return nil
}

func (s Variable) addCustomVariablesToList(data []byte, list map[string]string) error {
	var cfg struct {
		Variables []string `yaml:"variables"`
	}
	err := s.marshaller.Unmarshal(data, &cfg)
	if err != nil {
		return fmt.Errorf("service.variable.addCustomVariablesToList: unmarshal: %w", err)
	}
	for _, v := range cfg.Variables {
		parts := strings.SplitN(v, "=", 2)
		if len(v) < 2 {
			continue
		}
		list[parts[0]] = parts[1]
	}
	return nil
}
