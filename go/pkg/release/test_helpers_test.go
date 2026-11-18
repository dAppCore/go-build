package release

import (
	"context"
	"testing"

	"github.com/dappcore/go-build/go/internal/ax"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if result := ax.ExecDir(context.Background(), dir, "git", args...); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}
