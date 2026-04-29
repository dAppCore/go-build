package service

import (
	core "dappco.re/go"
	nativeservice "github.com/kardianos/service"
)

// Manager wraps OS service manager operations.
type Manager interface {
	Install(cfg Config) core.Result
	Start(cfg Config) core.Result
	Stop(cfg Config) core.Result
	Uninstall(cfg Config) core.Result
}

// NewManager returns the default OS service manager implementation.
func NewManager() Manager {
	return &OSManager{}
}

// OSManager uses github.com/kardianos/service to control native services.
type OSManager struct{}

type nativeController interface {
	Install() error
	Start() error
	Stop() error
	Uninstall() error
}

var newNativeService = func(program nativeservice.Interface, cfg *nativeservice.Config) (nativeController, error) {
	return nativeservice.New(program, cfg)
}

func (m *OSManager) Install(cfg Config) core.Result {
	controller := m.serviceFor(cfg)
	if !controller.OK {
		return controller
	}
	return core.ResultOf(nil, controller.Value.(nativeController).Install())
}

func (m *OSManager) Start(cfg Config) core.Result {
	controller := m.serviceFor(cfg)
	if !controller.OK {
		return controller
	}
	return core.ResultOf(nil, controller.Value.(nativeController).Start())
}

func (m *OSManager) Stop(cfg Config) core.Result {
	controller := m.serviceFor(cfg)
	if !controller.OK {
		return controller
	}
	return core.ResultOf(nil, controller.Value.(nativeController).Stop())
}

func (m *OSManager) Uninstall(cfg Config) core.Result {
	controller := m.serviceFor(cfg)
	if !controller.OK {
		return controller
	}
	return core.ResultOf(nil, controller.Value.(nativeController).Uninstall())
}

func (m *OSManager) serviceFor(cfg Config) core.Result {
	cfg = cfg.Normalized()

	serviceConfig := &nativeservice.Config{
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
		EnvVars: copyEnv(cfg.Environment),
		Option: nativeservice.KeyValue{
			"KeepAlive": true,
			"RunAtLoad": true,
		},
	}

	return core.ResultOf(newNativeService(nil, serviceConfig))
}

func copyEnv(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
