// cmd_service.go registers native OS service management for the build daemon.
package buildcmd

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
	serviceGetwd           = ax.Getwd
	resolveBuildServiceCfg = buildservice.ResolveConfig
	exportBuildService     = buildservice.Export
	runBuildServiceDaemon  = buildservice.Run
	buildServiceManager    = buildservice.NewManager()
)

type serviceRequest = servicecommon.Request

// AddServiceCommands registers `core service` commands.
func AddServiceCommands(c *core.Core) {
	c.Command("service", core.Command{
		Description: "cmd.service.short",
		Action: func(opts core.Options) core.Result {
			return core.Fail(core.E("service", "use a subcommand: install, start, stop, uninstall, export", nil))
		},
	})

	c.Command("service/install", core.Command{
		Description: "cmd.service.install.short",
		Action: func(opts core.Options) core.Result {
			return runServiceInstall(serviceRequestFromOptions(opts))
		},
	})

	c.Command("service/start", core.Command{
		Description: "cmd.service.start.short",
		Action: func(opts core.Options) core.Result {
			return runServiceStart(serviceRequestFromOptions(opts))
		},
	})

	c.Command("service/stop", core.Command{
		Description: "cmd.service.stop.short",
		Action: func(opts core.Options) core.Result {
			return runServiceStop(serviceRequestFromOptions(opts))
		},
	})

	c.Command("service/uninstall", core.Command{
		Description: "cmd.service.uninstall.short",
		Action: func(opts core.Options) core.Result {
			return runServiceUninstall(serviceRequestFromOptions(opts))
		},
	})

	c.Command("service/export", core.Command{
		Description: "cmd.service.export.short",
		Action: func(opts core.Options) core.Result {
			return runServiceExport(serviceRequestFromOptions(opts))
		},
	})

	c.Command("service/run", core.Command{
		Description: "cmd.service.run.short",
		Hidden:      true,
		Action: func(opts core.Options) core.Result {
			return runServiceRun(cmdutil.ContextOrBackground(), serviceRequestFromOptions(opts))
		},
	})
}

func serviceRequestFromOptions(opts core.Options) serviceRequest {
	return servicecommon.FromOptions(opts)
}

func runServiceInstall(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	cli.Print("%s %s\n", buildHeaderStyle.Render("Service"), "Installing daemon service")
	cli.Print("  name   %s\n", buildTargetStyle.Render(cfg.Name))
	cli.Print("  addr   %s\n", buildTargetStyle.Render(cfg.APIAddr))
	cli.Print("  health %s\n", buildTargetStyle.Render(cfg.HealthAddr))

	installed := buildServiceManager.Install(cfg)
	if !installed.OK {
		return installed
	}

	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service installed")
	return core.Ok(nil)
}

func runServiceStart(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	started := buildServiceManager.Start(cfg)
	if !started.OK {
		return started
	}
	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service started")
	return core.Ok(nil)
}

func runServiceStop(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	stopped := buildServiceManager.Stop(cfg)
	if !stopped.OK {
		return stopped
	}
	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service stopped")
	return core.Ok(nil)
}

func runServiceUninstall(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	uninstalled := buildServiceManager.Uninstall(cfg)
	if !uninstalled.OK {
		return uninstalled
	}
	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service uninstalled")
	return core.Ok(nil)
}

func runServiceExport(req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	exportedResult := exportBuildService(cfg, req.Format)
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

	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), outputPath)
	return core.Ok(nil)
}

func runServiceRun(ctx context.Context, req serviceRequest) core.Result {
	cfgResult := loadServiceConfig(req)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	signalContext, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return runBuildServiceDaemon(signalContext, cfg)
}

func loadServiceConfig(req serviceRequest) core.Result {
	return servicecommon.LoadConfig(req, serviceGetwd, resolveBuildServiceCfg)
}

func applyServiceOverrides(cfg *buildservice.Config, req serviceRequest) core.Result {
	return servicecommon.ApplyOverrides(cfg, req)
}

func parseServiceCSV(value string) []string {
	return servicecommon.ParseCSV(value)
}
