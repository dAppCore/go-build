// SPDX-Licence-Identifier: EUPL-1.2

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { BuildApi } from './shared/api.js';

interface TargetConfig {
  os: string;
  arch: string;
}

interface CacheConfigData {
  enabled?: boolean;
  path?: string;
  paths?: string[];
}

interface AppleTriggerData {
  branch?: string;
  tag?: string;
  action?: string;
}

interface AppleConfigData {
  team_id?: string;
  bundle_id?: string;
  arch?: string;
  cert_identity?: string;
  profile_path?: string;
  keychain_path?: string;
  metadata_path?: string;
  sign?: boolean;
  notarise?: boolean;
  dmg?: boolean;
  testflight?: boolean;
  appstore?: boolean;
  api_key_id?: string;
  api_key_issuer_id?: string;
  api_key_path?: string;
  apple_id?: string;
  password?: string;
  bundle_display_name?: string;
  min_system_version?: string;
  category?: string;
  copyright?: string;
  privacy_policy_url?: string;
  dmg_background?: string;
  dmg_volume_name?: string;
  entitlements_path?: string;
  xcode_cloud?: {
    workflow?: string;
    triggers?: AppleTriggerData[];
  };
}

interface BuildConfigData {
  config: {
    version: number;
    project: {
      name: string;
      description: string;
      main: string;
      binary: string;
    };
    build: {
      type: string;
      cgo: boolean;
      obfuscate?: boolean;
      deno_build?: string;
      nsis?: boolean;
      webview2?: string;
      flags: string[];
      ldflags: string[];
      build_tags?: string[];
      archive_format?: string;
      env: string[];
      cache?: CacheConfigData;
      dockerfile?: string;
      registry?: string;
      image?: string;
      tags?: string[];
      push?: boolean;
      load?: boolean;
      linuxkit_config?: string;
      formats?: string[];
    };
    targets: TargetConfig[];
    apple?: AppleConfigData;
    sign: any;
  };
  has_config: boolean;
  path: string;
}

interface DiscoverData {
  types: string[];
  primary: string;
  primary_stack?: string;
  suggested_stack?: string;
  dir: string;
  has_frontend?: boolean;
  distro?: string;
  ref?: string;
  branch?: string;
  tag?: string;
  short_sha?: string;
  has_subtree_npm?: boolean;
  linux_packages?: string[];
  build_options?: string;
  options?: {
    obfuscate?: boolean;
    tags?: string[];
    nsis?: boolean;
    webview2?: string;
    ldflags?: string[];
  };
}

/**
 * <core-build-config> — Displays .core/build.yaml fields and project type detection.
 */
@customElement('core-build-config')
export class BuildConfig extends LitElement {
  static styles = css`
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .section {
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      padding: 1rem;
      background: #fff;
      margin-bottom: 1rem;
    }

    .section-title {
      font-size: 0.75rem;
      font-weight: 700;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
      margin-bottom: 0.75rem;
    }

    .field {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      padding: 0.375rem 0;
      border-bottom: 1px solid #f3f4f6;
    }

    .field:last-child {
      border-bottom: none;
    }

    .field-label {
      font-size: 0.8125rem;
      font-weight: 500;
      color: #374151;
    }

    .field-value {
      font-size: 0.8125rem;
      font-family: monospace;
      color: #6b7280;
      max-width: 36rem;
      text-align: right;
      word-break: break-word;
    }

    .badge {
      display: inline-block;
      font-size: 0.6875rem;
      font-weight: 600;
      padding: 0.125rem 0.5rem;
      border-radius: 1rem;
    }

    .badge.present {
      background: #dcfce7;
      color: #166534;
    }

    .badge.absent {
      background: #fef3c7;
      color: #92400e;
    }

    .badge.type-go {
      background: #dbeafe;
      color: #1e40af;
    }

    .badge.type-wails {
      background: #f3e8ff;
      color: #6b21a8;
    }

    .badge.type-node {
      background: #dcfce7;
      color: #166534;
    }

    .badge.type-php {
      background: #fef3c7;
      color: #92400e;
    }

    .badge.type-docker {
      background: #e0e7ff;
      color: #3730a3;
    }

    .targets {
      display: flex;
      flex-wrap: wrap;
      gap: 0.375rem;
      margin-top: 0.25rem;
    }

    .target-badge {
      font-size: 0.75rem;
      padding: 0.125rem 0.5rem;
      background: #f3f4f6;
      border-radius: 0.25rem;
      font-family: monospace;
      color: #374151;
    }

    .flags {
      display: flex;
      flex-wrap: wrap;
      gap: 0.25rem;
    }

    .flag {
      font-size: 0.75rem;
      padding: 0.0625rem 0.375rem;
      background: #f9fafb;
      border: 1px solid #e5e7eb;
      border-radius: 0.25rem;
      font-family: monospace;
      color: #6b7280;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      color: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      color: #6b7280;
    }

    .error {
      color: #dc2626;
      padding: 0.75rem;
      background: #fef2f2;
      border-radius: 0.375rem;
      font-size: 0.875rem;
    }
  `;

  @property({ attribute: 'api-url' }) apiUrl = '';

  @state() private configData: BuildConfigData | null = null;
  @state() private discoverData: DiscoverData | null = null;
  @state() private loading = true;
  @state() private error = '';

  private api!: BuildApi;

  connectedCallback() {
    super.connectedCallback();
    this.api = new BuildApi(this.apiUrl);
    this.reload();
  }

  async reload() {
    this.loading = true;
    this.error = '';
    try {
      const [configData, discoverData] = await Promise.all([
        this.api.config(),
        this.api.discover(),
      ]);
      this.configData = configData;
      this.discoverData = discoverData;
    } catch (e: any) {
      this.error = e.message ?? 'Failed to load configuration';
    } finally {
      this.loading = false;
    }
  }

  private hasAppleConfig(apple?: AppleConfigData): boolean {
    if (!apple) {
      return false;
    }

    return Object.entries(apple).some(([, value]) => {
      if (value == null) {
        return false;
      }
      if (Array.isArray(value)) {
        return value.length > 0;
      }
      if (typeof value === 'object') {
        return Object.keys(value).length > 0;
      }
      if (typeof value === 'string') {
        return value.length > 0;
      }
      return true;
    });
  }

  private renderToggle(label: string, enabled: boolean | undefined, onLabel = 'Enabled', offLabel = 'Disabled') {
    if (enabled == null) {
      return nothing;
    }

    return html`
      <div class="field">
        <span class="field-label">${label}</span>
        <span class="badge ${enabled ? 'present' : 'absent'}">
          ${enabled ? onLabel : offLabel}
        </span>
      </div>
    `;
  }

  private renderFlags(label: string, values?: string[]) {
    if (!values || values.length === 0) {
      return nothing;
    }

    return html`
      <div class="field">
        <span class="field-label">${label}</span>
        <div class="flags">
          ${values.map((value: string) => html`<span class="flag">${value}</span>`)}
        </div>
      </div>
    `;
  }

  render() {
    if (this.loading) {
      return html`<div class="loading">Loading configuration\u2026</div>`;
    }
    if (this.error) {
      return html`<div class="error">${this.error}</div>`;
    }
    if (!this.configData) {
      return html`<div class="empty">No configuration available.</div>`;
    }

    const cfg = this.configData.config;
    const disc = this.discoverData;

    return html`
      <!-- Discovery -->
      <div class="section">
        <div class="section-title">Project Detection</div>
        <div class="field">
          <span class="field-label">Config file</span>
          <span class="badge ${this.configData.has_config ? 'present' : 'absent'}">
            ${this.configData.has_config ? 'Present' : 'Using defaults'}
          </span>
        </div>
        ${disc
          ? html`
              <div class="field">
                <span class="field-label">Primary type</span>
                <span class="badge type-${disc.primary || 'unknown'}">${disc.primary || 'none'}</span>
              </div>
              <div class="field">
                <span class="field-label">Suggested stack</span>
                <span class="field-value">${disc.suggested_stack || disc.primary_stack || disc.primary || 'none'}</span>
              </div>
              ${disc.types.length > 1
                ? html`
                    <div class="field">
                      <span class="field-label">Detected types</span>
                      <span class="field-value">${disc.types.join(', ')}</span>
                    </div>
                  `
                : nothing}
              <div class="field">
                <span class="field-label">Frontend</span>
                <span class="badge ${disc.has_frontend ? 'present' : 'absent'}">
                  ${disc.has_frontend ? 'Detected' : 'None'}
                </span>
              </div>
              <div class="field">
                <span class="field-label">Nested frontend</span>
                <span class="badge ${disc.has_subtree_npm ? 'present' : 'absent'}">
                  ${disc.has_subtree_npm ? 'Depth 2' : 'None'}
                </span>
              </div>
              ${disc.distro
                ? html`
                    <div class="field">
                      <span class="field-label">Distro</span>
                      <span class="field-value">${disc.distro}</span>
                    </div>
                  `
                : nothing}
              ${disc.linux_packages && disc.linux_packages.length > 0
                ? html`
                    <div class="field">
                      <span class="field-label">Linux packages</span>
                      <div class="flags">
                        ${disc.linux_packages.map((pkg: string) => html`<span class="flag">${pkg}</span>`)}
                      </div>
                    </div>
                  `
                : nothing}
              ${disc.build_options
                ? html`
                    <div class="field">
                      <span class="field-label">Computed options</span>
                      <span class="field-value">${disc.build_options}</span>
                    </div>
                  `
                : nothing}
              ${this.renderToggle('Computed obfuscation', disc.options?.obfuscate)}
              ${this.renderToggle('Computed NSIS', disc.options?.nsis)}
              ${disc.options?.webview2
                ? html`
                    <div class="field">
                      <span class="field-label">Computed WebView2</span>
                      <span class="field-value">${disc.options.webview2}</span>
                    </div>
                  `
                : nothing}
              ${this.renderFlags('Computed tags', disc.options?.tags)}
              ${this.renderFlags('Computed LD flags', disc.options?.ldflags)}
              ${disc.ref
                ? html`
                    <div class="field">
                      <span class="field-label">Git ref</span>
                      <span class="field-value">${disc.ref}</span>
                    </div>
                  `
                : nothing}
              ${disc.branch
                ? html`
                    <div class="field">
                      <span class="field-label">Branch</span>
                      <span class="field-value">${disc.branch}</span>
                    </div>
                  `
                : nothing}
              ${disc.tag
                ? html`
                    <div class="field">
                      <span class="field-label">Tag</span>
                      <span class="field-value">${disc.tag}</span>
                    </div>
                  `
                : nothing}
              ${disc.short_sha
                ? html`
                    <div class="field">
                      <span class="field-label">Short SHA</span>
                      <span class="field-value">${disc.short_sha}</span>
                    </div>
                  `
                : nothing}
              <div class="field">
                <span class="field-label">Directory</span>
                <span class="field-value">${disc.dir}</span>
              </div>
            `
          : nothing}
      </div>

      <!-- Project -->
      <div class="section">
        <div class="section-title">Project</div>
        ${cfg.project.name
          ? html`
              <div class="field">
                <span class="field-label">Name</span>
                <span class="field-value">${cfg.project.name}</span>
              </div>
            `
          : nothing}
        ${cfg.project.description
          ? html`
              <div class="field">
                <span class="field-label">Description</span>
                <span class="field-value">${cfg.project.description}</span>
              </div>
            `
          : nothing}
        ${cfg.project.binary
          ? html`
              <div class="field">
                <span class="field-label">Binary</span>
                <span class="field-value">${cfg.project.binary}</span>
              </div>
            `
          : nothing}
        <div class="field">
          <span class="field-label">Main</span>
          <span class="field-value">${cfg.project.main}</span>
        </div>
      </div>

      <!-- Build Settings -->
      <div class="section">
        <div class="section-title">Build Settings</div>
        ${cfg.build.type
          ? html`
              <div class="field">
                <span class="field-label">Type override</span>
                <span class="field-value">${cfg.build.type}</span>
              </div>
            `
          : nothing}
        <div class="field">
          <span class="field-label">CGO</span>
          <span class="field-value">${cfg.build.cgo ? 'Enabled' : 'Disabled'}</span>
        </div>
        ${this.renderToggle('Obfuscation', cfg.build.obfuscate)}
        ${this.renderToggle('NSIS packaging', cfg.build.nsis)}
        ${cfg.build.webview2
          ? html`
              <div class="field">
                <span class="field-label">WebView2 mode</span>
                <span class="field-value">${cfg.build.webview2}</span>
              </div>
            `
          : nothing}
        ${cfg.build.deno_build
          ? html`
              <div class="field">
                <span class="field-label">Deno build</span>
                <span class="field-value">${cfg.build.deno_build}</span>
              </div>
            `
          : nothing}
        ${cfg.build.archive_format
          ? html`
              <div class="field">
                <span class="field-label">Archive format</span>
                <span class="field-value">${cfg.build.archive_format}</span>
              </div>
            `
          : nothing}
        ${this.renderFlags('Build tags', cfg.build.build_tags)}
        ${cfg.build.flags && cfg.build.flags.length > 0
          ? html`
              <div class="field">
                <span class="field-label">Flags</span>
                <div class="flags">
                  ${cfg.build.flags.map((f: string) => html`<span class="flag">${f}</span>`)}
                </div>
              </div>
            `
          : nothing}
        ${cfg.build.ldflags && cfg.build.ldflags.length > 0
          ? html`
              <div class="field">
                <span class="field-label">LD flags</span>
                <div class="flags">
                  ${cfg.build.ldflags.map((f: string) => html`<span class="flag">${f}</span>`)}
                </div>
              </div>
            `
          : nothing}
        ${this.renderFlags('Environment', cfg.build.env)}
        ${cfg.build.cache?.enabled || cfg.build.cache?.path || (cfg.build.cache?.paths && cfg.build.cache.paths.length > 0)
          ? html`
              ${this.renderToggle('Build cache', cfg.build.cache?.enabled)}
              ${cfg.build.cache?.path
                ? html`
                    <div class="field">
                      <span class="field-label">Cache path</span>
                      <span class="field-value">${cfg.build.cache.path}</span>
                    </div>
                  `
                : nothing}
              ${this.renderFlags('Cache paths', cfg.build.cache?.paths)}
            `
          : nothing}
        ${cfg.build.dockerfile
          ? html`
              <div class="field">
                <span class="field-label">Dockerfile</span>
                <span class="field-value">${cfg.build.dockerfile}</span>
              </div>
            `
          : nothing}
        ${cfg.build.image
          ? html`
              <div class="field">
                <span class="field-label">Image</span>
                <span class="field-value">${cfg.build.image}</span>
              </div>
            `
          : nothing}
        ${cfg.build.registry
          ? html`
              <div class="field">
                <span class="field-label">Registry</span>
                <span class="field-value">${cfg.build.registry}</span>
              </div>
            `
          : nothing}
        ${this.renderFlags('Image tags', cfg.build.tags)}
        ${this.renderToggle('Push image', cfg.build.push)}
        ${this.renderToggle('Load image', cfg.build.load)}
        ${cfg.build.linuxkit_config
          ? html`
              <div class="field">
                <span class="field-label">LinuxKit config</span>
                <span class="field-value">${cfg.build.linuxkit_config}</span>
              </div>
            `
          : nothing}
        ${this.renderFlags('LinuxKit formats', cfg.build.formats)}
      </div>

      <!-- Targets -->
      <div class="section">
        <div class="section-title">Targets</div>
        <div class="targets">
          ${cfg.targets.map(
            (t: TargetConfig) => html`<span class="target-badge">${t.os}/${t.arch}</span>`,
          )}
        </div>
      </div>

      ${cfg.apple && this.hasAppleConfig(cfg.apple)
        ? html`
            <div class="section">
              <div class="section-title">Apple Pipeline</div>
              ${cfg.apple.bundle_id
                ? html`
                    <div class="field">
                      <span class="field-label">Bundle ID</span>
                      <span class="field-value">${cfg.apple.bundle_id}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.team_id
                ? html`
                    <div class="field">
                      <span class="field-label">Team ID</span>
                      <span class="field-value">${cfg.apple.team_id}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.arch
                ? html`
                    <div class="field">
                      <span class="field-label">Architecture</span>
                      <span class="field-value">${cfg.apple.arch}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.bundle_display_name
                ? html`
                    <div class="field">
                      <span class="field-label">Display name</span>
                      <span class="field-value">${cfg.apple.bundle_display_name}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.min_system_version
                ? html`
                    <div class="field">
                      <span class="field-label">Minimum macOS</span>
                      <span class="field-value">${cfg.apple.min_system_version}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.category
                ? html`
                    <div class="field">
                      <span class="field-label">Category</span>
                      <span class="field-value">${cfg.apple.category}</span>
                    </div>
                  `
                : nothing}
              ${this.renderToggle('Sign', cfg.apple.sign)}
              ${this.renderToggle('Notarise', cfg.apple.notarise)}
              ${this.renderToggle('DMG', cfg.apple.dmg)}
              ${this.renderToggle('TestFlight', cfg.apple.testflight)}
              ${this.renderToggle('App Store', cfg.apple.appstore)}
              ${cfg.apple.metadata_path
                ? html`
                    <div class="field">
                      <span class="field-label">Metadata path</span>
                      <span class="field-value">${cfg.apple.metadata_path}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.privacy_policy_url
                ? html`
                    <div class="field">
                      <span class="field-label">Privacy policy</span>
                      <span class="field-value">${cfg.apple.privacy_policy_url}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.dmg_volume_name
                ? html`
                    <div class="field">
                      <span class="field-label">DMG volume</span>
                      <span class="field-value">${cfg.apple.dmg_volume_name}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.dmg_background
                ? html`
                    <div class="field">
                      <span class="field-label">DMG background</span>
                      <span class="field-value">${cfg.apple.dmg_background}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.entitlements_path
                ? html`
                    <div class="field">
                      <span class="field-label">Entitlements</span>
                      <span class="field-value">${cfg.apple.entitlements_path}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.xcode_cloud?.workflow
                ? html`
                    <div class="field">
                      <span class="field-label">Xcode Cloud workflow</span>
                      <span class="field-value">${cfg.apple.xcode_cloud.workflow}</span>
                    </div>
                  `
                : nothing}
              ${cfg.apple.xcode_cloud?.triggers && cfg.apple.xcode_cloud.triggers.length > 0
                ? html`
                    <div class="field">
                      <span class="field-label">Xcode Cloud triggers</span>
                      <div class="flags">
                        ${cfg.apple.xcode_cloud.triggers.map((trigger: AppleTriggerData) => {
                          const ref = trigger.branch ? `branch:${trigger.branch}` : trigger.tag ? `tag:${trigger.tag}` : 'manual';
                          const action = trigger.action ?? 'archive';
                          return html`<span class="flag">${ref} → ${action}</span>`;
                        })}
                      </div>
                    </div>
                  `
                : nothing}
            </div>
          `
        : nothing}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'core-build-config': BuildConfig;
  }
}
