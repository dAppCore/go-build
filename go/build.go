// SPDX-License-Identifier: EUPL-1.2

// Package build is the root entry point for the go-build orchestration
// surface — composes pkg/{api, events, release, sdk, service, storage}
// into one Core-registerable Service per Mantis #1336.
//
// The subpackages are layers of one product (the dev/build orchestrator),
// not unrelated domains — Athena's #1336 adjudication 2026-05-10 placed
// go-build in the "Option A: lift root composer" cohort with 90%
// confidence. This file is the root composer; service.go holds the
// canonical NewService + Register surface.
//
//	c, _ := core.New(
//	    core.WithService(build.NewService(build.ServiceOptions{})),
//	)
//	svc := core.MustServiceFor[*build.Service](c, "build")
//	mgr := svc.Manager  // == buildservice.NewManager() result
package build

import (
	core "dappco.re/go"
	buildservice "dappco.re/go/build/pkg/service"
)

// Service is the root build service handle — composes the existing
// pkg/service.Manager (the de-facto orchestrator surface from
// pkg/service/manager.go) under a Core-registerable identity.
//
// Usage example: `svc := core.MustServiceFor[*build.Service](c, "build"); _ = svc.Manager`
type Service struct {
	*core.ServiceRuntime[ServiceOptions]
	// Manager is the live build orchestrator. Always non-nil — constructed
	// via buildservice.NewManager() during NewService.
	Manager buildservice.Manager
}
