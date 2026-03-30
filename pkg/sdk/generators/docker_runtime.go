package generators

import (
	"context"
	"sync"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

var (
	dockerRuntimeOnce sync.Once
	dockerRuntimeOK   bool
)

func dockerRuntimeAvailable() bool {
	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return false
	}

	dockerRuntimeOnce.Do(func() {
		dockerRuntimeOK = ax.Exec(context.Background(), dockerCommand, "info") == nil
	})
	return dockerRuntimeOK
}

func resolveDockerRuntimeCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/docker",
			"/opt/homebrew/bin/docker",
			"/Applications/Docker.app/Contents/Resources/bin/docker",
		}
	}

	command, err := ax.ResolveCommand("docker", paths...)
	if err != nil {
		return "", coreerr.E("sdk.resolveDockerRuntimeCli", "docker CLI not found. Install it from https://docs.docker.com/get-docker/", err)
	}

	return command, nil
}
