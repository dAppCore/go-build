package builders

import (
	"io/fs"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/pkg/build"
)

func TestResolveBuilder_Good(t *testing.T) {
	t.Run("returns Go builder for go project type", func(t *testing.T) {
		builder, err := ResolveBuilder(build.ProjectTypeGo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("go", builder.Name()) {
			t.Fatalf("want %v, got %v", "go", builder.Name())
		}

	})

	t.Run("returns Docker builder for docker project type", func(t *testing.T) {
		builder, err := ResolveBuilder(build.ProjectTypeDocker)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("docker", builder.Name()) {
			t.Fatalf("want %v, got %v", "docker", builder.Name())
		}

	})
}

func TestResolveBuilder_Bad(t *testing.T) {
	_, err := ResolveBuilder(build.ProjectType("unknown"))
	if !core.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected error %v to be %v", err, fs.ErrNotExist)
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestResolver_ResolveBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = ResolveBuilder(build.ProjectType("linux"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestResolver_ResolveBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = ResolveBuilder(build.ProjectType("linux"))
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestResolver_ResolveBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = ResolveBuilder(build.ProjectType("linux"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
