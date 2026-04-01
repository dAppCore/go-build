package sdk

import (
	"context"

	"github.com/oasdiff/kin-openapi/openapi3"

	coreerr "dappco.re/go/core/log"
)

// ValidateSpec detects and validates the OpenAPI specification for this SDK.
//
// detectedPath, err := s.ValidateSpec(context.Background())
func (s *SDK) ValidateSpec(ctx context.Context) (string, error) {
	specPath, err := s.DetectSpec()
	if err != nil {
		return "", err
	}

	loader := openapi3.NewLoader()
	loader.Context = ctx
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return "", coreerr.E("sdk.ValidateSpec", "failed to load OpenAPI spec", err)
	}

	if err := doc.Validate(ctx); err != nil {
		return "", coreerr.E("sdk.ValidateSpec", "invalid OpenAPI spec", err)
	}

	return specPath, nil
}
