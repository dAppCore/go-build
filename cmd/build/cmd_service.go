// cmd_service.go registers native OS service management for the build daemon.
package buildcmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	servicecommon "dappco.re/go/build/internal/servicecmd"
	buildservice "dappco.re/go/build/pkg/service"
	"dappco.re/go/cli/pkg/cli"
	coreerr "dappco.re/go/log"
	nativeservice "github.com/kardianos/service"
)

const serviceStopTimeout = 25 * time.Second

var (
	serviceGetwd             = ax.Getwd
	resolveBuildServiceCfg   = buildservice.ResolveConfig
	exportBuildService       = buildservice.Export
	runBuildServiceDaemon    = buildservice.Run
	newBuildNativeController = func(program nativeservice.Interface, cfg *nativeservice.Config) (serviceController, error) {
		return nativeservice.New(program, cfg)
	}
)

type serviceController interface {
	Install() error
	Start() error
	Stop() error
	Uninstall() error
	Run() error
}

type serviceRequest = servicecommon.Request

type serviceProgram struct {
	cfg    buildservice.Config
	cancel context.CancelFunc
	done   chan error
	mu     sync.Mutex
}

// Start launches the build-service daemon goroutine and stores its cancel
// handle so Stop can shut it down cleanly. Implements nativeservice.Service.
//
//	_ = p.Start(svc)
func (p *serviceProgram) Start(nativeservice.Service) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.done != nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	p.cancel = cancel
	p.done = done
	go func() {
		done <- runBuildServiceDaemon(ctx, p.cfg)
	}()

	return nil
}

// Stop signals the build-service daemon to exit and waits up to
// serviceStopTimeout for graceful shutdown. Implements nativeservice.Service.
//
//	_ = p.Stop(svc)
func (p *serviceProgram) Stop(nativeservice.Service) error {
	p.mu.Lock()
	cancel := p.cancel
	done := p.done
	p.cancel = nil
	p.done = nil
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}

	select {
	case err := <-done:
		return err
	case <-time.After(serviceStopTimeout):
		return coreerr.E("service.Stop", "timed out stopping build daemon", nil)
	}
}

type controlServiceProgram struct{}

// Start is a no-op for the control program — it exists only to satisfy
// nativeservice.Service when the binary is invoked for service control
// rather than running the daemon itself.
//
//	_ = controlServiceProgram{}.Start(svc)
func (controlServiceProgram) Start(nativeservice.Service) error { return nil }

// Stop is a no-op for the control program — see Start.
//
//	_ = controlServiceProgram{}.Stop(svc)
func (controlServiceProgram) Stop(nativeservice.Service) error { return nil }

// AddServiceCommands registers `core service` commands.
func AddServiceCommands(c *core.Core) {
	c.Command("service", core.Command{
		Description: "cmd.service.short",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(coreerr.E("service", "use a subcommand: install, start, stop, uninstall, export", nil))
		},
	})

	c.Command("service/install", core.Command{
		Description: "cmd.service.install.short",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceInstall(serviceRequestFromOptions(opts)))
		},
	})

	c.Command("service/start", core.Command{
		Description: "cmd.service.start.short",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceStart(serviceRequestFromOptions(opts)))
		},
	})

	c.Command("service/stop", core.Command{
		Description: "cmd.service.stop.short",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceStop(serviceRequestFromOptions(opts)))
		},
	})

	c.Command("service/uninstall", core.Command{
		Description: "cmd.service.uninstall.short",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceUninstall(serviceRequestFromOptions(opts)))
		},
	})

	c.Command("service/export", core.Command{
		Description: "cmd.service.export.short",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceExport(serviceRequestFromOptions(opts)))
		},
	})

	c.Command("service/run", core.Command{
		Description: "cmd.service.run.short",
		Hidden:      true,
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runServiceRun(cmdutil.ContextOrBackground(), serviceRequestFromOptions(opts)))
		},
	})
}

func serviceRequestFromOptions(opts core.Options) serviceRequest {
	return servicecommon.FromOptions(opts)
}

func runServiceInstall(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	controller, err := newServiceController(cfg, controlServiceProgram{}, nil)
	if err != nil {
		return err
	}

	cli.Print("%s %s\n", buildHeaderStyle.Render("Service"), "Installing daemon service")
	cli.Print("  name   %s\n", buildTargetStyle.Render(cfg.Name))
	cli.Print("  addr   %s\n", buildTargetStyle.Render(cfg.APIAddr))
	cli.Print("  health %s\n", buildTargetStyle.Render(cfg.HealthAddr))

	if err := controller.Install(); err != nil {
		return err
	}

	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service installed")
	return nil
}

func runServiceStart(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	controller, err := newServiceController(cfg, controlServiceProgram{}, nil)
	if err != nil {
		return err
	}
	if err := controller.Start(); err != nil {
		return err
	}

	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service started")
	return nil
}

func runServiceStop(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	controller, err := newServiceController(cfg, controlServiceProgram{}, nil)
	if err != nil {
		return err
	}
	if err := controller.Stop(); err != nil {
		return err
	}

	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service stopped")
	return nil
}

func runServiceUninstall(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	controller, err := newServiceController(cfg, controlServiceProgram{}, nil)
	if err != nil {
		return err
	}
	if err := controller.Uninstall(); err != nil {
		return err
	}

	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), "Service uninstalled")
	return nil
}

func runServiceExport(req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	exported, err := exportBuildService(cfg, req.Format)
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

	cli.Print("%s %s\n", buildSuccessStyle.Render("Done"), outputPath)
	return nil
}

func runServiceRun(ctx context.Context, req serviceRequest) error {
	cfg, err := loadServiceConfig(req)
	if err != nil {
		return err
	}

	controller, err := newServiceController(cfg, &serviceProgram{cfg: cfg}, serviceRunWait(ctx))
	if err != nil {
		return err
	}
	return controller.Run()
}

func loadServiceConfig(req serviceRequest) (buildservice.Config, error) {
	return servicecommon.LoadConfig(req, serviceGetwd, resolveBuildServiceCfg)
}

func applyServiceOverrides(cfg *buildservice.Config, req serviceRequest) error {
	return servicecommon.ApplyOverrides(cfg, req)
}

func newServiceController(cfg buildservice.Config, program nativeservice.Interface, runWait func()) (serviceController, error) {
	serviceConfig := nativeServiceConfig(cfg)
	if runWait != nil {
		serviceConfig.Option["RunWait"] = runWait
	}
	return newBuildNativeController(program, serviceConfig)
}

func nativeServiceConfig(cfg buildservice.Config) *nativeservice.Config {
	cfg = cfg.Normalized()
	return &nativeservice.Config{
		Name:             cfg.Name,
		DisplayName:      cfg.DisplayName,
		Description:      cfg.Description,
		Arguments:        append([]string(nil), cfg.Arguments...),
		Executable:       cfg.Executable,
		WorkingDirectory: cfg.WorkingDirectory,
		Dependencies: []string{
			"After=network-online.target",
			"Wants=network-online.target",
		},
		EnvVars: copyServiceEnv(cfg.Environment),
		Option: nativeservice.KeyValue{
			"KeepAlive": true,
			"RunAtLoad": true,
			"PIDFile":   cfg.PIDFile,
			"Restart":   "on-failure",
		},
	}
}

func serviceRunWait(ctx context.Context) func() {
	if ctx == nil {
		ctx = context.Background()
	}

	return func() {
		sigChan := make(chan os.Signal, 3)
		signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
		defer signal.Stop(sigChan)

		select {
		case <-ctx.Done():
		case <-sigChan:
		}
	}
}

func copyServiceEnv(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func parseServiceCSV(value string) []string {
	return servicecommon.ParseCSV(value)
}
