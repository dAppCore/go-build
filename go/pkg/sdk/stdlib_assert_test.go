package sdk

import "dappco.re/go/build/internal/testassert"

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertContains      = testassert.Contains
)
