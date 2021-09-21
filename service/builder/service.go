package builder

import (
	"context"

	"github.com/beldeveloper/app-lego/model"
)

// Service defines the builder interface.
type Service interface {
	Enqueue(context.Context, model.Branch) error
	Build(context.Context, model.Branch) error
}
