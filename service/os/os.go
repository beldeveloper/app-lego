package os

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
	"log"
	"os/exec"
)

// NewOS creates a new instance of the OS module.
func NewOS() OS {
	return OS{}
}

// OS implements a module that interacts with the operating system.
type OS struct {
}

// RunCmd executes the system command and returns the system output.
func (os OS) RunCmd(ctx context.Context, cmd model.Cmd) (string, error) {
	osCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	osCmd.Dir = cmd.Dir
	log.Printf("Exec OS command: %s %v; dir %s\n", cmd.Name, cmd.Args, cmd.Dir)
	res, err := osCmd.Output()
	return string(res), err
}
