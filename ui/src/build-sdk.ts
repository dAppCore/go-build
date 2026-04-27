// SPDX-Licence-Identifier: EUPL-1.2

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { BuildApi } from './shared/api.js';

/**
 * <core-build-sdk> — OpenAPI diff results and SDK generation controls.
 */
@customElement('core-build-sdk')
export class BuildSdk extends LitElement {
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

    .diff-form {
      display: flex;
      gap: 0.5rem;
      align-items: flex-end;
      margin-bottom: 1rem;
    }

    .diff-field {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
    }

    .diff-field label {
      font-size: 0.75rem;
      font-weight: 500;
      color: #6b7280;
    }

    .diff-field input {
      padding: 0.375rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      font-family: monospace;
    }

    .diff-field input:focus {
      outline: none;
      border-color: #6366f1;
      box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
    }

    button {
      padding: 0.375rem 1rem;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
      transition: background 0.15s;
    }

    button.primary {
      background: #6366f1;
      color: #fff;
      border: none;
    }

    button.primary:hover {
      background: #4f46e5;
    }

    button.primary:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    button.secondary {
      background: #fff;
      color: #374151;
      border: 1px solid #d1d5db;
    }

    button.secondary:hover {
      background: #f3f4f6;
    }

    .diff-result {
      padding: 0.75rem;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      margin-top: 0.75rem;
    }

    .diff-result.breaking {
      background: #fef2f2;
      border: 1px solid #fecaca;
      color: #991b1b;
    }

    .diff-result.safe {
      background: #f0fdf4;
      border: 1px solid #bbf7d0;
      color: #166534;
    }

    .diff-summary {
      font-weight: 600;
      margin-bottom: 0.5rem;
    }

    .diff-changes {
      list-style: disc;
      padding-left: 1.25rem;
      margin: 0;
    }

    .diff-changes li {
      font-size: 0.8125rem;
      margin-bottom: 0.25rem;
      font-family: monospace;
    }

    .generate-form {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }

    .generate-form select {
      padding: 0.375rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      background: #fff;
    }

    .empty {
      text-align: center;
      padding: 2rem;
      color: #9ca3af;
      font-size: 0.875rem;
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

    .loading {
      text-align: center;
      padding: 1rem;
      color: #6b7280;
      font-size: 0.875rem;
    }
  `;

  @property({ attribute: 'api-url' }) apiUrl = '';

  @state() private basePath = '';
  @state() private revisionPath = '';
  @state() private diffResult: any = null;
  @state() private diffing = false;
  @state() private diffError = '';

  @state() private selectedLanguage = '';
  @state() private generating = false;
  @state() private generateError = '';
  @state() private generateSuccess = '';

  private api!: BuildApi;

  connectedCallback() {
    super.connectedCallback();
    this.api = new BuildApi(this.apiUrl);
  }

  async reload() {
    // Reset state
    this.diffResult = null;
    this.diffError = '';
    this.generateError = '';
    this.generateSuccess = '';
  }

  private async handleDiff() {
    if (!this.basePath.trim() || !this.revisionPath.trim()) {
      this.diffError = 'Both base and revision spec paths are required.';
      return;
    }

    this.diffing = true;
    this.diffError = '';
    this.diffResult = null;
    try {
      this.diffResult = await this.api.sdkDiff(this.basePath.trim(), this.revisionPath.trim());
    } catch (e: any) {
      this.diffError = e.message ?? 'Diff failed';
    } finally {
      this.diffing = false;
    }
  }

  private async handleGenerate() {
    this.generating = true;
    this.generateError = '';
    this.generateSuccess = '';
    try {
      const result = await this.api.sdkGenerate(this.selectedLanguage || undefined);
      const lang = result.language || 'all languages';
      this.generateSuccess = `SDK generated successfully for ${lang}.`;
    } catch (e: any) {
      this.generateError = e.message ?? 'Generation failed';
    } finally {
      this.generating = false;
    }
  }

  render() {
    return html`
      <!-- OpenAPI Diff -->
      <div class="section">
        <div class="section-title">OpenAPI Diff</div>
        <div class="diff-form">
          <div class="diff-field">
            <label>Base spec</label>
            <input
              type="text"
              placeholder="path/to/base.yaml"
              .value=${this.basePath}
              @input=${(e: Event) => (this.basePath = (e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="diff-field">
            <label>Revision spec</label>
            <input
              type="text"
              placeholder="path/to/revision.yaml"
              .value=${this.revisionPath}
              @input=${(e: Event) => (this.revisionPath = (e.target as HTMLInputElement).value)}
            />
          </div>
          <button
            class="primary"
            ?disabled=${this.diffing}
            @click=${this.handleDiff}
          >
            ${this.diffing ? 'Comparing\u2026' : 'Compare'}
          </button>
        </div>

        ${this.diffError ? html`<div class="error">${this.diffError}</div>` : nothing}

        ${this.diffResult
          ? html`
              <div class="diff-result ${this.diffResult.Breaking ? 'breaking' : 'safe'}">
                <div class="diff-summary">${this.diffResult.Summary}</div>
                ${this.diffResult.Changes && this.diffResult.Changes.length > 0
                  ? html`
                      <ul class="diff-changes">
                        ${this.diffResult.Changes.map(
                          (change: string) => html`<li>${change}</li>`,
                        )}
                      </ul>
                    `
                  : nothing}
              </div>
            `
          : nothing}
      </div>

      <!-- SDK Generation -->
      <div class="section">
        <div class="section-title">SDK Generation</div>

        ${this.generateError ? html`<div class="error">${this.generateError}</div>` : nothing}
        ${this.generateSuccess ? html`<div class="success">${this.generateSuccess}</div>` : nothing}

        <div class="generate-form">
          <select
            .value=${this.selectedLanguage}
            @change=${(e: Event) => (this.selectedLanguage = (e.target as HTMLSelectElement).value)}
          >
            <option value="">All languages</option>
            <option value="typescript">TypeScript</option>
            <option value="python">Python</option>
            <option value="go">Go</option>
            <option value="php">PHP</option>
          </select>
          <button
            class="primary"
            ?disabled=${this.generating}
            @click=${this.handleGenerate}
          >
            ${this.generating ? 'Generating\u2026' : 'Generate SDK'}
          </button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'core-build-sdk': BuildSdk;
  }
}
