// SPDX-Licence-Identifier: EUPL-1.2

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { BuildApi } from './shared/api.js';

interface ArtifactInfo {
  name: string;
  path: string;
  size: number;
}

/**
 * <core-build-artifacts> — Shows dist/ contents and provides build trigger.
 * Includes a confirmation dialogue before triggering a build.
 */
@customElement('core-build-artifacts')
export class BuildArtifacts extends LitElement {
  static styles = css`
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1rem;
    }

    .toolbar-info {
      font-size: 0.8125rem;
      color: #6b7280;
    }

    button.build {
      padding: 0.5rem 1.25rem;
      background: #6366f1;
      color: #fff;
      border: none;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      font-weight: 500;
      cursor: pointer;
      transition: background 0.15s;
    }

    button.build:hover {
      background: #4f46e5;
    }

    button.build:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    .confirm {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      background: #fffbeb;
      border: 1px solid #fde68a;
      border-radius: 0.375rem;
      margin-bottom: 1rem;
      font-size: 0.8125rem;
    }

    .confirm-text {
      flex: 1;
      color: #92400e;
    }

    button.confirm-yes {
      padding: 0.375rem 1rem;
      background: #dc2626;
      color: #fff;
      border: none;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
    }

    button.confirm-yes:hover {
      background: #b91c1c;
    }

    button.confirm-no {
      padding: 0.375rem 0.75rem;
      background: #fff;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
    }

    .list {
      display: flex;
      flex-direction: column;
      gap: 0.375rem;
    }

    .artifact {
      border: 1px solid #e5e7eb;
      border-radius: 0.375rem;
      padding: 0.625rem 1rem;
      background: #fff;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .artifact-name {
      font-size: 0.875rem;
      font-family: monospace;
      font-weight: 500;
      color: #111827;
    }

    .artifact-size {
      font-size: 0.75rem;
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
      margin-bottom: 1rem;
    }

    .success {
      padding: 0.75rem;
      background: #f0fdf4;
      border: 1px solid #bbf7d0;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      color: #166534;
      margin-bottom: 1rem;
    }
  `;

  @property({ attribute: 'api-url' }) apiUrl = '';

  @state() private artifacts: ArtifactInfo[] = [];
  @state() private distExists = false;
  @state() private loading = true;
  @state() private error = '';
  @state() private building = false;
  @state() private confirmBuild = false;
  @state() private buildSuccess = '';

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
      const data = await this.api.artifacts();
      this.artifacts = data.artifacts ?? [];
      this.distExists = data.exists ?? false;
    } catch (e: any) {
      this.error = e.message ?? 'Failed to load artifacts';
    } finally {
      this.loading = false;
    }
  }

  private handleBuildClick() {
    this.confirmBuild = true;
    this.buildSuccess = '';
  }

  private handleCancelBuild() {
    this.confirmBuild = false;
  }

  private async handleConfirmBuild() {
    this.confirmBuild = false;
    this.building = true;
    this.error = '';
    this.buildSuccess = '';
    try {
      const result = await this.api.build();
      this.buildSuccess = `Build complete — ${result.artifacts?.length ?? 0} artifact(s) produced (${result.version})`;
      await this.reload();
    } catch (e: any) {
      this.error = e.message ?? 'Build failed';
    } finally {
      this.building = false;
    }
  }

  private formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }

  render() {
    if (this.loading) {
      return html`<div class="loading">Loading artifacts\u2026</div>`;
    }

    return html`
      <div class="toolbar">
        <span class="toolbar-info">
          ${this.distExists
            ? `${this.artifacts.length} file(s) in dist/`
            : 'No dist/ directory'}
        </span>
        <button
          class="build"
          ?disabled=${this.building}
          @click=${this.handleBuildClick}
        >
          ${this.building ? 'Building\u2026' : 'Build'}
        </button>
      </div>

      ${this.confirmBuild
        ? html`
            <div class="confirm">
              <span class="confirm-text">This will run a full build and overwrite dist/. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmBuild}>Build</button>
              <button class="confirm-no" @click=${this.handleCancelBuild}>Cancel</button>
            </div>
          `
        : nothing}

      ${this.error ? html`<div class="error">${this.error}</div>` : nothing}
      ${this.buildSuccess ? html`<div class="success">${this.buildSuccess}</div>` : nothing}

      ${this.artifacts.length === 0
        ? html`<div class="empty">${this.distExists ? 'dist/ is empty.' : 'Run a build to create artifacts.'}</div>`
        : html`
            <div class="list">
              ${this.artifacts.map(
                (a) => html`
                  <div class="artifact">
                    <span class="artifact-name">${a.name}</span>
                    <span class="artifact-size">${this.formatSize(a.size)}</span>
                  </div>
                `,
              )}
            </div>
          `}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'core-build-artifacts': BuildArtifacts;
  }
}
