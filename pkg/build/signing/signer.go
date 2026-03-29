// Package signing provides code signing for build artifacts.
package signing

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/io"
)

// Signer defines the interface for code signing implementations.
// Usage example: declare a value of type signing.Signer in integrating code.
type Signer interface {
	// Name returns the signer's identifier.
	Name() string
	// Available checks if this signer can be used.
	Available() bool
	// Sign signs the artifact at the given path.
	Sign(ctx context.Context, fs io.Medium, path string) error
}

// SignConfig holds signing configuration from .core/build.yaml.
// Usage example: declare a value of type signing.SignConfig in integrating code.
type SignConfig struct {
	Enabled bool          `yaml:"enabled"`
	GPG     GPGConfig     `yaml:"gpg,omitempty"`
	MacOS   MacOSConfig   `yaml:"macos,omitempty"`
	Windows WindowsConfig `yaml:"windows,omitempty"`
}

// GPGConfig holds GPG signing configuration.
// Usage example: declare a value of type signing.GPGConfig in integrating code.
type GPGConfig struct {
	Key string `yaml:"key"` // Key ID or fingerprint, supports $ENV
}

// MacOSConfig holds macOS codesign configuration.
// Usage example: declare a value of type signing.MacOSConfig in integrating code.
type MacOSConfig struct {
	Identity    string `yaml:"identity"`     // Developer ID Application: ...
	Notarize    bool   `yaml:"notarize"`     // Submit to Apple for notarization
	AppleID     string `yaml:"apple_id"`     // Apple account email
	TeamID      string `yaml:"team_id"`      // Team ID
	AppPassword string `yaml:"app_password"` // App-specific password
}

// WindowsConfig holds Windows signtool configuration (placeholder).
// Usage example: declare a value of type signing.WindowsConfig in integrating code.
type WindowsConfig struct {
	Certificate string `yaml:"certificate"` // Path to .pfx
	Password    string `yaml:"password"`    // Certificate password
}

// DefaultSignConfig returns sensible defaults.
// Usage example: call signing.DefaultSignConfig(...) from integrating code.
func DefaultSignConfig() SignConfig {
	return SignConfig{
		Enabled: true,
		GPG: GPGConfig{
			Key: core.Env("GPG_KEY_ID"),
		},
		MacOS: MacOSConfig{
			Identity:    core.Env("CODESIGN_IDENTITY"),
			AppleID:     core.Env("APPLE_ID"),
			TeamID:      core.Env("APPLE_TEAM_ID"),
			AppPassword: core.Env("APPLE_APP_PASSWORD"),
		},
	}
}

// ExpandEnv expands environment variables in config values.
// Usage example: call value.ExpandEnv(...) from integrating code.
func (c *SignConfig) ExpandEnv() {
	c.GPG.Key = expandEnv(c.GPG.Key)
	c.MacOS.Identity = expandEnv(c.MacOS.Identity)
	c.MacOS.AppleID = expandEnv(c.MacOS.AppleID)
	c.MacOS.TeamID = expandEnv(c.MacOS.TeamID)
	c.MacOS.AppPassword = expandEnv(c.MacOS.AppPassword)
	c.Windows.Certificate = expandEnv(c.Windows.Certificate)
	c.Windows.Password = expandEnv(c.Windows.Password)
}

// expandEnv expands $VAR or ${VAR} in a string.
func expandEnv(s string) string {
	if !core.Contains(s, "$") {
		return s
	}

	buf := core.NewBuilder()
	for i := 0; i < len(s); {
		if s[i] != '$' {
			buf.WriteByte(s[i])
			i++
			continue
		}

		if i+1 < len(s) && s[i+1] == '{' {
			j := i + 2
			for j < len(s) && s[j] != '}' {
				j++
			}
			if j < len(s) {
				buf.WriteString(core.Env(s[i+2 : j]))
				i = j + 1
				continue
			}
		}

		j := i + 1
		for j < len(s) {
			c := s[j]
			if c != '_' && (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
				break
			}
			j++
		}
		if j > i+1 {
			buf.WriteString(core.Env(s[i+1 : j]))
			i = j
			continue
		}

		buf.WriteByte(s[i])
		i++
	}

	return buf.String()
}
