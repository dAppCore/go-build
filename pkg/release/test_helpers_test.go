package release

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if err := ax.ExecDir(context.Background(), dir, "git", args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}
