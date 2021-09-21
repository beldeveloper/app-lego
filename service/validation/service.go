package validation

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the interface of the validation service.
type Service interface {
	AddRepository(ctx context.Context, f model.FormAddRepository) (model.FormAddRepository, error)
}
