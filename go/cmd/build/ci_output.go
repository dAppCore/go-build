package buildcmd

import (
	"dappco.re/go"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/pkg/build"
)

func emitCIErrorAnnotation(result core.Result) {
	if result.OK {
		return
	}

	message := core.Trim(result.Error())
	if message == "" {
		return
	}

	cli.Print("%s\n", build.FormatGitHubAnnotation("error", "", 1, message))
}
