// SPDX-License-Identifier: EUPL-1.2

package build

import (
	core "github.com/dappcore/go"
)

func ExampleService() {
	// The root build Service holds the live orchestrator Manager.
	svc := NewService(ServiceOptions{})(core.New()).Value.(*Service)
	_ = svc.Manager
}
