package variable

import (
	"bytes"
	"context"
	"github.com/beldeveloper/app-lego/model"
	"strconv"
)

// NewVariable creates a new instance of the variables service.
func NewVariable() Variable {
	return Variable{}
}

// Variable implements the variables service.
type Variable struct {
}

// Replace puts the variables values to the configuration.
func (s Variable) Replace(ctx context.Context, data []byte, v model.Variables) ([]byte, error) {
	if v.Repository.ID > 0 {
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_ID}"), s.castUint64(v.Repository.ID))
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_TYPE}"), s.castString(v.Repository.Type))
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_NAME}"), s.castString(v.Repository.Name))
		data = bytes.ReplaceAll(data, s.castString("{REPOSITORY_ALIAS}"), s.castString(v.Repository.Alias))
	}
	if v.Branch.ID > 0 {
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_ID}"), s.castUint64(v.Branch.ID))
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_TYPE}"), s.castString(v.Branch.Type))
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_NAME}"), s.castString(v.Branch.Name))
		data = bytes.ReplaceAll(data, s.castString("{BRANCH_HASH}"), s.castString(v.Branch.Hash))
	}
	return data, nil
}

func (s Variable) castString(v string) []byte {
	return []byte(v)
}

func (s Variable) castUint64(v uint64) []byte {
	return []byte(strconv.Itoa(int(v)))
}
