package os

import (
	"bytes"
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/go-errors-context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

// NewOS creates a new instance of the OS module.
func NewOS() Service {
	return OS{}
}

// OS implements a module that interacts with the operating system.
type OS struct {
}

// RunCmd executes the system command and returns the system output.
func (s OS) RunCmd(ctx context.Context, cmd model.Cmd) (string, error) {
	osCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	osCmd.Dir = cmd.Dir
	osCmd.Env = append(os.Environ(), cmd.Env...)
	if cmd.Log {
		log.Printf(
			"Exec OS command: [%s] %s %s\n",
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

// ReadFile returns the content of the file.
func (s OS) ReadFile(ctx context.Context, path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0755)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.os.ReadFile: open file",
			Params: errors.Params{"path": path},
		})
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println(errors.WrapContext(err, errors.Context{
				Path:   "service.os.ReadFile: close file",
				Params: errors.Params{"path": path},
			}))
		}
	}()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.os.ReadFile: read file",
			Params: errors.Params{"path": path},
		})
	}
	return data, nil
}
