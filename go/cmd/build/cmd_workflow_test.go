package buildcmd

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/buildtest"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

func TestBuildCmd_resolveReleaseWorkflowOutputPathInputGood(t *testing.T) {
	t.Run("accepts the preferred output path", func(t *testing.T) {
		path := requireBuildCmdString(t, build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "", ""))
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the snake_case output path alias", func(t *testing.T) {
		path := requireBuildCmdString(t, build.ResolveReleaseWorkflowOutputPath("", "ci/release.yml", ""))
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts the legacy output alias", func(t *testing.T) {
		path := requireBuildCmdString(t, build.ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml"))
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})

	t.Run("accepts matching output aliases", func(t *testing.T) {
		path := requireBuildCmdString(t, build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ci/release.yml", "ci/release.yml"))
		if !stdlibAssertEqual("ci/release.yml", path) {
			t.Fatalf("want %v, got %v", "ci/release.yml", path)
		}

	})
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathInputBad(t *testing.T) {
	message := requireBuildCmdError(t, build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops/release.yml", ""))
	if !stdlibAssertContains(message, "output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", message, "output aliases specify different locations")
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "./ci/release.yml", "ci/release.yml", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_CamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowCamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "ci/release.yml", "", "", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowHyphenGood(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "ci/release.yml", "", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowSnakeGood(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "", "ci/release.yml", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	message := requireBuildCmdError(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "ops/release.yml", "", "", "", ""))
	if !stdlibAssertContains(message, "workflow output aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", message, "workflow output aliases specify different locations")
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_HyphenatedGood(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "", "ci/release.yml", "", "", "", "", "", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteEquivalent_Good(t *testing.T) {
	projectDir := t.TempDir()
	absolutePath := ax.Join(projectDir, "ci", "release.yml")

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", absolutePath))
	if !stdlibAssertEqual(absolutePath, path) {
		t.Fatalf("want %v, got %v", absolutePath, path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteDirectory_Good(t *testing.T) {
	projectDir := t.TempDir()
	absoluteDir := ax.Join(projectDir, "ops")
	requireBuildCmdOK(t, storage.Local.EnsureDir(absoluteDir))

	path := requireBuildCmdString(t, resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", absoluteDir, "", "", "", ""))
	if !stdlibAssertEqual(ax.Join(absoluteDir, "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(absoluteDir, "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_WorkflowPathGood(t *testing.T) {
	projectDir := t.TempDir()

	path := requireBuildCmdString(t, resolveReleaseWorkflowInputPathAliases(projectDir, "", "ci/release.yml", "", ""))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "release.yml"), path)
	}

}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	message := requireBuildCmdError(t, resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "ops/release.yml", "", ""))
	if !stdlibAssertContains(message, "workflow path aliases specify different locations") {
		t.Fatalf("expected %v to contain %v", message, "workflow path aliases specify different locations")
	}

}

func TestBuildCmd_RunReleaseWorkflowGood(t *testing.T) {
	projectDir := t.TempDir()

	t.Run("writes to the conventional workflow path by default", func(t *testing.T) {
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "", ""))

		path := build.ReleaseWorkflowPath(projectDir)
		content := requireBuildCmdString(t, storage.Local.Read(path))
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
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, customPath, ""))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, customPath)))
		buildtest.AssertReleaseWorkflowContent(t, content)

	})

	t.Run("writes release.yml inside a directory-style relative path", func(t *testing.T) {
		customPath := "ci/"
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, customPath, ""))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, "ci", "release.yml")))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside an existing directory without a trailing slash", func(t *testing.T) {
		requireBuildCmdOK(t, storage.Local.EnsureDir(ax.Join(projectDir, "ops")))

		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "ops", ""))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, "ops", "release.yml")))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside a bare directory-style path", func(t *testing.T) {
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "ci", ""))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, "ci", "release.yml")))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside a current-directory-prefixed directory-style path", func(t *testing.T) {
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "./ci", ""))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, "ci", "release.yml")))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside the conventional workflows directory", func(t *testing.T) {
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, ".github/workflows", ""))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml")))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes release.yml inside a current-directory-prefixed workflows directory", func(t *testing.T) {
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "./.github/workflows", ""))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml")))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the output alias", func(t *testing.T) {
		customPath := "ci/alias-release.yml"
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "", customPath))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, customPath)))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the output-path alias", func(t *testing.T) {
		customPath := "ci/output-path-release.yml"
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "", customPath))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, customPath)))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the output_path alias", func(t *testing.T) {
		customPath := "ci/output_path-release.yml"
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "", customPath))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, customPath)))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the workflow-output alias", func(t *testing.T) {
		customPath := "ci/workflow-output-release.yml"
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "", customPath))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, customPath)))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})

	t.Run("writes to the workflow_output alias", func(t *testing.T) {
		customPath := "ci/workflow_output-release.yml"
		requireBuildCmdOK(t, runReleaseWorkflowInDir(projectDir, "", customPath))

		content := requireBuildCmdString(t, storage.Local.Read(ax.Join(projectDir, customPath)))
		buildtest.AssertReleaseWorkflowTriggers(t, content)

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestCmdWorkflow_AddWorkflowCommand_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AddWorkflowCommand(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdWorkflow_AddWorkflowCommand_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AddWorkflowCommand(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdWorkflow_AddWorkflowCommand_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AddWorkflowCommand(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
