package os

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Cmd is a model of the OS command.
type Cmd struct {
	Name string
	Args []string
	Env  []string
	Dir  string
	Log  bool
}

// Exec a system command and get the system output.
func Exec(ctx context.Context, cmd Cmd) (string, error) {
	osCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	osCmd.Dir = cmd.Dir
	osCmd.Env = append(os.Environ(), cmd.Env...)
	if cmd.Log {
		log.Printf(
			"Exec cmd: [%s] %s %s\n",
			cmd.Dir,
			cmd.Name,
			strings.Join(cmd.Args, " "),
		)
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	osCmd.Stdout = &out
	osCmd.Stderr = &stderr
	err := osCmd.Run()
	if err != nil {
		return "", fmt.Errorf("%w; output: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out.String(), nil
}
