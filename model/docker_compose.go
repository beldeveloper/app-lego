package model

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
	Command     []string `yaml:"command,omitempty"`
	Volumes     []string `yaml:"volumes,omitempty"`
	Labels      []string `yaml:"labels,omitempty"`
}

// DockerComposeNetwork is a model that represents the docker-compose network configuration.
type DockerComposeNetwork struct {
	Name     string `yaml:"name"`
	Driver   string `yaml:"driver"`
	External bool   `yaml:"external"`
}
