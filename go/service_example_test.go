// SPDX-License-Identifier: EUPL-1.2

package build

import (
	core "dappco.re/go"
)

func ExampleNewService() {
	// NewService returns a factory; call it with a Core to build the *Service.
	factory := NewService(ServiceOptions{})
	_ = factory(core.New())
}

func ExampleRegister() {
	// Register wires the build service into a Core with default options.
	_ = Register(core.New())
}
