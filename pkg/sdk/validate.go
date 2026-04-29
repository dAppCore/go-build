package sdk

import (
	"context"

	core "dappco.re/go"
	"github.com/oasdiff/kin-openapi/openapi3"
)

// ValidateSpec detects and validates the OpenAPI specification for this SDK.
//
// detectedPath, err := s.ValidateSpec(context.Background())
func (s *SDK) ValidateSpec(ctx context.Context) core.Result {
	spec := s.DetectSpec()
	if !spec.OK {
		return spec
	}
	specPath := spec.Value.(string)

	loader := openapi3.NewLoader()
	loader.Context = ctx
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return core.Fail(core.E("sdk.ValidateSpec", "failed to load OpenAPI spec", err))
	}

	if err := doc.Validate(ctx); err != nil {
		return core.Fail(core.E("sdk.ValidateSpec", "invalid OpenAPI spec", err))
	}

	return core.Ok(specPath)
}
