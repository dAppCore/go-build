package buildcmd

import (
	"github.com/dappcore/go"
	"github.com/dappcore/go-build/go/internal/cli"
	"github.com/dappcore/go-build/go/pkg/build"
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
