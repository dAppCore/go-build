package service

import (
	"runtime"
	"strconv"
	"time"
	"unicode"

	core "dappco.re/go"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

const (
	// DefaultAPIAddr is the default HTTP listen address for the build API.
	DefaultAPIAddr = "127.0.0.1:9101"
	// DefaultHealthAddr is the default health probe address for daemon mode.
	DefaultHealthAddr = "127.0.0.1:9102"
	// DefaultWatchInterval is the file polling interval for auto-rebuilds.
	DefaultWatchInterval = 2 * time.Second
	// DefaultScheduleInterval is the cadence for periodic daemon jobs.
	DefaultScheduleInterval = time.Minute
)

// NativeFormat identifies the native service definition to generate.
type NativeFormat string

const (
	NativeFormatSystemd NativeFormat = "systemd"
	NativeFormatLaunchd NativeFormat = "launchd"
	NativeFormatWindows NativeFormat = "windows"
)

// Config describes the installed background service and daemon runtime.
type Config struct {
	Name             string
	DisplayName      string
	Description      string
	ProjectDir       string
	WorkingDirectory string
	Executable       string
	Arguments        []string
	Environment      map[string]string
	PIDFile          string
	APIAddr          string
	HealthAddr       string
	WatchPaths       []string
	WatchInterval    time.Duration
	ScheduleInterval time.Duration
	AutoRebuild      bool
}

// ExportedConfig is a generated native service definition.
type ExportedConfig struct {
	Format   NativeFormat
	Filename string
	Content  string
}

// ResolveConfig loads service defaults for the project in projectDir.
func ResolveConfig(projectDir string) core.Result {
	if projectDir == "" {
		wd := core.Getwd()
		if !wd.OK {
			return core.Fail(core.E("service.ResolveConfig", "failed to get working directory", core.NewError(wd.Error())))
		}
		projectDir = wd.Value.(string)
	}

	projectDir = core.PathJoin(projectDir)
	cfg := DefaultConfig(projectDir)

	loaded := build.LoadConfig(io.Local, projectDir)
	if !loaded.OK {
		return core.Fail(core.E("service.ResolveConfig", "failed to load build config", core.NewError(loaded.Error())))
	}
	buildCfg := loaded.Value.(*build.BuildConfig)
	if buildCfg != nil {
		rawName := firstNonEmpty(buildCfg.Project.Binary, buildCfg.Project.Name, cfg.Name)
		cfg.Name = normaliseServiceName(rawName)
		cfg.DisplayName = displayName(rawName)
		if buildCfg.Project.Description != "" {
			cfg.Description = buildCfg.Project.Description
		}
	}

	return core.Ok(cfg.Normalized())
}

// DefaultConfig returns the default daemon and service manager settings.
func DefaultConfig(projectDir string) Config {
	projectDir = core.PathJoin(projectDir)
	rawName := core.PathBase(projectDir)
	name := normaliseServiceName(rawName)

	return Config{
		Name:             name,
		DisplayName:      displayName(rawName),
		Description:      "Core build daemon for " + displayName(rawName),
		ProjectDir:       projectDir,
		WorkingDirectory: projectDir,
		APIAddr:          DefaultAPIAddr,
		HealthAddr:       DefaultHealthAddr,
		WatchPaths:       []string{projectDir},
		WatchInterval:    DefaultWatchInterval,
		ScheduleInterval: DefaultScheduleInterval,
		AutoRebuild:      true,
	}
}

// Normalized returns a copy of cfg with defaults, environment, and arguments applied.
func (cfg Config) Normalized() Config {
	if cfg.ProjectDir == "" {
		if cwd := core.Getwd(); cwd.OK {
			cfg.ProjectDir = cwd.Value.(string)
		}
	}
	cfg.ProjectDir = core.PathJoin(cfg.ProjectDir)

	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = cfg.ProjectDir
	}
	cfg.WorkingDirectory = core.PathJoin(cfg.WorkingDirectory)

	if cfg.Name == "" {
		cfg.Name = normaliseServiceName(core.PathBase(cfg.ProjectDir))
	}
	if cfg.DisplayName == "" {
		cfg.DisplayName = displayName(cfg.Name)
	}
	if cfg.Description == "" {
		cfg.Description = "Core build daemon for " + cfg.DisplayName
	}
	if cfg.Executable == "" {
		args := core.Args()
		if len(args) > 0 {
			executable := core.PathAbs(args[0])
			if executable.OK {
				cfg.Executable = executable.Value.(string)
			} else {
				cfg.Executable = args[0]
			}
		}
	}
	if cfg.APIAddr == "" {
		cfg.APIAddr = DefaultAPIAddr
	}
	if cfg.HealthAddr == "" {
		cfg.HealthAddr = DefaultHealthAddr
	}
	if cfg.WatchInterval <= 0 {
		cfg.WatchInterval = DefaultWatchInterval
	}
	if cfg.ScheduleInterval <= 0 {
		cfg.ScheduleInterval = DefaultScheduleInterval
	}
	if len(cfg.WatchPaths) == 0 {
		cfg.WatchPaths = []string{cfg.ProjectDir}
	}

	watchPaths := make([]string, 0, len(cfg.WatchPaths))
	seen := make(map[string]struct{})
	for _, path := range cfg.WatchPaths {
		if path == "" {
			continue
		}
		if !core.PathIsAbs(path) {
			path = core.PathJoin(cfg.ProjectDir, path)
		}
		path = core.PathJoin(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		watchPaths = append(watchPaths, path)
	}
	if len(watchPaths) == 0 {
		watchPaths = []string{cfg.ProjectDir}
	}
	cfg.WatchPaths = watchPaths

	if cfg.PIDFile == "" {
		cfg.PIDFile = defaultPIDFile(cfg.Name)
	}

	env := make(map[string]string, len(cfg.Environment)+6)
	for key, value := range cfg.Environment {
		env[key] = value
	}
	setDefaultEnv(env, "CORE_BUILD_SERVICE", "1")
	setDefaultEnv(env, "CORE_BUILD_PROJECT_DIR", cfg.ProjectDir)
	setDefaultEnv(env, "CORE_BUILD_API_ADDR", cfg.APIAddr)
	setDefaultEnv(env, "CORE_BUILD_HEALTH_ADDR", cfg.HealthAddr)
	setDefaultEnv(env, "CORE_BUILD_PID_FILE", cfg.PIDFile)
	setDefaultEnv(env, "CORE_BUILD_WATCH_PATHS", core.Join(",", cfg.WatchPaths...))
	cfg.Environment = env

	cfg.Arguments = serviceRunArguments(cfg)
	return cfg
}

// ResolveNativeFormat maps an explicit format or the current platform to a native service type.
func ResolveNativeFormat(format string) core.Result {
	format = core.Trim(core.Lower(format))
	switch format {
	case "":
		switch runtime.GOOS {
		case "linux":
			return core.Ok(NativeFormatSystemd)
		case "darwin":
			return core.Ok(NativeFormatLaunchd)
		case "windows":
			return core.Ok(NativeFormatWindows)
		default:
			return core.Fail(core.E("service.ResolveNativeFormat", "unsupported platform: "+runtime.GOOS, nil))
		}
	case string(NativeFormatSystemd):
		return core.Ok(NativeFormatSystemd)
	case string(NativeFormatLaunchd), "plist":
		return core.Ok(NativeFormatLaunchd)
	case string(NativeFormatWindows), "windows-service", "powershell":
		return core.Ok(NativeFormatWindows)
	default:
		return core.Fail(core.E("service.ResolveNativeFormat", "unsupported native service format: "+format, nil))
	}
}

func defaultPIDFile(name string) string {
	home := core.UserHomeDir()
	if !home.OK {
		return core.PathJoin(core.TempDir(), name+".pid")
	}
	return core.PathJoin(home.Value.(string), ".core", "run", name+".pid")
}

func serviceRunArguments(cfg Config) []string {
	args := []string{
		"service",
		"run",
		"--project-dir", cfg.ProjectDir,
		"--name", cfg.Name,
		"--addr", cfg.APIAddr,
		"--health-addr", cfg.HealthAddr,
		"--pid-file", cfg.PIDFile,
		"--auto-rebuild", strconv.FormatBool(cfg.AutoRebuild),
		"--watch-interval", cfg.WatchInterval.String(),
		"--schedule-interval", cfg.ScheduleInterval.String(),
	}
	if len(cfg.WatchPaths) > 0 {
		args = append(args, "--watch-paths", core.Join(",", cfg.WatchPaths...))
	}
	return args
}

func setDefaultEnv(env map[string]string, key, value string) {
	if _, ok := env[key]; ok {
		return
	}
	env[key] = value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = core.Trim(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normaliseServiceName(name string) string {
	name = core.Trim(core.Lower(name))
	if name == "" {
		return "core-build"
	}

	b := core.NewBuilder()
	lastHyphen := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}

	value := trimHyphens(b.String())
	if value == "" {
		return "core-build"
	}
	return value
}

func displayName(name string) string {
	name = core.Trim(name)
	if name == "" {
		return "Core Build"
	}

	fields := serviceNameFields(name)
	if len(fields) == 0 {
		return "Core Build"
	}

	for i, field := range fields {
		runes := []rune(core.Lower(field))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		fields[i] = string(runes)
	}

	return core.Join(" ", fields...)
}

func serviceNameFields(name string) []string {
	var fields []string
	start := -1
	for i, r := range name {
		if r == '-' || r == '_' || unicode.IsSpace(r) {
			if start >= 0 {
				fields = append(fields, name[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fields = append(fields, name[start:])
	}
	return fields
}

func trimHyphens(value string) string {
	for core.HasPrefix(value, "-") {
		value = core.TrimPrefix(value, "-")
	}
	for core.HasSuffix(value, "-") {
		value = core.TrimSuffix(value, "-")
	}
	return value
}
