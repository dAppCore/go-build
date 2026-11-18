package buildcmd

import "github.com/dappcore/go-build/go/pkg/build"

func emitCIAnnotationForTest(err error) string {
	if err == nil {
		return ""
	}
	return build.FormatGitHubAnnotation("error", "", 1, err.Error())
}
