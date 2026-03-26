package generators

import (
	"io"
	"os/exec"
	"sync"
)

var (
	dockerRuntimeOnce sync.Once
	dockerRuntimeOK   bool
)

func dockerRuntimeAvailable() bool {
	dockerRuntimeOnce.Do(func() {
		cmd := exec.Command("docker", "info")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		dockerRuntimeOK = cmd.Run() == nil
	})
	return dockerRuntimeOK
}
