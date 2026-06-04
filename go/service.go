// SPDX-License-Identifier: EUPL-1.2

// Service registration for the root build package — exposes the
// canonical NewService(opts) + Register(c) shape per Mantis #1336.
// See build.go for the package doc + Service struct definition.
//
// This file holds the constructor surface so the canonical naming
// (NewService / Register / ServiceOptions) lives in the file consumers
// expect to find it.

package build

import (
	core "dappco.re/go"
	buildservice "dappco.re/go/build/pkg/service"
)

// ServiceOptions configures the root build service. v1 has no fields —
// the underlying buildservice.Manager + servicecmd command tree are
// configured via the Core's standard config layer (resolved by
// buildservice.ResolveConfig at command-execution time, not at
// service-registration time). Future fields (e.g. ProjectRoot override,
// disable specific subpackages) land here as needed.
type ServiceOptions struct{}

// NewService returns a factory that constructs the root build *Service
// holding a live buildservice.Manager and registers it under "build"
// via core.WithService.
//
// Usage example:
//
//	core.WithService(build.NewService(build.ServiceOptions{}))
//
// The Manager is always wired (no credentials needed) so consumers can
// reach it immediately via core.MustServiceFor[*build.Service](c, "build").Manager.
//
// Note: this does NOT register the `core service` command tree — that's
// servicecmd.AddServiceCommands(c)'s job and stays an explicit caller
// responsibility (the build CLI has multiple cmd subdirs each with its
// own AddXxxCommands, registered by the cmd binary, not the library).
func NewService(opts ServiceOptions) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		return core.Ok(&Service{
			ServiceRuntime: core.NewServiceRuntime(c, opts),
			Manager:        buildservice.NewManager(),
		})
	}
}

// Register wires the root build service into the Core with default
// ServiceOptions — the imperative-style alternative to NewService.
//
//	c := core.New()
//	if r := build.Register(c); !r.OK { return r }
func Register(c *core.Core) core.Result {
	return NewService(ServiceOptions{})(c)
}
