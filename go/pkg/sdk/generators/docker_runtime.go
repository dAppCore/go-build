package generators

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stdio "io"
	"strconv"
	"sync"
	"time"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

var (
	dockerRuntimeMu      sync.Mutex
	dockerRuntimeChecked bool
	dockerRuntimeOK      bool
	dockerRuntimeCommand string
	dockerRuntimeState   string
)

var availabilityProbeTimeout = 2 * time.Second

const dockerRuntimeFingerprintBytes = 4 * 1024

func dockerRuntimeAvailable() bool {
	ctx, cancel := availabilityProbeContext()
	defer cancel()

	return dockerRuntimeAvailableWithContext(ctx)
}

func dockerRuntimeAvailableWithContext(ctx context.Context) bool {
	if err := ctx.Err(); err != nil {
		return false
	}

	dockerCommand := resolveDockerRuntimeCli()
	if !dockerCommand.OK {
		return false
	}
	command := dockerCommand.Value.(string)

	commandState := dockerRuntimeCommandState(command)
	if !commandState.OK {
		return false
	}
	state := commandState.Value.(string)

	if cached, ok := cachedDockerRuntimeAvailability(command, state); ok {
		return cached
	}

	run := ax.Exec(ctx, command, "--help")
	if !run.OK && ctx.Err() != nil {
		return false
	}
	if ctx.Err() != nil {
		return false
	}

	available := run.OK
	storeDockerRuntimeAvailability(command, state, available)
	return available
}

func resolveDockerRuntimeCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/docker",
			"/usr/local/bin/docker",
			"/opt/homebrew/bin/docker",
			"/Applications/Docker.app/Contents/Resources/bin/docker",
		}
	}

	command := ax.ResolveCommand("docker", paths...)
	if !command.OK {
		return core.Fail(core.E("sdk.resolveDockerRuntimeCli", "docker CLI not found. Install it from https://docs.docker.com/get-docker/", core.NewError(command.Error())))
	}

	return command
}

func availabilityProbeContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), availabilityProbeTimeout)
}

func dockerRuntimeCommandState(command string) core.Result {
	info := ax.Stat(command)
	if !info.OK {
		return info
	}
	fileInfo := info.Value.(core.FsFileInfo)

	file := ax.Open(command)
	if !file.OK {
		return file
	}
	fileValue := file.Value.(core.FsFile)
	defer func() { _ = fileValue.Close() }()

	hasher := sha256.New()
	if _, err := stdio.CopyN(hasher, fileValue, dockerRuntimeFingerprintBytes); err != nil && !core.Is(err, stdio.EOF) {
		return core.Fail(err)
	}

	return core.Ok(command + "|" +
		strconv.FormatInt(fileInfo.Size(), 10) + "|" +
		strconv.FormatInt(fileInfo.ModTime().UnixNano(), 10) + "|" +
		fileInfo.Mode().String() + "|" +
		hex.EncodeToString(hasher.Sum(nil)))
}

func cachedDockerRuntimeAvailability(command, state string) (bool, bool) {
	dockerRuntimeMu.Lock()
	defer dockerRuntimeMu.Unlock()

	if !dockerRuntimeChecked {
		return false, false
	}
	if dockerRuntimeCommand != command || dockerRuntimeState != state {
		return false, false
	}
	return dockerRuntimeOK, true
}

func storeDockerRuntimeAvailability(command, state string, available bool) {
	dockerRuntimeMu.Lock()
	defer dockerRuntimeMu.Unlock()

	dockerRuntimeCommand = command
	dockerRuntimeState = state
	dockerRuntimeOK = available
	dockerRuntimeChecked = true
}

func resetDockerRuntimeAvailabilityCache() {
	dockerRuntimeMu.Lock()
	defer dockerRuntimeMu.Unlock()

	dockerRuntimeChecked = false
	dockerRuntimeOK = false
	dockerRuntimeCommand = ""
	dockerRuntimeState = ""
}
