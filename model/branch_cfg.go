package model

import (
	"fmt"
	"strings"
)

// BranchCfg is a model that represents branch build configuration.
type BranchCfg struct {
	Steps      map[string]BranchCfgStep `yaml:"steps"`
	Build      []string                 `yaml:"build"`
	PreDeploy  []string                 `yaml:"pre_deploy"`
	PostDeploy []string                 `yaml:"post_deploy"`
	Compose    DockerCompose            `yaml:"compose"`
	Variables  []Variable               `yaml:"-"`
}

// BranchCfgStep is a model that represents branch configuration step.
type BranchCfgStep struct {
	Name     string `yaml:"name"`
	Dir      string `yaml:"dir"`
	Commands []Cmd  `yaml:"commands"`
}

// Commands return all commands from the specific step ordered properly.
func (cfg BranchCfg) Commands(stepList []string) []Cmd {
	commands := make([]Cmd, 0, len(cfg.Steps)*5) // approx. capacity
	var step BranchCfgStep
	var exists bool
	env := make([]string, len(cfg.Variables))
	for i, v := range cfg.Variables {
		env[i] = fmt.Sprintf("%s=%s", v.Name, v.Value)
	}
	for _, sName := range stepList {
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
			cmd.Env = append(cmd.Env, env...)
			step.Commands[i] = cmd
		}
		commands = append(commands, step.Commands...)
	}
	return commands
}
