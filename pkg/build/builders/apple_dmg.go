package builders

import (
	"context"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
	"dappco.re/go/process"
)

// CreateDMG records the hdiutil DMG creation flow and writes a placeholder DMG.
func (b *AppleBuilder) CreateDMG(ctx context.Context, filesystem coreio.Medium, appPath string, cfg AppleDMGConfig) error {
	if filesystem == nil {
		filesystem = coreio.Local
	}
	if appPath == "" {
		return coreerr.E("AppleBuilder.CreateDMG", "app path is required", nil)
	}
	if cfg.OutputPath == "" {
		return coreerr.E("AppleBuilder.CreateDMG", "output path is required", nil)
	}
	if cfg.VolumeName == "" {
		cfg.VolumeName = core.TrimSuffix(ax.Base(appPath), ".app")
	}
	if cfg.IconSize <= 0 {
		cfg.IconSize = 128
	}
	if cfg.WindowSize[0] <= 0 || cfg.WindowSize[1] <= 0 {
		cfg.WindowSize = [2]int{640, 480}
	}

	outputDir := ax.Dir(cfg.OutputPath)
	if outputDir != "" && outputDir != "." {
		if err := filesystem.EnsureDir(outputDir); err != nil {
			return coreerr.E("AppleBuilder.CreateDMG", "failed to create DMG output directory", err)
		}
	}

	stageDMG := cfg.OutputPath + ".rw"
	mountPoint := cfg.OutputPath + ".mount"

	// TODO(#484): hdiutil requires macOS. The skeleton records each
	// go-process invocation and writes a placeholder DMG for downstream lanes.
	if err := b.runExternal(ctx, "hdiutil-create", process.RunOptions{
		Command: "hdiutil",
		Args: []string{
			"create",
			"-volname", cfg.VolumeName,
			"-srcfolder", appPath,
			"-ov",
			"-format", "UDRW",
			stageDMG,
		},
	}); err != nil {
		return err
	}

	if err := b.runExternal(ctx, "hdiutil-attach", process.RunOptions{
		Command: "hdiutil",
		Args: []string{
			"attach",
			"-readwrite",
			"-noverify",
			"-noautoopen",
			"-mountpoint", mountPoint,
			stageDMG,
		},
	}); err != nil {
		return err
	}

	if err := b.runExternal(ctx, "hdiutil-detach", process.RunOptions{
		Command: "hdiutil",
		Args:    []string{"detach", mountPoint},
	}); err != nil {
		return err
	}

	if err := b.runExternal(ctx, "hdiutil-convert", process.RunOptions{
		Command: "hdiutil",
		Args: []string{
			"convert",
			stageDMG,
			"-format", "UDZO",
			"-ov",
			"-o", cfg.OutputPath,
		},
	}); err != nil {
		return err
	}

	placeholder := core.Sprintf(
		"AppleBuilder DMG skeleton\napp=%s\nvolume=%s\nbackground=%s\n",
		appPath,
		cfg.VolumeName,
		cfg.BackgroundPath,
	)
	if err := filesystem.WriteMode(cfg.OutputPath, placeholder, 0o644); err != nil {
		return coreerr.E("AppleBuilder.CreateDMG", "failed to write placeholder DMG", err)
	}

	return nil
}
