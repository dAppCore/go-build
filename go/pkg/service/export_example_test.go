package service

import (
	core "github.com/dappcore/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleExport() {
	_ = Export(Config{}, "tar.gz")
	core.Println("Export")
	// Output: Export
}
