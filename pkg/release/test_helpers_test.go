package release

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if result := ax.ExecDir(context.Background(), dir, "git", args...); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}
