package service

import (
	"time"

	core "dappco.re/go"
)

type daemonOptions struct {
	PIDFile         string
	HealthAddr      string
	ShutdownTimeout time.Duration
}

type manageddaemon struct {
	options daemonOptions
	ready   bool
}

func newManagedDaemon(opts daemonOptions) *manageddaemon {
	return &manageddaemon{options: opts}
}

func (d *manageddaemon) Start() core.Result {
	if d == nil || d.options.PIDFile == "" {
		return core.Ok(nil)
	}
	written := core.WriteFile(d.options.PIDFile, []byte("0\n"), 0o644)
	if !written.OK {
		return written
	}
	return core.Ok(nil)
}

func (d *manageddaemon) Stop() core.Result {
	if d == nil || d.options.PIDFile == "" {
		return core.Ok(nil)
	}
	removed := core.Remove(d.options.PIDFile)
	if !removed.OK {
		if err, ok := removed.Value.(error); ok && core.IsNotExist(err) {
			return core.Ok(nil)
		}
		return removed
	}
	return core.Ok(nil)
}

func (d *manageddaemon) SetReady(ready bool) {
	if d != nil {
		d.ready = ready
	}
}
