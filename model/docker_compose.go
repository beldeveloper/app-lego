package model

// DockerCompose is a model that represents the docker-compose configuration.
type DockerCompose struct {
	Version  string `yaml:"version"`
	Services map[string]struct {
		Image       string   `yaml:"image,omitempty"`
		Restart     string   `yaml:"restart,omitempty"`
		Ports       []string `yaml:"ports,omitempty"`
		Environment []string `yaml:"environment,omitempty"`
	} `yaml:"services"`
}
