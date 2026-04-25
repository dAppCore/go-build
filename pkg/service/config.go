package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
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
func ResolveConfig(projectDir string) (Config, error) {
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return Config{}, coreerr.E("service.ResolveConfig", "failed to get working directory", err)
		}
	}

	projectDir = filepath.Clean(projectDir)
	cfg := DefaultConfig(projectDir)

	buildCfg, err := build.LoadConfig(io.Local, projectDir)
	if err != nil {
		return Config{}, coreerr.E("service.ResolveConfig", "failed to load build config", err)
	}
	if buildCfg != nil {
		rawName := firstNonEmpty(buildCfg.Project.Binary, buildCfg.Project.Name, cfg.Name)
		cfg.Name = normaliseServiceName(rawName)
		cfg.DisplayName = displayName(rawName)
		if buildCfg.Project.Description != "" {
			cfg.Description = buildCfg.Project.Description
		}
	}

	return cfg.Normalized(), nil
}

// DefaultConfig returns the default daemon and service manager settings.
func DefaultConfig(projectDir string) Config {
	projectDir = filepath.Clean(projectDir)
	rawName := filepath.Base(projectDir)
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
		if cwd, err := os.Getwd(); err == nil {
			cfg.ProjectDir = cwd
		}
	}
	cfg.ProjectDir = filepath.Clean(cfg.ProjectDir)

	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = cfg.ProjectDir
	}
	cfg.WorkingDirectory = filepath.Clean(cfg.WorkingDirectory)

	if cfg.Name == "" {
		cfg.Name = normaliseServiceName(filepath.Base(cfg.ProjectDir))
	}
	if cfg.DisplayName == "" {
		cfg.DisplayName = displayName(cfg.Name)
	}
	if cfg.Description == "" {
		cfg.Description = "Core build daemon for " + cfg.DisplayName
	}
	if cfg.Executable == "" {
		if executable, err := os.Executable(); err == nil {
			cfg.Executable = executable
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
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfg.ProjectDir, path)
		}
		path = filepath.Clean(path)
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
	setDefaultEnv(env, "CORE_BUILD_WATCH_PATHS", strings.Join(cfg.WatchPaths, ","))
	cfg.Environment = env

	cfg.Arguments = serviceRunArguments(cfg)
	return cfg
}

// ResolveNativeFormat maps an explicit format or the current platform to a native service type.
func ResolveNativeFormat(format string) (NativeFormat, error) {
	format = strings.TrimSpace(strings.ToLower(format))
	switch format {
	case "":
		switch runtime.GOOS {
		case "linux":
			return NativeFormatSystemd, nil
		case "darwin":
			return NativeFormatLaunchd, nil
		case "windows":
			return NativeFormatWindows, nil
		default:
			return "", coreerr.E("service.ResolveNativeFormat", "unsupported platform: "+runtime.GOOS, nil)
		}
	case string(NativeFormatSystemd):
		return NativeFormatSystemd, nil
	case string(NativeFormatLaunchd), "plist":
		return NativeFormatLaunchd, nil
	case string(NativeFormatWindows), "windows-service", "powershell":
		return NativeFormatWindows, nil
	default:
		return "", coreerr.E("service.ResolveNativeFormat", "unsupported native service format: "+format, nil)
	}
}

func defaultPIDFile(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), name+".pid")
	}
	return filepath.Join(home, ".core", "run", name+".pid")
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
		args = append(args, "--watch-paths", strings.Join(cfg.WatchPaths, ","))
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
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normaliseServiceName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "core-build"
	}

	var b strings.Builder
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

	value := strings.Trim(b.String(), "-")
	if value == "" {
		return "core-build"
	}
	return value
}

func displayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Core Build"
	}

	fields := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	if len(fields) == 0 {
		return "Core Build"
	}

	for i, field := range fields {
		runes := []rune(strings.ToLower(field))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		fields[i] = string(runes)
	}

	return strings.Join(fields, " ")
}
