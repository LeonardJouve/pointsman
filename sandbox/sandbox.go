package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/LeonardJouve/pointsman/workspace"
)

// dockerFailed is the exit code of docker run itself when it cannot run the container.
// A command that exits 125 is taken for it.
const dockerFailed = 125

// Run runs the command in a new container of the workspace's image, with the workspace folder at workspace.Mount.
// A failing command is a non-zero code; err is only for Docker's own failures.
// A cancelled ctx kills the docker client, not the container.
func Run(ctx context.Context, w workspace.Workspace, command ...string) (string, uint32, error) {
	path, err := w.Path()
	if err != nil {
		return "", 0, err
	}

	volume := fmt.Sprintf("%s:%s", path, workspace.Mount)
	args := append([]string{"run", "-q", "--rm", "-v", volume, "-w", workspace.Mount, w.Image}, command...)

	var output bytes.Buffer

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Run()
	if ctx.Err() != nil {
		return "", 0, ctx.Err()
	}

	if exitError, ok := errors.AsType[*exec.ExitError](err); ok && exitError.ExitCode() != dockerFailed {
		return output.String(), uint32(exitError.ExitCode()), nil
	}

	if err != nil {
		return "", 0, fmt.Errorf("docker run: %w: %s", err, strings.TrimSpace(output.String()))
	}

	return output.String(), 0, nil
}
