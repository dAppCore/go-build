package builders

import (
	"context"

	core "dappco.re/go"
)

func TestAppleNotarise_AppleBuilder_Notarise_Good(t *core.T) {
	runner := newRecordingAppleRunner()
	// runExternal returns early on a non-darwin host, so without pinning the
	// host the runner records nothing on Linux, the length assertion fails
	// without stopping, and the next line indexes an empty slice and panics.
	// What this test is about is which command gets composed, not which
	// machine it runs on — WithAppleHostOS exists for exactly that, and
	// apple_realexec_test.go in this package already uses it this way.
	builder := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(runner))

	result := builder.Notarise(context.Background(), "dist/Core.zip", AppleOptions{NotarisationProfile: "core-notary"})
	core.RequireTrue(t, result.OK)
	core.RequireTrue(t, len(runner.calls) == 2)
	core.AssertContains(t, runner.calls[0].Args, "--keychain-profile")
}

func TestAppleNotarise_AppleBuilder_Notarise_Bad(t *core.T) {
	builder := NewAppleBuilder(WithAppleCommandRunner(newRecordingAppleRunner()))
	result := builder.Notarise(context.Background(), "", AppleOptions{})
	core.AssertFalse(t, result.OK)
}

func TestAppleNotarise_AppleBuilder_Notarise_Ugly(t *core.T) {
	runner := newRecordingAppleRunner()
	builder := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(runner))

	result := builder.Notarise(context.Background(), "dist/Core.zip", AppleOptions{APIKeyID: "KEY", APIKeyIssuerID: "ISSUER", APIKeyPath: "AuthKey.p8"})
	core.RequireTrue(t, result.OK)
	core.RequireTrue(t, len(runner.calls) > 0)
	core.AssertContains(t, runner.calls[0].Args, "--issuer")
}
