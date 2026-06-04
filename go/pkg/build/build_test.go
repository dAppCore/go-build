package build

import core "dappco.re/go"

func TestBuild_Target_String_Good(t *core.T) {
	target := Target{OS: "linux", Arch: "amd64"}
	got := target.String()
	core.AssertEqual(t, "linux/amd64", got)
	core.AssertContains(t, got, "linux")
}

func TestBuild_Target_String_Bad(t *core.T) {
	target := Target{}
	got := target.String()
	core.AssertEqual(t, "/", got)
	core.AssertLen(t, got, 1)
}

func TestBuild_Target_String_Ugly(t *core.T) {
	target := Target{OS: "darwin", Arch: "arm64/v8"}
	got := target.String()
	core.AssertEqual(t, "darwin/arm64/v8", got)
	core.AssertContains(t, got, "arm64")
}
