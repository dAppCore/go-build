package build

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/buildtest"
	"dappco.re/go/io"
)

func TestWorkflow_WriteReleaseWorkflow_Good(t *testing.T) {
	t.Run("writes the embedded template to the default path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read(DefaultReleaseWorkflowPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(string(template), content) {
			t.Fatalf("want %v, got %v", string(template), content)
		}
		buildtest.AssertReleaseWorkflowContent(t, content)

	})

	t.Run("writes to a custom path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "custom/workflow.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("custom/workflow.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("trims surrounding whitespace from the output path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "  ci  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("writes release.yml for a bare directory-style path", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "ci")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("writes release.yml inside an existing directory", func(t *testing.T) {
		projectDir := t.TempDir()
		outputDir := ax.Join(projectDir, "ci")
		if err := ax.MkdirAll(outputDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := WriteReleaseWorkflow(io.Local, outputDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(outputDir, DefaultReleaseWorkflowFileName))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(string(template), content) {
			t.Fatalf("want %v, got %v", string(template), content)
		}

	})

	t.Run("writes release.yml for directory-style output paths", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		err := WriteReleaseWorkflow(fs, "ci/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := fs.Read("ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(content) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("creates parent directories on a real filesystem", func(t *testing.T) {
		projectDir := t.TempDir()
		path := ax.Join(projectDir, ".github", "workflows", "release.yml")

		err := WriteReleaseWorkflow(io.Local, path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(string(template), content) {
			t.Fatalf("want %v, got %v", string(template), content)
		}

	})
}

func TestWorkflow_WriteReleaseWorkflow_Bad(t *testing.T) {
	t.Run("rejects a nil filesystem medium", func(t *testing.T) {
		err := WriteReleaseWorkflow(nil, "")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "filesystem medium is required") {
			t.Fatalf("expected %v to contain %v", err.Error(), "filesystem medium is required")
		}

	})
}

func TestWorkflow_ReleaseWorkflowPath_Good(t *testing.T) {
	if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ReleaseWorkflowPath("/tmp/project")) {
		t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ReleaseWorkflowPath("/tmp/project"))
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPathWithMedium_Good(t *testing.T) {
	t.Run("treats an existing directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path := ResolveReleaseWorkflowOutputPathWithMedium(fs, "/tmp/project", "ci")
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("keeps explicit file paths unchanged", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path := ResolveReleaseWorkflowOutputPathWithMedium(fs, "/tmp/project", "ci/release.yml")
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowPath_Good(t *testing.T) {
	t.Run("uses the conventional path when empty", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "")) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ""))
		}

	})

	t.Run("joins relative paths to the project directory", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml"))
		}

	})

	t.Run("treats bare relative directory names as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci"))
		}

	})

	t.Run("treats current-directory-prefixed directory names as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./ci")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./ci"))
		}

	})

	t.Run("treats the conventional workflows directory as a directory", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ".github/workflows")) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ".github/workflows"))
		}

	})

	t.Run("treats current-directory-prefixed workflows directories as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./.github/workflows")) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./.github/workflows"))
		}

	})

	t.Run("keeps nested extensionless paths as files", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release", ResolveReleaseWorkflowPath("/tmp/project", "ci/release")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release", ResolveReleaseWorkflowPath("/tmp/project", "ci/release"))
		}

	})

	t.Run("treats the current directory as a workflow directory", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ".")) {
			t.Fatalf("want %v, got %v", "/tmp/project/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "."))
		}

	})

	t.Run("treats trailing-slash relative paths as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/")) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/"))
		}

	})

	t.Run("keeps absolute paths unchanged", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml")) {
			t.Fatalf("want %v, got %v", "/tmp/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml"))
		}

	})

	t.Run("treats trailing-slash absolute paths as directories", func(t *testing.T) {
		if !stdlibAssertEqual("/tmp/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/workflows/")) {
			t.Fatalf("want %v, got %v", "/tmp/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/workflows/"))
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPath_Good(t *testing.T) {
	t.Run("uses the conventional path when both inputs are empty", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("accepts path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts bare directory-style path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts current-directory-prefixed directory-style path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "./ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the conventional workflows directory as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", ".github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("accepts current-directory-prefixed workflows directories as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "./.github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("keeps nested extensionless paths as files", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release", path)
		}

	})

	t.Run("accepts the current directory as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", ".", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/release.yml", path)
		}

	})

	t.Run("accepts output as an alias for path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("trims surrounding whitespace from inputs", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "  ci  ", "  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts matching path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts matching directory-style path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/", "ci/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPath_Bad(t *testing.T) {
	t.Run("rejects conflicting path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ops/release.yml")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(path) {
			t.Fatalf("expected empty, got %v", path)
		}
		if !stdlibAssertContains(err.Error(), "path and output specify different locations") {
			t.Fatalf("expected %v to contain %v", err.Error(), "path and output specify different locations")
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathWithMedium_Good(t *testing.T) {
	t.Run("treats an existing directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("treats a bare directory-style path as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("treats a current-directory-prefixed directory-style path as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "./ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("treats the conventional workflows directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", ".github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("treats current-directory-prefixed workflows directories as workflow directories", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "./.github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/.github/workflows/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/.github/workflows/release.yml", path)
		}

	})

	t.Run("keeps a file path unchanged when the target is not a directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("normalizes matching directory aliases", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "ci/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("trims surrounding whitespace before resolving", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "  ci  ", "  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathAliases_Good(t *testing.T) {
	t.Run("accepts the preferred path input", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the workflowPath alias", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "ci", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow_path alias", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow-path alias", func(t *testing.T) {
		fs := io.NewMemoryMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "", "", "ci")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})

	t.Run("normalises matching aliases", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir("/tmp/project/ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci/", "./ci", "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("/tmp/project/ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "/tmp/project/ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathAliases_Bad(t *testing.T) {
	fs := io.NewMemoryMedium()

	path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci/release.yml", "ops/release.yml", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "path aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "path aliases specify different locations")
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPath_Good(t *testing.T) {
	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the snake_case output path alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the hyphenated output path alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "ci/release.yml", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the legacy output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("trims surrounding whitespace from aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("  ci/release.yml  ", "  ", "  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts matching aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("normalises equivalent path aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "./ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPath_Bad(t *testing.T) {
	path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops/release.yml", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliases_Good(t *testing.T) {
	t.Run("accepts workflowOutputPath aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "ci/release.yml", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the hyphenated workflowOutputPath alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "", "ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow_output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "ci/release.yml", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the workflow-output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "", "ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("normalises matching workflow output aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("ci/release.yml", "", "", "./ci/release.yml", "ci/release.yml", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliasesInProject_Good(t *testing.T) {
	projectDir := t.TempDir()
	absolutePath := ax.Join(projectDir, "ci", "release.yml")
	absoluteDirectory := ax.Join(projectDir, "ops")
	if err := ax.MkdirAll(absoluteDirectory, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(absolutePath, path) {
			t.Fatalf("want %v, got %v", absolutePath, path)
		}

	})

	t.Run("accepts an absolute workflow output alias equivalent to the project path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "", "", "", "", absolutePath, "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(absolutePath, path) {
			t.Fatalf("want %v, got %v", absolutePath, path)
		}

	})

	t.Run("accepts matching relative and absolute aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", absolutePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(absolutePath, path) {
			t.Fatalf("want %v, got %v", absolutePath, path)
		}

	})

	t.Run("treats an existing absolute directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMemoryMedium()
		if err := fs.EnsureDir(absoluteDirectory); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, err := ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(fs, projectDir, "", "", "", "", absoluteDirectory, "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ax.Join(absoluteDirectory, DefaultReleaseWorkflowFileName), path) {
			t.Fatalf("want %v, got %v", ax.Join(absoluteDirectory, DefaultReleaseWorkflowFileName), path)
		}

	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliasesInProject_Bad(t *testing.T) {
	projectDir := t.TempDir()

	path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", ax.Join(projectDir, "ops", "release.yml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliases_Bad(t *testing.T) {
	path, err := ResolveReleaseWorkflowOutputPathAliases("ci/release.yml", "", "", "", "ops/release.yml", "", "", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(path) {
		t.Fatalf("expected empty, got %v", path)
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}
