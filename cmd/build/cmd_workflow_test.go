package buildcmd

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/buildtest"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	"dappco.re/go/io"
)

func TestBuildCmd_resolveReleaseWorkflowOutputPathInput_Good(t *testing.T) {
	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the snake_case output path alias", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the legacy output alias", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts matching output aliases", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ci/release.yml", "ci/release.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathInput_Bad(t *testing.T) {
	_, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops/release.yml", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "output aliases specify different locations")
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "./ci/release.yml", "ci/release.yml", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_CamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowCamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "ci/release.yml", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowHyphenGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "ci/release.yml", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowSnakeGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "", "ci/release.yml", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	_, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "ops/release.yml", "", "", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "workflow output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "workflow output aliases specify different locations")
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_HyphenatedGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "ci/release.yml", "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteEquivalent_Good(t *testing.T) {
	projectDir := t.TempDir()
	absolutePath := ax.Join(projectDir, "ci", "release.yml")

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", absolutePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(absolutePath, path) {
		t.Fatalf("want %v, got %v", absolutePath, path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteDirectory_Good(t *testing.T) {
	projectDir := t.TempDir()
	absoluteDir := ax.Join(projectDir, "ops")
	if err := io.Local.EnsureDir(absoluteDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", absoluteDir, "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(absoluteDir, "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(absoluteDir, "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_WorkflowPathGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowInputPathAliases(projectDir, "", "ci/release.yml", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	_, err := resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "ops/release.yml", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "workflow path aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", err.Error(), "workflow path aliases specify different locations")
	}

}

func TestBuildCmd_RunReleaseWorkflow_Good(t *testing.T) {
	projectDir := t.TempDir()

	t.Run("writes to the conventional workflow path by default", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path := build.ReleaseWorkflowPath(projectDir)
		content, err := io.Local.Read(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowContent(t, content)

	})

	t.Run("registers the build/workflow command", func(t *testing.T) {
		c := core.New()
		AddWorkflowCommand(c)

		result := c.Command("build/workflow")
		if !(result.OK) {
			t.Fatal("expected true")
		}

		command, ok := result.Value.(*core.Command)
		if !(ok) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("build/workflow", command.Path) {
			t.Fatalf("want %v, got %v", "build/workflow", command.Path)
		}
		if !stdlibAssertEqual("cmd.build.workflow.long", command.Description) {
			t.Fatalf("want %v, got %v", "cmd.build.workflow.long", command.Description)
		}

	})

	t.Run("writes to a custom relative path", func(t *testing.T) {
		customPath := "ci/release.yml"
		err := runReleaseWorkflowInDir(projectDir, customPath, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowContent(t, content)

	})

	t.Run("writes release.yml inside a directory-style relative path", func(t *testing.T) {
		customPath := "ci/"
		err := runReleaseWorkflowInDir(projectDir, customPath, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside an existing directory without a trailing slash", func(t *testing.T) {
		if err := io.Local.EnsureDir(ax.Join(projectDir, "ops")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := runReleaseWorkflowInDir(projectDir, "ops", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ops", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside a bare directory-style path", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside a current-directory-prefixed directory-style path", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "./ci", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside the conventional workflows directory", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, ".github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside a current-directory-prefixed workflows directory", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "./.github/workflows", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the output alias", func(t *testing.T) {
		customPath := "ci/alias-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the output-path alias", func(t *testing.T) {
		customPath := "ci/output-path-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the output_path alias", func(t *testing.T) {
		customPath := "ci/output_path-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the workflow-output alias", func(t *testing.T) {
		customPath := "ci/workflow-output-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the workflow_output alias", func(t *testing.T) {
		customPath := "ci/workflow_output-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})
}
