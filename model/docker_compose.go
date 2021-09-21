package model

const (
	// DockerComposeVersion defines the docker-compose version used for deploying the target application.
	DockerComposeVersion = "3"
)

// DockerCompose is a model that represents the docker-compose configuration.
type DockerCompose struct {
	Version  string                          `yaml:"version"`
	Services map[string]DockerComposeService `yaml:"services"`
}

// DockerComposeService is a model that represents the docker-compose service configuration.
type DockerComposeService struct {
	Image       string   `yaml:"image,omitempty"`
	Restart     string   `yaml:"restart,omitempty"`
	Ports       []string `yaml:"ports,omitempty"`
	Environment []string `yaml:"environment,omitempty"`
}
