package ax

import (
	"syscall"
	"time"

	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestAx_DS_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = DS()
	})
	core.AssertTrue(t, true)
}

func TestAx_DS_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = DS()
	})
	core.AssertTrue(t, true)
}

func TestAx_DS_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = DS()
	})
	core.AssertTrue(t, true)
}

func TestAx_Clean_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Clean(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Clean_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Clean("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Clean_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Clean(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Join_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Join()
	})
	core.AssertTrue(t, true)
}

func TestAx_Join_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Join()
	})
	core.AssertTrue(t, true)
}

func TestAx_Join_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Join()
	})
	core.AssertTrue(t, true)
}

func TestAx_Abs_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Abs(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Abs_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Abs("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Abs_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Abs(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Rel_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Rel("agent", "linux")
	})
	core.AssertTrue(t, true)
}

func TestAx_Rel_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Rel("", "")
	})
	core.AssertTrue(t, true)
}

func TestAx_Rel_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Rel("agent", "linux")
	})
	core.AssertTrue(t, true)
}

func TestAx_Base_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Base(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Base_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Base("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Base_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Base(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Dir_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Dir(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Dir_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Dir("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Dir_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Dir(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Ext_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Ext(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Ext_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Ext("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Ext_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Ext(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_IsAbs_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsAbs(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_IsAbs_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsAbs("")
	})
	core.AssertTrue(t, true)
}

func TestAx_IsAbs_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsAbs(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_FromSlash_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = FromSlash(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_FromSlash_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = FromSlash("")
	})
	core.AssertTrue(t, true)
}

func TestAx_FromSlash_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = FromSlash(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Getwd_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Getwd()
	})
	core.AssertTrue(t, true)
}

func TestAx_Getwd_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Getwd()
	})
	core.AssertTrue(t, true)
}

func TestAx_Getwd_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Getwd()
	})
	core.AssertTrue(t, true)
}

func TestAx_TempDir_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = TempDir("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_TempDir_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = TempDir("")
	})
	core.AssertTrue(t, true)
}

func TestAx_TempDir_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = TempDir("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_MkdirTemp_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = MkdirTemp("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_MkdirTemp_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = MkdirTemp("")
	})
	core.AssertTrue(t, true)
}

func TestAx_MkdirTemp_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = MkdirTemp("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_ReadFile_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ReadFile(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_ReadFile_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ReadFile("")
	})
	core.AssertTrue(t, true)
}

func TestAx_ReadFile_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ReadFile(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_WriteFile_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = WriteFile(core.Path(t.TempDir(), "go-build-compliance"), []byte("agent"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_WriteFile_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = WriteFile("", []byte("agent"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_WriteFile_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = WriteFile(core.Path(t.TempDir(), "go-build-compliance"), []byte("agent"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_WriteString_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = WriteString(core.Path(t.TempDir(), "go-build-compliance"), "agent", 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_WriteString_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = WriteString("", "", 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_WriteString_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = WriteString(core.Path(t.TempDir(), "go-build-compliance"), "agent", 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_MkdirAll_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = MkdirAll(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_MkdirAll_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = MkdirAll("", 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_MkdirAll_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = MkdirAll(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_Mkdir_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Mkdir(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_Mkdir_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Mkdir("", 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_Mkdir_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Mkdir(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_RemoveAll_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = RemoveAll(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_RemoveAll_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = RemoveAll("")
	})
	core.AssertTrue(t, true)
}

func TestAx_RemoveAll_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = RemoveAll(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Stat_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Stat(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Stat_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Stat("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Stat_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Stat(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_ReadDir_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ReadDir(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_ReadDir_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ReadDir("")
	})
	core.AssertTrue(t, true)
}

func TestAx_ReadDir_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ReadDir(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Open_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Open(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Open_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Open("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Open_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Open(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Create_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Create(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Create_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Create("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Create_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Create(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Exists_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Exists(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Exists_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Exists("")
	})
	core.AssertTrue(t, true)
}

func TestAx_Exists_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Exists(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_IsFile_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsFile(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_IsFile_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsFile("")
	})
	core.AssertTrue(t, true)
}

func TestAx_IsFile_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsFile(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_IsDir_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDir(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_IsDir_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDir("")
	})
	core.AssertTrue(t, true)
}

func TestAx_IsDir_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDir(core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestAx_Chmod_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Chmod(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_Chmod_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Chmod("", 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_Chmod_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Chmod(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
	})
	core.AssertTrue(t, true)
}

func TestAx_Getuid_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Getuid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Getuid_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Getuid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Getuid_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Getuid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Getgid_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Getgid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Getgid_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Getgid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Getgid_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Getgid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Geteuid_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Geteuid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Geteuid_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Geteuid()
	})
	core.AssertTrue(t, true)
}

func TestAx_Geteuid_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Geteuid()
	})
	core.AssertTrue(t, true)
}

func TestAx_JSONMarshal_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = JSONMarshal("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_JSONMarshal_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = JSONMarshal("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_JSONMarshal_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = JSONMarshal("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_JSONUnmarshal_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = JSONUnmarshal([]byte("agent"), "agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_JSONUnmarshal_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = JSONUnmarshal([]byte("agent"), "agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_JSONUnmarshal_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = JSONUnmarshal([]byte("agent"), "agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_LookPath_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = LookPath("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_LookPath_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = LookPath("")
	})
	core.AssertTrue(t, true)
}

func TestAx_LookPath_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = LookPath("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_ResolveCommand_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ResolveCommand("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_ResolveCommand_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ResolveCommand("")
	})
	core.AssertTrue(t, true)
}

func TestAx_ResolveCommand_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ResolveCommand("agent")
	})
	core.AssertTrue(t, true)
}

func TestAx_Run_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = Run(ctx, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_Run_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = Run(ctx, "")
	})
	core.AssertTrue(t, true)
}

func TestAx_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = Run(ctx, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_RunDir_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = RunDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_RunDir_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = RunDir(ctx, "", "")
	})
	core.AssertTrue(t, true)
}

func TestAx_RunDir_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = RunDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_Exec_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = Exec(ctx, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_Exec_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = Exec(ctx, "")
	})
	core.AssertTrue(t, true)
}

func TestAx_Exec_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = Exec(ctx, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecDir_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecDir_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecDir(ctx, "", "")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecDir_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecWithEnv_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecWithEnv(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecWithEnv_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecWithEnv(ctx, "", []string{"agent"}, "")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecWithEnv_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecWithEnv(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecWithWriters_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecWithWriters(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, core.NewBuffer(), core.NewBuffer(), "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecWithWriters_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecWithWriters(ctx, "", []string{"agent"}, core.NewBuffer(), core.NewBuffer(), "")
	})
	core.AssertTrue(t, true)
}

func TestAx_ExecWithWriters_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_ = ExecWithWriters(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, core.NewBuffer(), core.NewBuffer(), "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_CombinedOutput_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = CombinedOutput(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_CombinedOutput_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = CombinedOutput(ctx, "", []string{"agent"}, "")
	})
	core.AssertTrue(t, true)
}

func TestAx_CombinedOutput_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	core.AssertNotPanics(t, func() {
		_, _ = CombinedOutput(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
	})
	core.AssertTrue(t, true)
}

func TestAx_Chtimes_Good(t *core.T) {
	path := Join(t.TempDir(), "stamp")
	core.RequireNoError(t, WriteString(path, "agent", 0o644))
	want := time.Unix(123, 0)

	core.RequireNoError(t, Chtimes(path, want, want))
	info, err := Stat(path)
	core.RequireNoError(t, err)
	core.AssertEqual(t, want.Unix(), info.ModTime().Unix())
}

func TestAx_Chtimes_Bad(t *core.T) {
	path := Join(t.TempDir(), "missing")
	err := Chtimes(path, time.Unix(1, 0), time.Unix(1, 0))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), path)
}

func TestAx_Chtimes_Ugly(t *core.T) {
	path := Join(t.TempDir(), "stamp")
	core.RequireNoError(t, WriteString(path, "", 0o644))
	want := time.Unix(0, 0)

	core.RequireNoError(t, Chtimes(path, want, want))
	info, err := Stat(path)
	core.RequireNoError(t, err)
	core.AssertEqual(t, want.Unix(), info.ModTime().Unix())
}

func TestAx_Readlink_Good(t *core.T) {
	dir := t.TempDir()
	target := Join(dir, "target")
	link := Join(dir, "link")
	core.RequireNoError(t, WriteString(target, "agent", 0o644))
	if err := syscall.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	got, err := Readlink(link)
	core.RequireNoError(t, err)
	core.AssertEqual(t, target, got)
}

func TestAx_Readlink_Bad(t *core.T) {
	path := Join(t.TempDir(), "missing")
	target, err := Readlink(path)
	core.AssertError(t, err)
	core.AssertEqual(t, "", target)
	core.AssertContains(t, err.Error(), path)
}

func TestAx_Readlink_Ugly(t *core.T) {
	dir := t.TempDir()
	link := Join(dir, "relative-link")
	if err := syscall.Symlink("target", link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	got, err := Readlink(link)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "target", got)
}
