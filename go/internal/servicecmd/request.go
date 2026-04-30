package servicecmd

import (
	"time"

	"dappco.re/go"
	"dappco.re/go/build/internal/cmdutil"
	buildservice "dappco.re/go/build/pkg/service"
)

// Request bundles the inputs the `core service install/start/stop/status`
// commands need: project layout (Name/ProjectDir/Output/Format), networking
// (APIAddr/HealthAddr/PIDFile), and watcher tuning (WatchPaths/WatchInterval).
//
//	req := servicecmd.Request{Name: "myapp", ProjectDir: ".", APIAddr: ":7300"}
type Request struct {
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

// FromOptions returns a service request decoded from CLI options.
//
// Example:
//
//	req := servicecmd.FromOptions(opts)
//	err := runServiceInstall(req)
func FromOptions(opts core.Options) Request {
	return Request{
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

// LoadConfig resolves, overrides, and normalizes a service config for a request.
//
// Example:
//
//	cfg, err := servicecmd.LoadConfig(req, ax.Getwd, buildservice.ResolveConfig)
//	if err != nil {
//		return err
//	}
func LoadConfig(req Request, getwd func() core.Result, resolve func(string) core.Result) core.Result {
	cwdResult := getwd()
	if !cwdResult.OK {
		return core.Fail(core.E("service.loadServiceConfig", "failed to get working directory", core.NewError(cwdResult.Error())))
	}
	cwd := cwdResult.Value.(string)

	projectDir := req.ProjectDir
	if projectDir == "" {
		projectDir = cwd
	} else if !core.PathIsAbs(projectDir) {
		projectDir = core.PathJoin(cwd, projectDir)
	}

	cfgResult := resolve(projectDir)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(buildservice.Config)

	overridden := ApplyOverrides(&cfg, req)
	if !overridden.OK {
		return overridden
	}
	return core.Ok(cfg.Normalized())
}

// ApplyOverrides applies request-level service overrides to a config.
//
// Example:
//
//	cfg := buildservice.Config{ProjectDir: projectDir}
//	err := servicecmd.ApplyOverrides(&cfg, servicecmd.Request{Name: "core-build"})
func ApplyOverrides(cfg *buildservice.Config, req Request) core.Result {
	if cfg == nil {
		return core.Ok(nil)
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
		if !core.PathIsAbs(cfg.PIDFile) {
			cfg.PIDFile = core.PathJoin(cfg.ProjectDir, cfg.PIDFile)
		}
	}
	if req.WatchPaths != "" {
		cfg.WatchPaths = ParseCSV(req.WatchPaths)
	}
	if req.WatchInterval != "" {
		duration, err := time.ParseDuration(req.WatchInterval)
		if err != nil {
			return core.Fail(core.E("service.applyServiceOverrides", "invalid watch interval", err))
		}
		cfg.WatchInterval = duration
	}
	if req.ScheduleInterval != "" {
		duration, err := time.ParseDuration(req.ScheduleInterval)
		if err != nil {
			return core.Fail(core.E("service.applyServiceOverrides", "invalid schedule interval", err))
		}
		cfg.ScheduleInterval = duration
	}
	if req.AutoRebuildSet {
		cfg.AutoRebuild = req.AutoRebuild
	}

	return core.Ok(nil)
}

// ParseCSV splits comma-separated service option values and drops blank entries.
//
// Example:
//
//	paths := servicecmd.ParseCSV("src, .core/build.yaml")
//	cfg.WatchPaths = paths
func ParseCSV(value string) []string {
	parts := core.Split(value, ",")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		part = core.Trim(part)
		if part == "" {
			continue
		}
		paths = append(paths, part)
	}
	return paths
}
