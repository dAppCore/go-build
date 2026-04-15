package generators

import (
	"context"
	"strconv"
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
	dockerRuntimeState   string
)

var availabilityProbeTimeout = 2 * time.Second

func dockerRuntimeAvailable() bool {
	ctx, cancel := availabilityProbeContext()
	defer cancel()

	return dockerRuntimeAvailableWithContext(ctx)
}

func dockerRuntimeAvailableWithContext(ctx context.Context) bool {
	if err := ctx.Err(); err != nil {
		return false
	}

	dockerCommand, err := resolveDockerRuntimeCli()
	if err != nil {
		return false
	}

	commandState, err := dockerRuntimeCommandState(dockerCommand)
	if err != nil {
		return false
	}

	err = ax.Exec(ctx, dockerCommand, "--help")
	if err != nil && ctx.Err() != nil {
		return false
	}

	dockerRuntimeMu.Lock()
	defer dockerRuntimeMu.Unlock()

	dockerRuntimeCommand = dockerCommand
	dockerRuntimeState = commandState
	dockerRuntimeOK = err == nil
	dockerRuntimeChecked = true

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

func dockerRuntimeCommandState(command string) (string, error) {
	info, err := ax.Stat(command)
	if err != nil {
		return "", err
	}

	return command + "|" +
		strconv.FormatInt(info.Size(), 10) + "|" +
		strconv.FormatInt(info.ModTime().UnixNano(), 10), nil
}
