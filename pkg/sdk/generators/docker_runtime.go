package generators

import (
	"context"
	"sync"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

var (
	dockerRuntimeMu      sync.Mutex
	dockerRuntimeChecked bool
	dockerRuntimeOK      bool
)

func dockerRuntimeAvailable() bool {
	return dockerRuntimeAvailableWithContext(context.Background())
}

func dockerRuntimeAvailableWithContext(ctx context.Context) bool {
	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return false
	}

	dockerRuntimeMu.Lock()
	defer dockerRuntimeMu.Unlock()

	if dockerRuntimeChecked {
		return dockerRuntimeOK
	}

	if err := ctx.Err(); err != nil {
		return false
	}

	err = ax.Exec(ctx, dockerCommand, "info")
	if err != nil && ctx.Err() != nil {
		return false
	}

	dockerRuntimeChecked = true
	dockerRuntimeOK = err == nil

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
