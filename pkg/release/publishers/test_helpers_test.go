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
	if result := ax.ExecDir(context.Background(), dir, command, args...); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}
