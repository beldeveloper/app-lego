package model

// Cmd is a model of the OS command.
type Cmd struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
	Dir  string   `yaml:"dir"`
	Log  bool     `yaml:"-"`
}
