package buildcmd

import (
	"testing"

	"dappco.re/go"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/pkg/build"
)

// captureBuildStdout redirects cli output into a buffer for the test duration so
// assertions can inspect rendered CLI output instead of leaking it into the test
// log. The original writers are restored on cleanup.
func captureBuildStdout(t testing.TB) *core.Buffer {
	t.Helper()
	buf := core.NewBuffer()
	cli.SetStdout(buf)
	cli.SetStderr(buf)
	t.Cleanup(func() {
		cli.SetStdout(nil)
		cli.SetStderr(nil)
	})
	return buf
}

func requireBuildCmdOK(t testing.TB, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireBuildCmdError(t testing.TB, result core.Result) string {
	t.Helper()
	if result.OK {
		t.Fatal("expected error")
	}
	return result.Error()
}

func requireBuildCmdString(t testing.TB, result core.Result) string {
	t.Helper()
	return requireBuildCmdValue[string](t, result)
}

func requireBuildCmdBytes(t testing.TB, result core.Result) []byte {
	t.Helper()
	return requireBuildCmdValue[[]byte](t, result)
}

func requireBuildCmdArchiveFormat(t testing.TB, result core.Result) build.ArchiveFormat {
	t.Helper()
	return requireBuildCmdValue[build.ArchiveFormat](t, result)
}

func requireBuildCmdArtifacts(t testing.TB, result core.Result) []build.Artifact {
	t.Helper()
	return requireBuildCmdValue[[]build.Artifact](t, result)
}

func requireBuildCmdBuilder(t testing.TB, result core.Result) build.Builder {
	t.Helper()
	return requireBuildCmdValue[build.Builder](t, result)
}

func requireBuildCmdStringMap(t testing.TB, result core.Result) map[string]string {
	t.Helper()
	return requireBuildCmdValue[map[string]string](t, result)
}

func requireBuildCmdPWAExtraction(t testing.TB, result core.Result) pwaHTMLExtraction {
	t.Helper()
	return requireBuildCmdValue[pwaHTMLExtraction](t, result)
}

func requireBuildCmdValue[T any](t testing.TB, result core.Result) T {
	t.Helper()
	var zero T
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
		return zero
	}
	value, ok := result.Value.(T)
	if !ok {
		t.Fatalf("unexpected result type %T", result.Value)
		return zero
	}
	return value
}
