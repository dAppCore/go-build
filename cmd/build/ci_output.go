package buildcmd

import (
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
)

func emitCIErrorAnnotation(err error) {
	if err == nil {
		return
	}

	message := core.Trim(err.Error())
	if message == "" {
		return
	}

	cli.Print("%s\n", build.FormatGitHubAnnotation("error", "", 1, message))
}
