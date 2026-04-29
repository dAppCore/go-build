package service

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleExport() {
	_, _ = Export(Config{}, "tar.gz")
	core.Println("Export")
	// Output: Export
}
