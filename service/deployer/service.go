package deployer

import "context"

// Service defines the deployer service interface.
type Service interface {
	Run(ctx context.Context) error
}
