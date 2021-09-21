package model

// Cmd is a model of the OS command.
type Cmd struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
	Env  []string `yaml:"env"`
	Dir  string   `yaml:"dir"`
	Log  bool     `yaml:"-"`
}
