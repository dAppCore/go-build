// SPDX-License-Identifier: EUPL-1.2

package build

import (
	core "dappco.re/go"
)

func TestService_NewService_Good(t *core.T) {
	factory := NewService(ServiceOptions{})
	core.AssertNotNil(t, factory)

	result := factory(core.New())
	core.AssertTrue(t, result.OK)

	svc, ok := result.Value.(*Service)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, svc.Manager)
}

func TestService_NewService_Bad(t *core.T) {
	// NewService has no failure path (no credentials needed), so the "bad"
	// axis exercises independence: each factory call must build its own
	// Manager, never a shared global a second registration could clobber.
	first := NewService(ServiceOptions{})(core.New())
	second := NewService(ServiceOptions{})(core.New())
	core.AssertTrue(t, first.OK)
	core.AssertTrue(t, second.OK)
	core.AssertFalse(t, first.Value.(*Service) == second.Value.(*Service))
}

func TestService_NewService_Ugly(t *core.T) {
	// The constructed Service is bound to the exact Core that built it.
	c := core.New()
	svc := NewService(ServiceOptions{})(c).Value.(*Service)
	core.AssertNotNil(t, svc.ServiceRuntime)
	core.AssertTrue(t, c == svc.Core())
}

func TestService_Register_Good(t *core.T) {
	result := Register(core.New())
	core.AssertTrue(t, result.OK)

	svc, ok := result.Value.(*Service)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, svc.Manager)
}

func TestService_Register_Bad(t *core.T) {
	// Register is the imperative shorthand for NewService(ServiceOptions{});
	// both routes must yield a Service carrying a live Manager.
	viaRegister := Register(core.New())
	viaFactory := NewService(ServiceOptions{})(core.New())
	core.AssertNotNil(t, viaRegister.Value.(*Service).Manager)
	core.AssertNotNil(t, viaFactory.Value.(*Service).Manager)
}

func TestService_Register_Ugly(t *core.T) {
	// Register binds the service runtime to the supplied Core.
	c := core.New()
	svc := Register(c).Value.(*Service)
	core.AssertNotNil(t, svc.ServiceRuntime)
	core.AssertTrue(t, c == svc.Core())
}
