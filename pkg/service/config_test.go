package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	nativeservice "github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingNativeController struct {
	installed bool
	started   bool
	stopped   bool
	removed   bool
}

func (r *recordingNativeController) Install() error {
	r.installed = true
	return nil
}

func (r *recordingNativeController) Start() error {
	r.started = true
	return nil
}

func (r *recordingNativeController) Stop() error {
	r.stopped = true
	return nil
}

func (r *recordingNativeController) Uninstall() error {
	r.removed = true
	return nil
}

func TestDefaultConfig_Normalized_Good(t *testing.T) {
	projectDir := t.TempDir()

	cfg := DefaultConfig(projectDir).Normalized()

	assert.Equal(t, projectDir, cfg.ProjectDir)
	assert.Equal(t, "127.0.0.1:9101", cfg.APIAddr)
	assert.Equal(t, "127.0.0.1:9102", cfg.HealthAddr)
	assert.True(t, cfg.AutoRebuild)
	assert.Contains(t, cfg.Arguments, "service")
	assert.Contains(t, cfg.Arguments, "run")
	assert.Contains(t, cfg.Arguments, projectDir)
	assert.Equal(t, projectDir, cfg.Environment["CORE_BUILD_PROJECT_DIR"])
}

func TestResolveConfig_UsesBuildMetadata_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".core", "build.yaml"), []byte(`version: 1
project:
  name: "Core Build"
  binary: "core-builder"
  description: "Background build daemon"
`), 0o644))

	cfg, err := ResolveConfig(projectDir)
	require.NoError(t, err)

	assert.Equal(t, "core-builder", cfg.Name)
	assert.Equal(t, "Core Builder", cfg.DisplayName)
	assert.Equal(t, "Background build daemon", cfg.Description)
}

func TestResolveNativeFormat_Good(t *testing.T) {
	format, err := ResolveNativeFormat("launchd")
	require.NoError(t, err)
	assert.Equal(t, NativeFormatLaunchd, format)
}

func TestExport_Systemd_Good(t *testing.T) {
	cfg := DefaultConfig(t.TempDir()).Normalized()

	exported, err := Export(cfg, "systemd")
	require.NoError(t, err)

	assert.Equal(t, NativeFormatSystemd, exported.Format)
	assert.Equal(t, cfg.Name+".service", exported.Filename)
	assert.Contains(t, exported.Content, "[Unit]")
	assert.Contains(t, exported.Content, "ExecStart=")
	assert.Contains(t, exported.Content, cfg.ProjectDir)
}

func TestExport_Launchd_Good(t *testing.T) {
	cfg := DefaultConfig(t.TempDir()).Normalized()

	exported, err := Export(cfg, "launchd")
	require.NoError(t, err)

	assert.Equal(t, NativeFormatLaunchd, exported.Format)
	assert.Equal(t, cfg.Name+".plist", exported.Filename)
	assert.Contains(t, exported.Content, "<plist")
	assert.Contains(t, exported.Content, "<key>ProgramArguments</key>")
	assert.Contains(t, exported.Content, xmlEscape(cfg.Executable))
}

func TestOSManager_ServiceConfigMapping_Good(t *testing.T) {
	originalNewNativeService := newNativeService
	t.Cleanup(func() {
		newNativeService = originalNewNativeService
	})

	controller := &recordingNativeController{}
	var recorded *nativeservice.Config
	newNativeService = func(program nativeservice.Interface, cfg *nativeservice.Config) (nativeController, error) {
		recorded = cfg
		return controller, nil
	}

	manager := &OSManager{}
	cfg := DefaultConfig(t.TempDir()).Normalized()
	cfg.Name = "core-build"
	cfg.DisplayName = "Core Build"
	cfg.Description = "Background build daemon"
	cfg.WatchInterval = 5 * time.Second
	cfg = cfg.Normalized()

	err := manager.Install(cfg)
	require.NoError(t, err)
	require.NotNil(t, recorded)

	assert.True(t, controller.installed)
	assert.Equal(t, cfg.Name, recorded.Name)
	assert.Equal(t, cfg.DisplayName, recorded.DisplayName)
	assert.Equal(t, cfg.Description, recorded.Description)
	assert.Equal(t, cfg.Executable, recorded.Executable)
	assert.Equal(t, cfg.WorkingDirectory, recorded.WorkingDirectory)
	assert.Equal(t, cfg.Environment["CORE_BUILD_API_ADDR"], recorded.EnvVars["CORE_BUILD_API_ADDR"])
	assert.Contains(t, recorded.Arguments, "--watch-interval")
}
