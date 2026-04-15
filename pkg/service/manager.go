package service

import (
	nativeservice "github.com/kardianos/service"
)

// Manager wraps OS service manager operations.
type Manager interface {
	Install(cfg Config) error
	Start(cfg Config) error
	Stop(cfg Config) error
	Uninstall(cfg Config) error
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

type noopProgram struct{}

func (noopProgram) Start(nativeservice.Service) error { return nil }
func (noopProgram) Stop(nativeservice.Service) error  { return nil }

var newNativeService = func(program nativeservice.Interface, cfg *nativeservice.Config) (nativeController, error) {
	return nativeservice.New(program, cfg)
}

func (m *OSManager) Install(cfg Config) error {
	controller, err := m.serviceFor(cfg)
	if err != nil {
		return err
	}
	return controller.Install()
}

func (m *OSManager) Start(cfg Config) error {
	controller, err := m.serviceFor(cfg)
	if err != nil {
		return err
	}
	return controller.Start()
}

func (m *OSManager) Stop(cfg Config) error {
	controller, err := m.serviceFor(cfg)
	if err != nil {
		return err
	}
	return controller.Stop()
}

func (m *OSManager) Uninstall(cfg Config) error {
	controller, err := m.serviceFor(cfg)
	if err != nil {
		return err
	}
	return controller.Uninstall()
}

func (m *OSManager) serviceFor(cfg Config) (nativeController, error) {
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

	return newNativeService(noopProgram{}, serviceConfig)
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
