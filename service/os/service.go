package os

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
)

// Service defines the interface of the service that is in charge of interacting with the operating system.
type Service interface {
	RunCmd(ctx context.Context, cmd model.Cmd) (string, error)
}
