package publishers

import (
	"context"
	"io"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

var publisherStdout io.Writer
var publisherStderr io.Writer

func publisherPrint(format string, args ...any) {
	core.Print(publisherStdout, format, args...)
}

func publisherPrintln(args ...any) {
	if len(args) == 0 {
		publisherPrint("")
		return
	}

	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, core.Sprintf("%v", arg))
	}

	publisherPrint("%s", core.Join(" ", parts...))
}

func publisherRun(ctx context.Context, dir string, env []string, command string, args ...string) error {
	output, err := ax.CombinedOutput(ctx, dir, env, command, args...)
	if output != "" {
		publisherPrint("%s", output)
	}
	return err
}
