package variable

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the interface for variables service.
type Service interface {
	Replace(ctx context.Context, data []byte, v model.Variables) ([]byte, error)
}
