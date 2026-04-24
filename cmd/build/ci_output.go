package buildcmd

import (
	"strings"

	"dappco.re/go/build/pkg/build"
	"dappco.re/go/cli/pkg/cli"
)

func emitCIErrorAnnotation(err error) {
	if err == nil {
		return
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return
	}

	cli.Print("%s\n", build.FormatGitHubAnnotation("error", "", 1, message))
}
