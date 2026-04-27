// Package servicecmd registers background service and daemon commands.
package servicecmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	servicecommon "dappco.re/go/build/internal/servicecmd"
	buildservice "dappco.re/go/build/pkg/service"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/core"
	coreerr "dappco.re/go/log"
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
	c.Command("service", core.Command{
		Description: "cmd.service.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(coreerr.E("service", "use a subcommand: install, start, stop, uninstall, export", nil))
		},
	})

	c.Command("service/install", core.Command{
		Description: "cmd.service.install.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceInstall(requestFromOptions(opts)))
		},
	})

	c.Command("service/start", core.Command{
		Description: "cmd.service.start.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceStart(requestFromOptions(opts)))
		},
	})

	c.Command("service/stop", core.Command{
		Description: "cmd.service.stop.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceStop(requestFromOptions(opts)))
		},
	})

	c.Command("service/uninstall", core.Command{
		Description: "cmd.service.uninstall.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceUninstall(requestFromOptions(opts)))
		},
	})

	c.Command("service/export", core.Command{
		Description: "cmd.service.export.long",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceExport(requestFromOptions(opts)))
		},
	})

	c.Command("service/run", core.Command{
		Description: "cmd.service.run.long",
		Hidden:      true,
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceRun(cmdutil.ContextOrBackground(), requestFromOptions(opts)))
		},
	})
}

func requestFromOptions(opts core.Options) serviceRequest {
	return servicecommon.FromOptions(opts)
}

func runServiceInstall(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	cli.Print("%s %s\n", serviceHeaderStyle.Render("Service"), "Installing daemon service")
	cli.Print("  name   %s\n", serviceValueStyle.Render(cfg.Name))
	cli.Print("  addr   %s\n", serviceValueStyle.Render(cfg.APIAddr))
	cli.Print("  health %s\n", serviceValueStyle.Render(cfg.HealthAddr))

	if err := serviceManager.Install(cfg); err != nil {
		return err
	}

	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service installed")
	return nil
}

func runServiceStart(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}
	if err := serviceManager.Start(cfg); err != nil {
		return err
	}
	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service started")
	return nil
}

func runServiceStop(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}
	if err := serviceManager.Stop(cfg); err != nil {
		return err
	}
	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service stopped")
	return nil
}

func runServiceUninstall(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}
	if err := serviceManager.Uninstall(cfg); err != nil {
		return err
	}
	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), "Service uninstalled")
	return nil
}

func runServiceExport(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	exported, err := exportService(cfg, req.Format)
	if err != nil {
		return err
	}

	if req.Output == "" {
		cli.Print("%s", exported.Content)
		return nil
	}

	outputPath := req.Output
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(cfg.ProjectDir, outputPath)
	}
	if err := ax.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	if err := ax.WriteFile(outputPath, []byte(exported.Content), 0o644); err != nil {
		return err
	}

	cli.Print("%s %s\n", serviceSuccessStyle.Render("Done"), outputPath)
	return nil
}

func runServiceRun(ctx context.Context, req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	signalContext, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runDaemon(signalContext, cfg)
}

func loadServiceConfig(req serviceRequest) (buildservice.Config, error) {
	return servicecommon.LoadConfig(req, serviceGetwd, resolveServiceCfg)
}

func applyServiceOverrides(cfg *buildservice.Config, req serviceRequest) error {
	return servicecommon.ApplyOverrides(cfg, req)
}

func parseCSV(value string) []string {
	return servicecommon.ParseCSV(value)
}
