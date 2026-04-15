// Package servicecmd registers background service and daemon commands.
package servicecmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/cmdutil"
	buildservice "dappco.re/go/core/build/pkg/service"
	"dappco.re/go/core/cli/pkg/cli"
	coreerr "dappco.re/go/core/log"
)

func init() {
	cli.RegisterCommands(AddServiceCommands)
}

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

type serviceRequest struct {
	Name             string
	DisplayName      string
	Description      string
	ProjectDir       string
	Output           string
	Format           string
	APIAddr          string
	HealthAddr       string
	PIDFile          string
	WatchPaths       string
	WatchInterval    string
	ScheduleInterval string
	AutoRebuild      bool
	AutoRebuildSet   bool
}

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
	return serviceRequest{
		Name:             cmdutil.OptionString(opts, "name"),
		DisplayName:      cmdutil.OptionString(opts, "display-name", "display_name"),
		Description:      cmdutil.OptionString(opts, "description"),
		ProjectDir:       cmdutil.OptionString(opts, "project-dir", "project_dir"),
		Output:           cmdutil.OptionString(opts, "output"),
		Format:           cmdutil.OptionString(opts, "format"),
		APIAddr:          cmdutil.OptionString(opts, "addr", "api-addr", "api_addr"),
		HealthAddr:       cmdutil.OptionString(opts, "health-addr", "health_addr"),
		PIDFile:          cmdutil.OptionString(opts, "pid-file", "pid_file"),
		WatchPaths:       cmdutil.OptionString(opts, "watch-paths", "watch_paths"),
		WatchInterval:    cmdutil.OptionString(opts, "watch-interval", "watch_interval"),
		ScheduleInterval: cmdutil.OptionString(opts, "schedule-interval", "schedule_interval"),
		AutoRebuild:      cmdutil.OptionBoolDefault(opts, true, "auto-rebuild", "auto_rebuild"),
		AutoRebuildSet:   cmdutil.OptionHas(opts, "auto-rebuild", "auto_rebuild"),
	}
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
	cwd, err := serviceGetwd()
	if err != nil {
		return buildservice.Config{}, coreerr.E("service.loadServiceConfig", "failed to get working directory", err)
	}

	projectDir := req.ProjectDir
	if projectDir == "" {
		projectDir = cwd
	} else if !filepath.IsAbs(projectDir) {
		projectDir = filepath.Join(cwd, projectDir)
	}

	cfg, err := resolveServiceCfg(projectDir)
	if err != nil {
		return buildservice.Config{}, err
	}

	if err := applyServiceOverrides(&cfg, req); err != nil {
		return buildservice.Config{}, err
	}
	return cfg.Normalized(), nil
}

func applyServiceOverrides(cfg *buildservice.Config, req serviceRequest) error {
	if cfg == nil {
		return nil
	}

	if req.Name != "" {
		cfg.Name = req.Name
	}
	if req.DisplayName != "" {
		cfg.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		cfg.Description = req.Description
	}
	if req.APIAddr != "" {
		cfg.APIAddr = req.APIAddr
	}
	if req.HealthAddr != "" {
		cfg.HealthAddr = req.HealthAddr
	}
	if req.PIDFile != "" {
		cfg.PIDFile = req.PIDFile
		if !filepath.IsAbs(cfg.PIDFile) {
			cfg.PIDFile = filepath.Join(cfg.ProjectDir, cfg.PIDFile)
		}
	}
	if req.WatchPaths != "" {
		cfg.WatchPaths = parseCSV(req.WatchPaths)
	}
	if req.WatchInterval != "" {
		duration, err := time.ParseDuration(req.WatchInterval)
		if err != nil {
			return coreerr.E("service.applyServiceOverrides", "invalid watch interval", err)
		}
		cfg.WatchInterval = duration
	}
	if req.ScheduleInterval != "" {
		duration, err := time.ParseDuration(req.ScheduleInterval)
		if err != nil {
			return coreerr.E("service.applyServiceOverrides", "invalid schedule interval", err)
		}
		cfg.ScheduleInterval = duration
	}
	if req.AutoRebuildSet {
		cfg.AutoRebuild = req.AutoRebuild
	}

	return nil
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		paths = append(paths, part)
	}
	return paths
}
