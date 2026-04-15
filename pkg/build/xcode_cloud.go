package build

import (
	"strings"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

const (
	// XcodeCloudScriptsDir is the repository-relative directory used by Xcode Cloud.
	XcodeCloudScriptsDir = "ci_scripts"

	// XcodeCloudPostCloneScriptName installs toolchains and project dependencies.
	XcodeCloudPostCloneScriptName = "ci_post_clone.sh"
	// XcodeCloudPreXcodebuildScriptName runs the Apple pipeline before xcodebuild.
	XcodeCloudPreXcodebuildScriptName = "ci_pre_xcodebuild.sh"
	// XcodeCloudPostXcodebuildScriptName verifies the built bundle after xcodebuild.
	XcodeCloudPostXcodebuildScriptName = "ci_post_xcodebuild.sh"
)

// HasXcodeCloudConfig reports whether apple.xcode_cloud contains workflow metadata.
func HasXcodeCloudConfig(cfg *BuildConfig) bool {
	if cfg == nil {
		return false
	}

	if core.Trim(cfg.Apple.XcodeCloud.Workflow) != "" {
		return true
	}

	return len(cfg.Apple.XcodeCloud.Triggers) > 0
}

// GenerateXcodeCloudScripts renders the three Xcode Cloud helper scripts.
func GenerateXcodeCloudScripts(projectDir string, cfg *BuildConfig) map[string]string {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	bundleName := resolveXcodeCloudBundleName(projectDir, cfg)
	buildCommand := resolveXcodeCloudBuildCommand(cfg)

	return map[string]string{
		XcodeCloudPostCloneScriptName:     generateXcodeCloudPostCloneScript(),
		XcodeCloudPreXcodebuildScriptName: generateXcodeCloudPreXcodebuildScript(buildCommand),
		XcodeCloudPostXcodebuildScriptName: generateXcodeCloudPostXcodebuildScript(
			bundleName,
		),
	}
}

// WriteXcodeCloudScripts writes the Xcode Cloud helper scripts to ci_scripts/.
func WriteXcodeCloudScripts(filesystem io.Medium, projectDir string, cfg *BuildConfig) ([]string, error) {
	if filesystem == nil {
		return nil, coreerr.E("build.WriteXcodeCloudScripts", "filesystem medium is required", nil)
	}

	scripts := GenerateXcodeCloudScripts(projectDir, cfg)
	orderedNames := []string{
		XcodeCloudPostCloneScriptName,
		XcodeCloudPreXcodebuildScriptName,
		XcodeCloudPostXcodebuildScriptName,
	}

	baseDir := ax.Join(projectDir, XcodeCloudScriptsDir)
	if err := filesystem.EnsureDir(baseDir); err != nil {
		return nil, coreerr.E("build.WriteXcodeCloudScripts", "failed to create Xcode Cloud scripts directory", err)
	}

	paths := make([]string, 0, len(orderedNames))
	for _, name := range orderedNames {
		path := ax.Join(baseDir, name)
		if err := filesystem.WriteMode(path, scripts[name], 0o755); err != nil {
			return nil, coreerr.E("build.WriteXcodeCloudScripts", "failed to write "+name, err)
		}
		paths = append(paths, path)
	}

	return paths, nil
}

func resolveXcodeCloudBundleName(projectDir string, cfg *BuildConfig) string {
	if cfg != nil {
		if cfg.Project.Binary != "" {
			return cfg.Project.Binary
		}
		if cfg.Project.Name != "" {
			return cfg.Project.Name
		}
	}

	if core.Trim(projectDir) == "" {
		return "App"
	}

	return ax.Base(projectDir)
}

func resolveXcodeCloudBuildCommand(cfg *BuildConfig) string {
	options := DefaultAppleOptions()
	if cfg != nil {
		options = cfg.Apple.Resolve()
	}

	args := []string{
		"core",
		"build",
		"apple",
		"--arch",
		shellQuote(firstNonEmpty(options.Arch, defaultAppleArch)),
		"--config",
		shellQuote(ax.Join(ConfigDir, ConfigFileName)),
	}

	if !options.Sign {
		args = append(args, "--sign=false")
	}
	if !options.Notarise {
		args = append(args, "--notarise=false")
	}
	if options.DMG {
		args = append(args, "--dmg")
	}
	if options.TestFlight {
		args = append(args, "--testflight")
	}
	if options.AppStore {
		args = append(args, "--appstore")
	}
	if core.Trim(options.BundleID) != "" {
		args = append(args, "--bundle-id", shellQuote(options.BundleID))
	}
	if core.Trim(options.TeamID) != "" {
		args = append(args, "--team-id", shellQuote(options.TeamID))
	}

	return strings.Join(args, " ")
}

func generateXcodeCloudPostCloneScript() string {
	return strings.TrimSpace(`#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/go/bin:${HOME}/.deno/bin:${HOME}/.bun/bin:${PATH}"

deno_requested() {
  case "${DENO_ENABLE:-}" in
    1|true|TRUE|yes|YES|on|ON)
      return 0
      ;;
  esac

  [ -n "${DENO_BUILD:-}" ]
}

find_visible_files() {
  local maxdepth="$1"
  shift
  find . -maxdepth "$maxdepth" \
    \( -path './.*' -o -path '*/.*' -o -path '*/node_modules' -o -path '*/node_modules/*' \) -prune -o \
    "$@" -print
}

package_manager_from_manifest() {
  local manifest_path="$1/package.json"
  if [ ! -f "$manifest_path" ]; then
    return 0
  fi

  node -e '
const fs = require("fs");
const manifestPath = process.argv[1];
try {
  const pkg = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
  const raw = typeof pkg.packageManager === "string" ? pkg.packageManager.trim() : "";
  if (!raw) process.exit(0);
  const manager = raw.split("@")[0];
  if (["bun", "npm", "pnpm", "yarn"].includes(manager)) {
    process.stdout.write(manager);
  }
} catch (_) {}
' "$manifest_path"
}

install_node_package_dir() {
  local dir="$1"
  if [ ! -f "$dir/package.json" ]; then
    return 0
  fi

  declared_manager="$(package_manager_from_manifest "$dir")"
  case "$declared_manager" in
    pnpm)
      corepack enable pnpm
      if [ -f "$dir/pnpm-lock.yaml" ]; then
        (cd "$dir" && pnpm install --frozen-lockfile)
      else
        (cd "$dir" && pnpm install)
      fi
      return 0
      ;;
    yarn)
      corepack enable yarn
      if [ -f "$dir/yarn.lock" ]; then
        (cd "$dir" && yarn install --immutable)
      else
        (cd "$dir" && yarn install)
      fi
      return 0
      ;;
    bun)
      if ! command -v bun >/dev/null 2>&1; then
        curl -fsSL https://bun.sh/install | bash
        export PATH="${HOME}/.bun/bin:${PATH}"
      fi
      if [ -f "$dir/bun.lockb" ] || [ -f "$dir/bun.lock" ]; then
        (cd "$dir" && bun install --frozen-lockfile)
      else
        (cd "$dir" && bun install)
      fi
      return 0
      ;;
    npm)
      if [ -f "$dir/package-lock.json" ]; then
        (cd "$dir" && npm ci)
      else
        (cd "$dir" && npm install)
      fi
      return 0
      ;;
  esac

  if [ -f "$dir/pnpm-lock.yaml" ]; then
    corepack enable pnpm
    (cd "$dir" && pnpm install --frozen-lockfile)
    return 0
  fi

  if [ -f "$dir/yarn.lock" ]; then
    corepack enable yarn
    (cd "$dir" && yarn install --immutable)
    return 0
  fi

  if [ -f "$dir/bun.lockb" ] || [ -f "$dir/bun.lock" ]; then
    if ! command -v bun >/dev/null 2>&1; then
      curl -fsSL https://bun.sh/install | bash
      export PATH="${HOME}/.bun/bin:${PATH}"
    fi
    (cd "$dir" && bun install --frozen-lockfile)
    return 0
  fi

  if [ -f "$dir/package-lock.json" ]; then
    (cd "$dir" && npm ci)
    return 0
  fi

  (cd "$dir" && npm install)
}

if ! command -v go >/dev/null 2>&1; then
  if command -v brew >/dev/null 2>&1; then
    brew install go
  else
    echo "Go is required for Xcode Cloud builds." >&2
    exit 1
  fi
fi

if ! command -v node >/dev/null 2>&1; then
  if command -v brew >/dev/null 2>&1; then
    brew install node
  else
    echo "Node.js is required for Xcode Cloud builds." >&2
    exit 1
  fi
fi

if ! command -v wails3 >/dev/null 2>&1 && ! command -v wails >/dev/null 2>&1; then
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
fi

if deno_requested || find_visible_files 3 \( -name deno.json -o -name deno.jsonc \) | grep -q .; then
  if ! command -v deno >/dev/null 2>&1; then
    curl -fsSL https://deno.land/install.sh | sh
    export PATH="${HOME}/.deno/bin:${PATH}"
  fi
fi

install_node_package_dir "."

if [ -d frontend ]; then
  install_node_package_dir "./frontend"
fi

while IFS= read -r manifest; do
  dir="$(dirname "$manifest")"
  case "$dir" in
    "."|"./frontend")
      continue
      ;;
  esac
  install_node_package_dir "$dir"
done < <(find_visible_files 3 -name package.json | sort)
`) + "\n"
}

func generateXcodeCloudPreXcodebuildScript(buildCommand string) string {
	return strings.TrimSpace(core.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/go/bin:${HOME}/.deno/bin:${HOME}/.bun/bin:${PATH}"

%s
`, buildCommand)) + "\n"
}

func generateXcodeCloudPostXcodebuildScript(bundleName string) string {
	bundlePath := ax.Join("dist", "apple", bundleName+".app")
	executablePath := ax.Join(bundlePath, "Contents", "MacOS", bundleName)

	return strings.TrimSpace(core.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

BUNDLE_PATH=%s
EXECUTABLE_PATH=%s

if [ ! -d "$BUNDLE_PATH" ]; then
  echo "Expected bundle not found: $BUNDLE_PATH" >&2
  exit 1
fi

if [ ! -x "$EXECUTABLE_PATH" ]; then
  echo "Expected executable not found: $EXECUTABLE_PATH" >&2
  exit 1
fi

if command -v codesign >/dev/null 2>&1; then
  codesign --verify --deep --strict "$BUNDLE_PATH"
fi

if command -v spctl >/dev/null 2>&1; then
  spctl --assess --type execute "$BUNDLE_PATH" || true
fi
`, shellQuote(bundlePath), shellQuote(executablePath))) + "\n"
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
