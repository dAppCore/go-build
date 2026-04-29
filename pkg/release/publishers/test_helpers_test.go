package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func capturePublisherOutput(t *testing.T, fn func()) string {
	t.Helper()

	buf := core.NewBuffer()
	oldStdout := publisherStdout
	oldStderr := publisherStderr
	publisherStdout = buf
	publisherStderr = buf
	defer func() {
		publisherStdout = oldStdout
		publisherStderr = oldStderr
	}()

	fn()
	return buf.String()
}

func runPublisherCommand(t *testing.T, dir, command string, args ...string) {
	t.Helper()
	if err := ax.ExecDir(context.Background(), dir, command, args...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}
