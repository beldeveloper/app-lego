package variable

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the interface for variables service.
type Service interface {
	List(ctx context.Context, v model.Variables) (map[string]string, error)
	ListEnv(ctx context.Context, v model.Variables) ([]string, error)
	Replace(ctx context.Context, data []byte, v model.Variables) ([]byte, error)
}
