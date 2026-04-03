package publishers

import (
	"bytes"
	"context"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"github.com/stretchr/testify/require"
)

func capturePublisherOutput(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	oldStdout := publisherStdout
	oldStderr := publisherStderr
	publisherStdout = &buf
	publisherStderr = &buf
	defer func() {
		publisherStdout = oldStdout
		publisherStderr = oldStderr
	}()

	fn()
	return buf.String()
}

func runPublisherCommand(t *testing.T, dir, command string, args ...string) {
	t.Helper()
	require.NoError(t, ax.ExecDir(context.Background(), dir, command, args...))
}
