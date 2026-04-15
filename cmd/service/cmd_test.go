package servicecmd

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
	buildservice "dappco.re/go/build/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubManager struct {
	install func(buildservice.Config) error
	start   func(buildservice.Config) error
	stop    func(buildservice.Config) error
	remove  func(buildservice.Config) error
}

func (s stubManager) Install(cfg buildservice.Config) error {
	if s.install != nil {
		return s.install(cfg)
	}
	return nil
}

func (s stubManager) Start(cfg buildservice.Config) error {
	if s.start != nil {
		return s.start(cfg)
	}
	return nil
}

func (s stubManager) Stop(cfg buildservice.Config) error {
	if s.stop != nil {
		return s.stop(cfg)
	}
	return nil
}

func (s stubManager) Uninstall(cfg buildservice.Config) error {
	if s.remove != nil {
		return s.remove(cfg)
	}
	return nil
}

func TestAddServiceCommands_RegistersSubcommands_Good(t *testing.T) {
	c := core.New()

	AddServiceCommands(c)

	assert.True(t, c.Command("service").OK)
	assert.True(t, c.Command("service/install").OK)
	assert.True(t, c.Command("service/start").OK)
	assert.True(t, c.Command("service/stop").OK)
	assert.True(t, c.Command("service/uninstall").OK)
	assert.True(t, c.Command("service/export").OK)
	assert.True(t, c.Command("service/run").OK)
}

func TestRunServiceInstall_UsesManager_Good(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalManager := serviceManager
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		serviceManager = originalManager
	})

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{
			Name:       "core-build",
			ProjectDir: projectDir,
			APIAddr:    "127.0.0.1:9101",
			HealthAddr: "127.0.0.1:9102",
		}, nil
	}

	called := false
	serviceManager = stubManager{
		install: func(cfg buildservice.Config) error {
			called = true
			assert.Equal(t, projectDir, cfg.ProjectDir)
			assert.Equal(t, "core-build", cfg.Name)
			return nil
		},
	}

	err := runServiceInstall(serviceRequest{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRunServiceExport_WritesFile_Good(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalExport := exportService
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		exportService = originalExport
	})

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{Name: "core-build", ProjectDir: projectDir}, nil
	}
	exportService = func(cfg buildservice.Config, format string) (buildservice.ExportedConfig, error) {
		return buildservice.ExportedConfig{
			Format:   buildservice.NativeFormatSystemd,
			Filename: "core-build.service",
			Content:  "[Unit]\nDescription=Core Build\n",
		}, nil
	}

	outputPath := filepath.Join("dist", "core-build.service")
	err := runServiceExport(serviceRequest{Output: outputPath})
	require.NoError(t, err)

	content, readErr := ax.ReadFile(filepath.Join(projectDir, outputPath))
	require.NoError(t, readErr)
	assert.Equal(t, "[Unit]\nDescription=Core Build\n", string(content))
}

func TestRunServiceRun_InvokesDaemon_Good(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalRun := runDaemon
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		runDaemon = originalRun
	})

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{Name: "core-build", ProjectDir: projectDir}, nil
	}

	called := false
	runDaemon = func(ctx context.Context, cfg buildservice.Config) error {
		called = true
		assert.Equal(t, projectDir, cfg.ProjectDir)
		return nil
	}

	err := runServiceRun(context.Background(), serviceRequest{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestApplyServiceOverrides_BadDuration(t *testing.T) {
	cfg := buildservice.Config{ProjectDir: t.TempDir()}

	err := applyServiceOverrides(&cfg, serviceRequest{WatchInterval: "not-a-duration"})
	require.Error(t, err)
}

func TestRunServiceInstall_BubblesManagerError_Bad(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalManager := serviceManager
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		serviceManager = originalManager
	})

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{Name: "core-build", ProjectDir: projectDir}, nil
	}
	serviceManager = stubManager{
		install: func(buildservice.Config) error {
			return errors.New("boom")
		},
	}

	err := runServiceInstall(serviceRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}
