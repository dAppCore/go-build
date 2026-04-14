// SPDX-Licence-Identifier: EUPL-1.2

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { BuildApi } from './shared/api.js';

interface TargetConfig {
  os: string;
  arch: string;
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
      flags: string[];
      ldflags: string[];
      env: string[];
    };
    targets: TargetConfig[];
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
  has_subtree_npm?: boolean;
  linux_packages?: string[];
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
      colour: #6b7280;
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
      colour: #374151;
    }

    .field-value {
      font-size: 0.8125rem;
      font-family: monospace;
      colour: #6b7280;
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
      colour: #166534;
    }

    .badge.absent {
      background: #fef3c7;
      colour: #92400e;
    }

    .badge.type-go {
      background: #dbeafe;
      colour: #1e40af;
    }

    .badge.type-wails {
      background: #f3e8ff;
      colour: #6b21a8;
    }

    .badge.type-node {
      background: #dcfce7;
      colour: #166534;
    }

    .badge.type-php {
      background: #fef3c7;
      colour: #92400e;
    }

    .badge.type-docker {
      background: #e0e7ff;
      colour: #3730a3;
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
      colour: #374151;
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
      colour: #6b7280;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      colour: #9ca3af;
      font-size: 0.875rem;
    }

    .loading {
      text-align: center;
      padding: 2rem;
      colour: #6b7280;
    }

    .error {
      colour: #dc2626;
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
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'core-build-config': BuildConfig;
  }
}
