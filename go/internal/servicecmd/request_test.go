package servicecmd

import (
	"time"

	core "dappco.re/go"
	buildservice "dappco.re/go/build/pkg/service"
)

// okGetwd returns a getwd stub that always resolves to dir.
func okGetwd(dir string) func() core.Result {
	return func() core.Result { return core.Ok(dir) }
}

// resolveTo returns a resolve stub that yields cfg for whatever project dir it
// is given, recording the directory it was asked to resolve into *seen.
func resolveTo(cfg buildservice.Config, seen *string) func(string) core.Result {
	return func(dir string) core.Result {
		if seen != nil {
			*seen = dir
		}
		return core.Ok(cfg)
	}
}

// --- FromOptions: decode CLI options into a Request ---

func TestRequest_FromOptions_Good(t *core.T) {
	req := FromOptions(core.NewOptions(
		core.Option{Key: "name", Value: "myapp"},
		core.Option{Key: "display-name", Value: "My App"},
		core.Option{Key: "description", Value: "the daemon"},
		core.Option{Key: "project-dir", Value: "/srv/app"},
		core.Option{Key: "output", Value: "dist/app.service"},
		core.Option{Key: "format", Value: "systemd"},
		core.Option{Key: "addr", Value: ":7300"},
		core.Option{Key: "health-addr", Value: ":7301"},
		core.Option{Key: "pid-file", Value: "run/app.pid"},
		core.Option{Key: "watch-paths", Value: "src,docs"},
		core.Option{Key: "watch-interval", Value: "5s"},
		core.Option{Key: "schedule-interval", Value: "1m"},
		core.Option{Key: "auto-rebuild", Value: false},
	))

	core.AssertEqual(t, "myapp", req.Name)
	core.AssertEqual(t, "My App", req.DisplayName)
	core.AssertEqual(t, "the daemon", req.Description)
	core.AssertEqual(t, "/srv/app", req.ProjectDir)
	core.AssertEqual(t, "dist/app.service", req.Output)
	core.AssertEqual(t, "systemd", req.Format)
	core.AssertEqual(t, ":7300", req.APIAddr)
	core.AssertEqual(t, ":7301", req.HealthAddr)
	core.AssertEqual(t, "run/app.pid", req.PIDFile)
	core.AssertEqual(t, "src,docs", req.WatchPaths)
	core.AssertEqual(t, "5s", req.WatchInterval)
	core.AssertEqual(t, "1m", req.ScheduleInterval)
	// An explicit auto-rebuild=false is captured and marked as set.
	core.AssertFalse(t, req.AutoRebuild)
	core.AssertTrue(t, req.AutoRebuildSet)
}

func TestRequest_FromOptions_Bad(t *core.T) {
	// Empty options: all string fields blank; auto-rebuild defaults to true but
	// is NOT marked set, so the override layer leaves the resolved value alone.
	req := FromOptions(core.NewOptions())

	core.AssertEqual(t, "", req.Name)
	core.AssertEqual(t, "", req.ProjectDir)
	core.AssertEqual(t, "", req.APIAddr)
	core.AssertEqual(t, "", req.WatchPaths)
	core.AssertEqual(t, "", req.WatchInterval)
	core.AssertTrue(t, req.AutoRebuild)
	core.AssertFalse(t, req.AutoRebuildSet)
}

func TestRequest_FromOptions_Ugly(t *core.T) {
	// Edge case: snake_case aliases resolve identically to the hyphenated
	// forms, and an explicit auto_rebuild=true is recorded as set.
	req := FromOptions(core.NewOptions(
		core.Option{Key: "display_name", Value: "Alias Display"},
		core.Option{Key: "project_dir", Value: "/alias/dir"},
		core.Option{Key: "api_addr", Value: ":9000"},
		core.Option{Key: "health_addr", Value: ":9001"},
		core.Option{Key: "pid_file", Value: "/var/run/app.pid"},
		core.Option{Key: "watch_paths", Value: "internal"},
		core.Option{Key: "watch_interval", Value: "10s"},
		core.Option{Key: "schedule_interval", Value: "30s"},
		core.Option{Key: "auto_rebuild", Value: true},
	))

	core.AssertEqual(t, "Alias Display", req.DisplayName)
	core.AssertEqual(t, "/alias/dir", req.ProjectDir)
	core.AssertEqual(t, ":9000", req.APIAddr)
	core.AssertEqual(t, ":9001", req.HealthAddr)
	core.AssertEqual(t, "/var/run/app.pid", req.PIDFile)
	core.AssertEqual(t, "internal", req.WatchPaths)
	core.AssertEqual(t, "10s", req.WatchInterval)
	core.AssertEqual(t, "30s", req.ScheduleInterval)
	core.AssertTrue(t, req.AutoRebuild)
	core.AssertTrue(t, req.AutoRebuildSet)
}

// TestRequest_FromOptions_AddrPrecedence verifies the primary key wins over its
// aliases for the API address (addr > api-addr > api_addr).
func TestRequest_FromOptions_AddrPrecedence(t *core.T) {
	req := FromOptions(core.NewOptions(
		core.Option{Key: "api_addr", Value: ":1111"},
		core.Option{Key: "addr", Value: ":2222"},
	))
	core.AssertEqual(t, ":2222", req.APIAddr)
}

// --- LoadConfig: resolve + override + normalise ---

func TestRequest_LoadConfig_Good(t *core.T) {
	// Happy path: cwd resolves, config resolves, overrides apply, and the result
	// is the Normalized() config (defaults filled, env populated).
	var seenDir string
	result := LoadConfig(
		Request{Name: "core-build"},
		okGetwd("/work"),
		resolveTo(buildservice.Config{ProjectDir: "/work"}, &seenDir),
	)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "/work", seenDir)

	cfg := result.Value.(buildservice.Config)
	core.AssertEqual(t, "core-build", cfg.Name)
	// Normalized() fills the default watch interval and the service env vars.
	core.AssertEqual(t, buildservice.DefaultWatchInterval, cfg.WatchInterval)
	core.AssertNotEmpty(t, cfg.Environment)
}

func TestRequest_LoadConfig_Bad(t *core.T) {
	// Failure path: the working-directory lookup fails and the error is wrapped
	// before any resolution is attempted.
	resolveCalled := false
	result := LoadConfig(
		Request{},
		func() core.Result { return core.Fail(core.NewError("no-cwd")) },
		func(string) core.Result {
			resolveCalled = true
			return core.Ok(buildservice.Config{})
		},
	)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to get working directory")
	core.AssertFalse(t, resolveCalled)
}

func TestRequest_LoadConfig_Ugly(t *core.T) {
	// Edge case: a relative project dir is joined onto the resolved cwd before
	// being handed to the resolver.
	var seenDir string
	result := LoadConfig(
		Request{ProjectDir: "sub/dir"},
		okGetwd("/work"),
		resolveTo(buildservice.Config{ProjectDir: "/work/sub/dir"}, &seenDir),
	)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, core.PathJoin("/work", "sub/dir"), seenDir)
}

// TestRequest_LoadConfig_AbsoluteProjectDir confirms an absolute project dir is
// passed through unchanged (not re-joined onto the cwd).
func TestRequest_LoadConfig_AbsoluteProjectDir(t *core.T) {
	var seenDir string
	result := LoadConfig(
		Request{ProjectDir: "/abs/path"},
		okGetwd("/work"),
		resolveTo(buildservice.Config{ProjectDir: "/abs/path"}, &seenDir),
	)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "/abs/path", seenDir)
}

// TestRequest_LoadConfig_ResolveError bubbles the resolver's failure unchanged.
func TestRequest_LoadConfig_ResolveError(t *core.T) {
	result := LoadConfig(
		Request{},
		okGetwd("/work"),
		func(string) core.Result { return core.Fail(core.NewError("resolve-failed")) },
	)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "resolve-failed")
}

// TestRequest_LoadConfig_OverrideError surfaces a bad request override (an
// invalid duration) through LoadConfig.
func TestRequest_LoadConfig_OverrideError(t *core.T) {
	result := LoadConfig(
		Request{WatchInterval: "not-a-duration"},
		okGetwd("/work"),
		resolveTo(buildservice.Config{ProjectDir: "/work"}, nil),
	)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "invalid watch interval")
}

// --- ApplyOverrides: request-level config overrides ---

func TestRequest_ApplyOverrides_Good(t *core.T) {
	// Every populated request field overrides the corresponding config field.
	cfg := buildservice.Config{ProjectDir: "/proj"}
	result := ApplyOverrides(&cfg, Request{
		Name:             "renamed",
		DisplayName:      "Renamed Service",
		Description:      "overridden",
		APIAddr:          ":8000",
		HealthAddr:       ":8001",
		WatchPaths:       "a, b ,,c",
		WatchInterval:    "15s",
		ScheduleInterval: "2m",
		AutoRebuild:      false,
		AutoRebuildSet:   true,
	})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "renamed", cfg.Name)
	core.AssertEqual(t, "Renamed Service", cfg.DisplayName)
	core.AssertEqual(t, "overridden", cfg.Description)
	core.AssertEqual(t, ":8000", cfg.APIAddr)
	core.AssertEqual(t, ":8001", cfg.HealthAddr)
	// WatchPaths is parsed as CSV with blanks dropped.
	core.AssertEqual(t, []string{"a", "b", "c"}, cfg.WatchPaths)
	core.AssertEqual(t, 15*time.Second, cfg.WatchInterval)
	core.AssertEqual(t, 2*time.Minute, cfg.ScheduleInterval)
	// AutoRebuildSet=true means the explicit false is applied.
	core.AssertFalse(t, cfg.AutoRebuild)
}

func TestRequest_ApplyOverrides_Bad(t *core.T) {
	// Failure path: an unparseable watch interval is reported as an error and no
	// later fields are reached.
	cfg := buildservice.Config{ProjectDir: "/proj"}
	result := ApplyOverrides(&cfg, Request{WatchInterval: "nope", ScheduleInterval: "5s"})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "invalid watch interval")
	// ScheduleInterval (parsed after WatchInterval) must remain unset.
	core.AssertEqual(t, time.Duration(0), cfg.ScheduleInterval)
}

func TestRequest_ApplyOverrides_Ugly(t *core.T) {
	// Edge case: a nil config pointer is tolerated and returns OK without panic.
	result := ApplyOverrides(nil, Request{Name: "ignored"})
	core.AssertTrue(t, result.OK)
}

// TestRequest_ApplyOverrides_RelativePIDFile joins a relative PID file onto the
// config's project dir; an absolute PID file is left as-is.
func TestRequest_ApplyOverrides_RelativePIDFile(t *core.T) {
	relative := buildservice.Config{ProjectDir: "/proj"}
	core.AssertTrue(t, ApplyOverrides(&relative, Request{PIDFile: "run/app.pid"}).OK)
	core.AssertEqual(t, core.PathJoin("/proj", "run/app.pid"), relative.PIDFile)

	absolute := buildservice.Config{ProjectDir: "/proj"}
	core.AssertTrue(t, ApplyOverrides(&absolute, Request{PIDFile: "/var/run/app.pid"}).OK)
	core.AssertEqual(t, "/var/run/app.pid", absolute.PIDFile)
}

// TestRequest_ApplyOverrides_ScheduleIntervalError covers the second duration
// parser branch independently of the watch interval.
func TestRequest_ApplyOverrides_ScheduleIntervalError(t *core.T) {
	cfg := buildservice.Config{ProjectDir: "/proj"}
	result := ApplyOverrides(&cfg, Request{ScheduleInterval: "definitely-not-a-duration"})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "invalid schedule interval")
}

// TestRequest_ApplyOverrides_EmptyRequestPreservesConfig confirms an empty
// request leaves an existing config untouched.
func TestRequest_ApplyOverrides_EmptyRequestPreservesConfig(t *core.T) {
	cfg := buildservice.Config{Name: "original", APIAddr: ":1234", ProjectDir: "/proj"}
	result := ApplyOverrides(&cfg, Request{})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "original", cfg.Name)
	core.AssertEqual(t, ":1234", cfg.APIAddr)
}

// --- ParseCSV: comma-separated option parsing ---

func TestRequest_ParseCSV_Good(t *core.T) {
	core.AssertEqual(t, []string{"src", "docs", "internal"}, ParseCSV("src,docs,internal"))
}

func TestRequest_ParseCSV_Bad(t *core.T) {
	// An empty string yields an empty (non-nil) slice — no blank entries.
	result := ParseCSV("")
	core.AssertEqual(t, 0, len(result))
}

func TestRequest_ParseCSV_Ugly(t *core.T) {
	// Edge case: surrounding whitespace is trimmed and blank fields (from
	// leading/trailing/double commas) are dropped.
	core.AssertEqual(t, []string{"a", "b", "c"}, ParseCSV("  a , ,b,  , c ,"))
}
