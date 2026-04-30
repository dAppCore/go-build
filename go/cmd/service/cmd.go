// Package servicecmd registers background service and daemon commands.
package servicecmd

import (
	"context"
	"os/signal"
	"syscall"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/internal/cmdutil"
	servicecommon "dappco.re/go/build/internal/servicecmd"
	buildservice "dappco.re/go/build/pkg/service"
)

var (
	serviceHeaderStyle  = cli.TitleStyle
	serviceValueStyle   = cli.ValueStyle
	serviceSuccessStyle = cli.SuccessStyle
	serviceGetwd        = ax.Getwd
	resolveServiceCfg   = buildservice.ResolveConfig
	serviceManager      = buildservice.NewManager()
	exportService       = buildservice.Export
	runDaemon           = buildservice.Run
)

type serviceRequest = servicecommon.Request

// AddServiceCommands registers `core service` commands.
func AddServiceCommands(c *core.Core) {
	_ = c.Command("service", core.Command{
		Description: "cmd.service.long",
		Action: func(opts core.Options) core.Result {
			return core.Fail(core.E("service", "use a subcommand: install, start, stop, uninstall, export", nil))
		},
	})

	_ = c.Command("service/install", core.Command{
		Description: "cmd.service.install.long",
		Action: func(opts core.Options) core.Result {
			return runServiceInstall(requestFromOptions(opts))
		},
	})

	_ = c.Command("service/start", core.Command{
		Description: "cmd.service.start.long",
		Action: func(opts core.Options) core.Result {
			return runServiceStart(requestFromOptions(opts))
		},
	})

	_ = c.Command("service/stop", core.Command{
		Description: "cmd.service.stop.long",
		Action: func(opts core.Options) core.Result {
			return runServiceStop(requestFromOptions(opts))
		},
	})

	_ = c.Command("service/uninstall", core.Command{
		Description: "cmd.service.uninstall.long",
		Action: func(opts core.Options) core.Result {
			return runServiceUninstall(requestFromOptions(opts))
		},
	})

	_ = c.Command("service/export", core.Command{
		Description: "cmd.service.export.long",
		Action: func(opts core.Options) core.Result {
			return runServiceExport(requestFromOptions(opts))
		},
	})

	_ = c.Command("service/run", core.Command{
		Description: "cmd.service.run.long",
		Hidden:      true,
		Action: func(opts core.Options) core.Result {
			return runServiceRun(cmdutil.ContextOrBackground(), requestFromOptions(opts))
		},
	})
}

func requestFromOptions(opts core.Options) serviceRequest {
	return servicecommon.FromOptions(opts)
}

func runServiceInstall(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	cli.Print("%s %s\n", serviceHeaderStyle.Render("Service"), "Installing daemon service")
	cli.Print("  name   %s\n", serviceValueStyle.Render(cfg.Name))
	cli.Print("  addr   %s\n", serviceValueStyle.Render(cfg.APIAddr))
	cli.Print("  health %s\n", serviceValueStyle.Render(cfg.HealthAddr))

	installed := serviceManager.Install(cfg)
	if !installed.OK {
		return installed
	}

	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service installed")
	return core.Ok(nil)
}

func runServiceStart(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)
	started := serviceManager.Start(cfg)
	if !started.OK {
		return started
	}
	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service started")
	return core.Ok(nil)
}

func runServiceStop(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)
	stopped := serviceManager.Stop(cfg)
	if !stopped.OK {
		return stopped
	}
	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service stopped")
	return core.Ok(nil)
}

func runServiceUninstall(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)
	uninstalled := serviceManager.Uninstall(cfg)
	if !uninstalled.OK {
		return uninstalled
	}
	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service uninstalled")
	return core.Ok(nil)
}

func runServiceExport(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	exportedResult := exportService(cfg, req.Format)
	if !exportedResult.OK {
		return exportedResult
	}
	exported := exportedResult.Value.(buildservice.ExportedConfig)

	if req.Output == "" {
		cli.Print("%s", exported.Content)
		return core.Ok(nil)
	}

	outputPath := req.Output
	if !core.PathIsAbs(outputPath) {
		outputPath = core.PathJoin(cfg.ProjectDir, outputPath)
	}
	created := ax.MkdirAll(core.PathDir(outputPath), 0o755)
	if !created.OK {
		return created
	}
	written := ax.WriteFile(outputPath, []byte(exported.Content), 0o644)
	if !written.OK {
		return written
	}

	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), outputPath)
	return core.Ok(nil)
}

func runServiceRun(ctx context.Context, req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	signalContext, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return runDaemon(signalContext, cfg)
}

func loadServiceConfig(req serviceRequest) core.Result {
	return servicecommon.LoadConfig(req, serviceGetwd, resolveServiceCfg)
}

func applyServiceOverrides(cfg *buildservice.Config, req serviceRequest) core.Result {
	return servicecommon.ApplyOverrides(cfg, req)
}
