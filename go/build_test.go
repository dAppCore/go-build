// SPDX-License-Identifier: EUPL-1.2

package build

import (
	core "dappco.re/go"
)

func TestBuild_Service_Good(t *core.T) {
	svc := NewService(ServiceOptions{})(core.New()).Value.(*Service)
	core.AssertNotNil(t, svc)
	core.AssertNotNil(t, svc.Manager)
	core.AssertNotNil(t, svc.ServiceRuntime)
}

func TestBuild_Service_Bad(t *core.T) {
	// The embedded runtime exposes the constructing Core — the Service is
	// never detached from its owner.
	c := core.New()
	svc := NewService(ServiceOptions{})(c).Value.(*Service)
	core.AssertTrue(t, c == svc.Core())
}

func TestBuild_Service_Ugly(t *core.T) {
	// Distinct constructions hold distinct, independently-live Managers.
	a := NewService(ServiceOptions{})(core.New()).Value.(*Service)
	b := NewService(ServiceOptions{})(core.New()).Value.(*Service)
	core.AssertFalse(t, a == b)
	core.AssertNotNil(t, a.Manager)
	core.AssertNotNil(t, b.Manager)
}
