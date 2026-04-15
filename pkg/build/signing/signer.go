// Package signing provides code signing for build artifacts.
package signing

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/io"
)

// Signer defines the interface for code signing implementations.
//
// var s signing.Signer = signing.NewGPGSigner(keyID)
// err := s.Sign(ctx, io.Local, "dist/myapp")
type Signer interface {
	// Name returns the signer's identifier.
	Name() string
	// Available checks if this signer can be used.
	Available() bool
	// Sign signs the artifact at the given path.
	Sign(ctx context.Context, fs io.Medium, path string) error
}

// SignConfig holds signing configuration from .core/build.yaml.
//
// cfg := signing.DefaultSignConfig()
type SignConfig struct {
	Enabled bool          `json:"enabled" yaml:"enabled"`
	GPG     GPGConfig     `json:"gpg,omitempty" yaml:"gpg,omitempty"`
	MacOS   MacOSConfig   `json:"macos,omitempty" yaml:"macos,omitempty"`
	Windows WindowsConfig `json:"windows,omitempty" yaml:"windows,omitempty"`
}

// GPGConfig holds GPG signing configuration.
//
// cfg := signing.GPGConfig{Key: "ABCD1234"}
type GPGConfig struct {
	Key string `json:"key" yaml:"key"` // Key ID or fingerprint, supports $ENV
}

// MacOSConfig holds macOS codesign configuration.
//
// cfg := signing.MacOSConfig{Identity: "Developer ID Application: Acme Inc (TEAM123)"}
type MacOSConfig struct {
	Identity    string `json:"identity" yaml:"identity"`         // Developer ID Application: ...
	Notarize    bool   `json:"notarize" yaml:"notarize"`         // Submit to Apple for notarization
	AppleID     string `json:"apple_id" yaml:"apple_id"`         // Apple account email
	TeamID      string `json:"team_id" yaml:"team_id"`           // Team ID
	AppPassword string `json:"app_password" yaml:"app_password"` // App-specific password
}

// WindowsConfig holds Windows signtool configuration.
//
// cfg := signing.WindowsConfig{Certificate: "cert.pfx", Password: "secret"}
type WindowsConfig struct {
	Signtool    bool   `json:"signtool" yaml:"signtool"`       // Enable/disable signtool integration.
	Certificate string `json:"certificate" yaml:"certificate"` // Path to .pfx
	Password    string `json:"password" yaml:"password"`       // Certificate password

	signtoolExplicit bool `json:"-" yaml:"-"`
}

// DefaultSignConfig returns sensible defaults.
//
// cfg := signing.DefaultSignConfig()
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
		Windows: WindowsConfig{
			Signtool:    true,
			Certificate: core.Env("SIGNTOOL_CERTIFICATE"),
			Password:    core.Env("SIGNTOOL_PASSWORD"),
		},
	}
}

// ExpandEnv expands environment variables in config values.
//
// cfg.ExpandEnv() // expands $GPG_KEY_ID, $CODESIGN_IDENTITY etc.
func (c *SignConfig) ExpandEnv() {
	c.GPG.Key = expandEnv(c.GPG.Key)
	c.MacOS.Identity = expandEnv(c.MacOS.Identity)
	c.MacOS.AppleID = expandEnv(c.MacOS.AppleID)
	c.MacOS.TeamID = expandEnv(c.MacOS.TeamID)
	c.MacOS.AppPassword = expandEnv(c.MacOS.AppPassword)
	c.Windows.Certificate = expandEnv(c.Windows.Certificate)
	c.Windows.Password = expandEnv(c.Windows.Password)
}

func (c WindowsConfig) signtoolEnabled() bool {
	if c.signtoolExplicit {
		return c.Signtool
	}
	return true
}

// SetSigntool records an explicit signtool preference from config.
func (c *WindowsConfig) SetSigntool(enabled bool) {
	if c == nil {
		return
	}
	c.Signtool = enabled
	c.signtoolExplicit = true
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
