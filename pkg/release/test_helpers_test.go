package release

import (
	"context"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	require.NoError(t, ax.ExecDir(context.Background(), dir, "git", args...))
}
