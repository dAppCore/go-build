package builders

import (
	"context"

	core "dappco.re/go"
)

func TestAppleNotarise_AppleBuilder_Notarise_Good(t *core.T) {
	runner := &recordingAppleRunner{}
	builder := NewAppleBuilder(WithAppleCommandRunner(runner))

	err := builder.Notarise(context.Background(), "dist/Core.zip", AppleOptions{NotarisationProfile: "core-notary"})
	core.RequireNoError(t, err)
	core.AssertLen(t, runner.calls, 2)
	core.AssertContains(t, runner.calls[0].Args, "--keychain-profile")
}

func TestAppleNotarise_AppleBuilder_Notarise_Bad(t *core.T) {
	builder := NewAppleBuilder(WithAppleCommandRunner(&recordingAppleRunner{}))
	err := builder.Notarise(context.Background(), "", AppleOptions{})
	core.AssertError(t, err)
}

func TestAppleNotarise_AppleBuilder_Notarise_Ugly(t *core.T) {
	runner := &recordingAppleRunner{}
	builder := NewAppleBuilder(WithAppleCommandRunner(runner))

	err := builder.Notarise(context.Background(), "dist/Core.zip", AppleOptions{APIKeyID: "KEY", APIKeyIssuerID: "ISSUER", APIKeyPath: "AuthKey.p8"})
	core.RequireNoError(t, err)
	core.AssertContains(t, runner.calls[0].Args, "--issuer")
}
