package builders

import (
	"context"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/build/pkg/storage"
)

// CreateDMG records the hdiutil DMG creation flow and writes a placeholder DMG.
func (b *AppleBuilder) CreateDMG(ctx context.Context, filesystem coreio.Medium, appPath string, cfg AppleDMGConfig) core.Result {
	if filesystem == nil {
		filesystem = coreio.Local
	}
	if appPath == "" {
		return core.Fail(core.E("AppleBuilder.CreateDMG", "app path is required", nil))
	}
	if cfg.OutputPath == "" {
		return core.Fail(core.E("AppleBuilder.CreateDMG", "output path is required", nil))
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
		created := filesystem.EnsureDir(outputDir)
		if !created.OK {
			return core.Fail(core.E("AppleBuilder.CreateDMG", "failed to create DMG output directory", core.NewError(created.Error())))
		}
	}

	stageDMG := cfg.OutputPath + ".rw"
	mountPoint := cfg.OutputPath + ".mount"

	// TODO(#484): hdiutil requires macOS. The skeleton records each
	// command invocation and writes a placeholder DMG for downstream lanes.
	created := b.runExternal(ctx, "hdiutil-create", RunOptions{
		Command: "hdiutil",
		Args: []string{
			"create",
			"-volname", cfg.VolumeName,
			"-srcfolder", appPath,
			"-ov",
			"-format", "UDRW",
			stageDMG,
		},
	})
	if !created.OK {
		return created
	}

	attached := b.runExternal(ctx, "hdiutil-attach", RunOptions{
		Command: "hdiutil",
		Args: []string{
			"attach",
			"-readwrite",
			"-noverify",
			"-noautoopen",
			"-mountpoint", mountPoint,
			stageDMG,
		},
	})
	if !attached.OK {
		return attached
	}

	detached := b.runExternal(ctx, "hdiutil-detach", RunOptions{
		Command: "hdiutil",
		Args:    []string{"detach", mountPoint},
	})
	if !detached.OK {
		return detached
	}

	converted := b.runExternal(ctx, "hdiutil-convert", RunOptions{
		Command: "hdiutil",
		Args: []string{
			"convert",
			stageDMG,
			"-format", "UDZO",
			"-ov",
			"-o", cfg.OutputPath,
		},
	})
	if !converted.OK {
		return converted
	}

	// On non-darwin hosts hdiutil did not execute; write a skeleton marker so
	// downstream lanes still receive a file. On darwin the real hdiutil convert
	// output above is the artifact and must not be overwritten.
	if firstNonEmptyApple(b.hostOS, runtime.GOOS) != "darwin" {
		placeholder := core.Sprintf(
			"AppleBuilder DMG skeleton\napp=%s\nvolume=%s\nbackground=%s\n",
			appPath, cfg.VolumeName, cfg.BackgroundPath,
		)
		written := filesystem.WriteMode(cfg.OutputPath, placeholder, 0o644)
		if !written.OK {
			return core.Fail(core.E("AppleBuilder.CreateDMG", "failed to write placeholder DMG", written))
		}
	}

	return core.Ok(nil)
}
