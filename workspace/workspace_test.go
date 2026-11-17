package workspace

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

func requireDocker(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not on PATH")
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker daemon not answering")
	}
}

func exists(id string) bool {
	return exec.Command("docker", "volume", "inspect", id).Run() == nil
}

func TestCreateRemove(t *testing.T) {
	requireDocker(t)

	ctx := context.Background()
	id := fmt.Sprintf("pointsman-test-%d", time.Now().UnixNano())
	t.Cleanup(func() { exec.Command("docker", "volume", "rm", id).Run() })

	if err := Create(ctx, id); err != nil {
		t.Fatal(err)
	}

	if !exists(id) {
		t.Fatal("volume not created")
	}

	if err := Remove(ctx, id); err != nil {
		t.Fatal(err)
	}

	if exists(id) {
		t.Error("volume not removed")
	}
}

func TestRemoveMissing(t *testing.T) {
	requireDocker(t)

	err := Remove(context.Background(), fmt.Sprintf("pointsman-missing-%d", time.Now().UnixNano()))
	if err == nil {
		t.Fatal("expected an error")
	}

	t.Log(err)
}
