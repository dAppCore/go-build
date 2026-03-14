// SPDX-Licence-Identifier: EUPL-1.2

package api

import "embed"

// Assets holds the built UI bundle (core-build.js and related files).
// The directory is populated by running `npm run build` in the ui/ directory.
//
//go:embed all:ui/dist
var Assets embed.FS
