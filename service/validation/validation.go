package validation

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"strings"
)

// NewValidation creates a new instance of the validation service.
func NewValidation() Validation {
	return Validation{}
}

// Validation implements the validation service.
type Validation struct {
}

// AddRepository validates the input for add repository request.
func (v Validation) AddRepository(ctx context.Context, f model.FormAddRepository) (model.FormAddRepository, error) {
	if f.Type != model.RepositoryTypeGit {
		return f, fmt.Errorf("%w: repository type is invalid; allowed values: %s", model.ErrBadInput, model.RepositoryTypeGit)
	}
	f.Alias = strings.TrimSpace(f.Alias)
	f.Name = strings.TrimSpace(f.Name)
	if f.Alias == "" {
		return f, fmt.Errorf("%w: repository alias must not be empty", model.ErrBadInput)
	}
	if f.Name == "" {
		return f, fmt.Errorf("%w: repository name must not be empty", model.ErrBadInput)
	}
	return f, nil
}
