package generators

import (
	"context"
	"sync"

	"dappco.re/go/core/build/internal/ax"
)

var (
	dockerRuntimeOnce sync.Once
	dockerRuntimeOK   bool
)

func dockerRuntimeAvailable() bool {
	dockerRuntimeOnce.Do(func() {
		dockerRuntimeOK = ax.Exec(context.Background(), "docker", "info") == nil
	})
	return dockerRuntimeOK
}
