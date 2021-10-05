package model

import "strings"

// BranchCfg is a model that represents branch build configuration.
type BranchCfg struct {
	Steps     map[string]BranchCfgStep `yaml:"steps"`
	Build     []string                 `yaml:"build"`
	Compose   DockerCompose            `yaml:"compose"`
	Variables []string                 `yaml:"variables"`
}

// BranchCfgStep is a model that represents branch configuration step.
type BranchCfgStep struct {
	Name     string `yaml:"name"`
	Dir      string `yaml:"dir"`
	Commands []Cmd  `yaml:"commands"`
}

// Commands return all commands from the configuration ordered properly.
func (cfg BranchCfg) Commands() []Cmd {
	commands := make([]Cmd, 0, len(cfg.Steps)*5) // approx. capacity
	var step BranchCfgStep
	var exists bool
	for _, sName := range cfg.Build {
		step, exists = cfg.Steps[sName]
		if !exists {
			continue
		}
		for i, cmd := range step.Commands {
			if cmd.Dir == "" {
				cmd.Dir = step.Dir
			} else if step.Dir != "" && strings.HasPrefix(cmd.Dir, ".") {
				cmd.Dir = strings.TrimRight(step.Dir, "/") + "/" + cmd.Dir
			}
			cmd.Env = append(cmd.Env, cfg.Variables...)
			step.Commands[i] = cmd
		}
		commands = append(commands, step.Commands...)
	}
	return commands
}
