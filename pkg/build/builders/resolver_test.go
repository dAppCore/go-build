package builders

import (
	"io/fs"
	"testing"

	"dappco.re/go/build/pkg/build"
	"errors"
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
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected error %v to be %v", err, fs.ErrNotExist)
	}

}
