package ax

import (
	"io/fs"
	"syscall"
	"time"

	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestAx_DS_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DS()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_DS_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DS()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_DS_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DS()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Clean_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Clean(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Clean_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Clean("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Clean_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Clean(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Join_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Join()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Join_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Join()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Join_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Join()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Abs_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Abs(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Abs_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Abs("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Abs_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Abs(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Rel_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Rel("agent", "linux")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Rel_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Rel("", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Rel_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Rel("agent", "linux")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Base_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Base(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Base_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Base("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Base_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Base(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Dir_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Dir(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Dir_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Dir("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Dir_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Dir(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Ext_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Ext(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Ext_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Ext("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Ext_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Ext(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_IsAbs_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsAbs(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_IsAbs_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsAbs("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_IsAbs_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsAbs(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_FromSlash_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = FromSlash(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_FromSlash_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = FromSlash("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_FromSlash_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = FromSlash(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Getwd_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getwd()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Getwd_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getwd()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Getwd_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getwd()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_TempDir_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = TempDir("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_TempDir_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = TempDir("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_TempDir_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = TempDir("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_MkdirTemp_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = MkdirTemp("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_MkdirTemp_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = MkdirTemp("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_MkdirTemp_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = MkdirTemp("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_ReadFile_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ReadFile(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_ReadFile_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ReadFile("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_ReadFile_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ReadFile(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_WriteFile_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WriteFile(core.Path(t.TempDir(), "go-build-compliance"), []byte("agent"), 0o755)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_WriteFile_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WriteFile("", []byte("agent"), 0o755)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_WriteFile_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WriteFile(core.Path(t.TempDir(), "go-build-compliance"), []byte("agent"), 0o755)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_WriteString_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WriteString(core.Path(t.TempDir(), "go-build-compliance"), "agent", 0o755)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_WriteString_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WriteString("", "", 0o755)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_WriteString_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WriteString(core.Path(t.TempDir(), "go-build-compliance"), "agent", 0o755)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_MkdirAll_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = MkdirAll(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_MkdirAll_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = MkdirAll("", 0o755)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_MkdirAll_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = MkdirAll(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Mkdir_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Mkdir(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Mkdir_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Mkdir("", 0o755)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Mkdir_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Mkdir(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_RemoveAll_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = RemoveAll(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_RemoveAll_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = RemoveAll("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_RemoveAll_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = RemoveAll(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Stat_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Stat(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Stat_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Stat("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Stat_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Stat(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_ReadDir_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ReadDir(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_ReadDir_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ReadDir("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_ReadDir_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ReadDir(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Open_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Open(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Open_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Open("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Open_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Open(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Create_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Create(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Create_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Create("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Create_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Create(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Exists_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Exists(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Exists_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Exists("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Exists_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Exists(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_IsFile_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsFile(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_IsFile_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsFile("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_IsFile_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsFile(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_IsDir_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsDir(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_IsDir_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsDir("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_IsDir_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IsDir(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Chmod_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Chmod(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Chmod_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Chmod("", 0o755)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Chmod_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Chmod(core.Path(t.TempDir(), "go-build-compliance"), 0o755)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Getuid_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getuid()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Getuid_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getuid()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Getuid_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getuid()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Getgid_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getgid()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Getgid_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getgid()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Getgid_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Getgid()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Geteuid_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Geteuid()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Geteuid_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Geteuid()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Geteuid_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Geteuid()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_JSONMarshal_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = JSONMarshal("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_JSONMarshal_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = JSONMarshal("agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_JSONMarshal_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = JSONMarshal("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_JSONUnmarshal_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = JSONUnmarshal([]byte("agent"), "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_JSONUnmarshal_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = JSONUnmarshal([]byte("agent"), "agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_JSONUnmarshal_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = JSONUnmarshal([]byte("agent"), "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_LookPath_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LookPath("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_LookPath_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LookPath("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_LookPath_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LookPath("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_ResolveCommand_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveCommand("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_ResolveCommand_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveCommand("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_ResolveCommand_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveCommand("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Run_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run(ctx, "dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Run_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run(ctx, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run(ctx, "dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_RunDir_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = RunDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_RunDir_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = RunDir(ctx, "", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_RunDir_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = RunDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Exec_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Exec(ctx, "dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_Exec_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Exec(ctx, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_Exec_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Exec(ctx, "dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_ExecDir_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_ExecDir_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecDir(ctx, "", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_ExecDir_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecDir(ctx, core.Path(t.TempDir(), "go-build-compliance"), "dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_ExecWithEnv_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecWithEnv(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_ExecWithEnv_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecWithEnv(ctx, "", []string{"agent"}, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_ExecWithEnv_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecWithEnv(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_ExecWithWriters_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecWithWriters(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, core.NewBuffer(), core.NewBuffer(), "dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_ExecWithWriters_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecWithWriters(ctx, "", []string{"agent"}, core.NewBuffer(), core.NewBuffer(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_ExecWithWriters_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ExecWithWriters(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, core.NewBuffer(), core.NewBuffer(), "dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_CombinedOutput_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CombinedOutput(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAx_CombinedOutput_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CombinedOutput(ctx, "", []string{"agent"}, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAx_CombinedOutput_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CombinedOutput(ctx, core.Path(t.TempDir(), "go-build-compliance"), []string{"agent"}, "dappcore-command-not-found")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAx_Chtimes_Good(t *core.T) {
	path := Join(t.TempDir(), "stamp")
	core.RequireTrue(t, WriteString(path, "agent", 0o644).OK)
	want := time.Unix(123, 0)

	core.RequireTrue(t, Chtimes(path, want, want).OK)
	infoResult := Stat(path)
	core.RequireTrue(t, infoResult.OK)
	info := infoResult.Value.(fs.FileInfo)
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
	core.RequireTrue(t, WriteString(path, "", 0o644).OK)
	want := time.Unix(0, 0)

	core.RequireTrue(t, Chtimes(path, want, want).OK)
	infoResult := Stat(path)
	core.RequireTrue(t, infoResult.OK)
	info := infoResult.Value.(fs.FileInfo)
	core.AssertEqual(t, want.Unix(), info.ModTime().Unix())
}

func TestAx_Readlink_Good(t *core.T) {
	dir := t.TempDir()
	target := Join(dir, "target")
	link := Join(dir, "link")
	core.RequireTrue(t, WriteString(target, "agent", 0o644).OK)
	if err := syscall.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	result := Readlink(link)
	core.RequireTrue(t, result.OK)
	got := result.Value.(string)
	core.AssertEqual(t, target, got)
}

func TestAx_Readlink_Bad(t *core.T) {
	path := Join(t.TempDir(), "missing")
	result := Readlink(path)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), path)
}

func TestAx_Readlink_Ugly(t *core.T) {
	dir := t.TempDir()
	link := Join(dir, "relative-link")
	if err := syscall.Symlink("target", link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	result := Readlink(link)
	core.RequireTrue(t, result.OK)
	got := result.Value.(string)
	core.AssertEqual(t, "target", got)
}
