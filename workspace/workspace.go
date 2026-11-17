package workspace

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func Create(ctx context.Context, id string) error {
	return docker(ctx, "volume", "create", id)
}

func Remove(ctx context.Context, id string) error {
	return docker(ctx, "volume", "rm", id)
}

func docker(ctx context.Context, args ...string) error {
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}

	return nil
}
