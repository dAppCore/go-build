package generators

import (
	"context"
	"sync"
	"time"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

var (
	dockerRuntimeMu      sync.Mutex
	dockerRuntimeChecked bool
	dockerRuntimeOK      bool
	dockerRuntimeCommand string
)

var availabilityProbeTimeout = 2 * time.Second

func dockerRuntimeAvailable() bool {
	ctx, cancel := availabilityProbeContext()
	defer cancel()

	return dockerRuntimeAvailableWithContext(ctx)
}

func dockerRuntimeAvailableWithContext(ctx context.Context) bool {
	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return false
	}

	dockerRuntimeMu.Lock()
	defer dockerRuntimeMu.Unlock()

	if dockerRuntimeChecked && dockerRuntimeOK && dockerRuntimeCommand == dockerCommand {
		return dockerRuntimeOK
	}

	if err := ctx.Err(); err != nil {
		return false
	}

	err = ax.Exec(ctx, dockerCommand, "--help")
	if err != nil && ctx.Err() != nil {
		return false
	}

	dockerRuntimeCommand = dockerCommand
	dockerRuntimeOK = err == nil
	dockerRuntimeChecked = dockerRuntimeOK

	return dockerRuntimeOK
}

func resolveDockerRuntimeCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/docker",
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

func availabilityProbeContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), availabilityProbeTimeout)
}
