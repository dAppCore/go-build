package buildcmd

import "dappco.re/go/build/pkg/build"

func emitCIAnnotationForTest(err error) string {
	if err == nil {
		return ""
	}
	return build.FormatGitHubAnnotation("error", "", 1, err.Error())
}
