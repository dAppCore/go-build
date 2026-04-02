// SPDX-Licence-Identifier: EUPL-1.2

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { BuildApi } from './shared/api.js';

/**
 * <core-build-release> — Version display, changelog preview, and release trigger.
 * Includes confirmation dialogue and dry-run support for safety.
 */
@customElement('core-build-release')
export class BuildRelease extends LitElement {
  static styles = css`
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .version-bar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 1rem;
      background: #fff;
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      margin-bottom: 1rem;
    }

    .version-label {
      font-size: 0.75rem;
      font-weight: 600;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .version-value {
      font-size: 1.25rem;
      font-weight: 700;
      font-family: monospace;
      color: #111827;
    }

    .actions {
      display: flex;
      gap: 0.5rem;
      flex-wrap: wrap;
    }

    button {
      padding: 0.5rem 1rem;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
      transition: background 0.15s;
    }

    button.release {
      background: #6366f1;
      color: #fff;
      border: none;
      font-weight: 500;
    }

    button.release:hover {
      background: #4f46e5;
    }

    button.release:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    button.dry-run {
      background: #fff;
      color: #6366f1;
      border: 1px solid #6366f1;
    }

    button.dry-run:hover {
      background: #eef2ff;
    }

    .workflow-section {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
      padding: 0.875rem 1rem;
      background: linear-gradient(180deg, #fff, #f8fafc);
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      margin-bottom: 1rem;
    }

    .workflow-fields {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }

    .workflow-field {
      display: flex;
      gap: 0.5rem;
      align-items: center;
      flex-wrap: wrap;
    }

    .workflow-field-label {
      min-width: 9rem;
      font-size: 0.8125rem;
      font-weight: 600;
      color: #374151;
    }

    .workflow-row {
      display: flex;
      gap: 0.5rem;
      align-items: center;
      flex-wrap: wrap;
    }

    .workflow-label {
      font-size: 0.75rem;
      font-weight: 700;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .workflow-input {
      flex: 1;
      min-width: 16rem;
      padding: 0.5rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      font-family: monospace;
      color: #111827;
      background: #fff;
    }

    .workflow-input:focus {
      outline: none;
      border-color: #6366f1;
      box-shadow: 0 0 0 3px rgb(99 102 241 / 12%);
    }

    button.workflow {
      background: #111827;
      color: #fff;
      border: none;
      font-weight: 500;
    }

    button.workflow:hover {
      background: #1f2937;
    }

    button.workflow:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    .confirm {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      background: #fef2f2;
      border: 1px solid #fecaca;
      border-radius: 0.375rem;
      margin-bottom: 1rem;
      font-size: 0.8125rem;
    }

    .confirm-text {
      flex: 1;
      color: #991b1b;
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

    button.confirm-no {
      padding: 0.375rem 0.75rem;
      background: #fff;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      font-size: 0.8125rem;
      cursor: pointer;
    }

    .changelog-section {
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      background: #fff;
    }

    .changelog-header {
      padding: 0.75rem 1rem;
      border-bottom: 1px solid #e5e7eb;
      font-size: 0.75rem;
      font-weight: 700;
      color: #6b7280;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .changelog-content {
      padding: 1rem;
      font-size: 0.875rem;
      line-height: 1.6;
      white-space: pre-wrap;
      font-family: system-ui, -apple-system, sans-serif;
      color: #374151;
      max-height: 400px;
      overflow-y: auto;
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

  @state() private version = '';
  @state() private changelog = '';
  @state() private loading = true;
  @state() private error = '';
  @state() private releasing = false;
  @state() private confirmRelease = false;
  @state() private releaseSuccess = '';
  @state() private workflowPath = '.github/workflows/release.yml';
  @state() private workflowOutputPath = '';
  @state() private generatingWorkflow = false;
  @state() private workflowSuccess = '';

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
      const [versionData, changelogData] = await Promise.all([
        this.api.version(),
        this.api.changelog(),
      ]);
      this.version = versionData.version ?? '';
      this.changelog = changelogData.changelog ?? '';
    } catch (e: any) {
      this.error = e.message ?? 'Failed to load release information';
    } finally {
      this.loading = false;
    }
  }

  private handleReleaseClick() {
    this.confirmRelease = true;
    this.releaseSuccess = '';
  }

  private handleWorkflowPathInput(event: InputEvent) {
    const target = event.target as HTMLInputElement | null;
    this.workflowPath = target?.value ?? '';
  }

  private handleWorkflowOutputPathInput(event: InputEvent) {
    const target = event.target as HTMLInputElement | null;
    this.workflowOutputPath = target?.value ?? '';
  }

  private async handleGenerateWorkflow() {
    this.generatingWorkflow = true;
    this.error = '';
    this.workflowSuccess = '';
    try {
      const request: {
        path?: string;
        workflowPath?: string;
        workflow_path?: string;
        'workflow-path'?: string;
        outputPath?: string;
        'output-path'?: string;
        output_path?: string;
        output?: string;
        workflowOutputPath?: string;
        workflow_output?: string;
        'workflow-output'?: string;
        workflow_output_path?: string;
        'workflow-output-path'?: string;
      } = {};
      const path = this.workflowPath.trim();
      const outputPath = this.workflowOutputPath.trim();
      if (path) request.path = path;
      if (path) {
        request.workflowPath = path;
        request.workflow_path = path;
        request['workflow-path'] = path;
      }
      if (outputPath) request.outputPath = outputPath;
      if (outputPath) {
        request['output-path'] = outputPath;
        request.output_path = outputPath;
        request.output = outputPath;
        request.workflowOutputPath = outputPath;
        request.workflow_output = outputPath;
        request['workflow-output'] = outputPath;
        request.workflow_output_path = outputPath;
        request['workflow-output-path'] = outputPath;
      }

      const result = await this.api.releaseWorkflow(request);
      const generatedPath = result.path ?? outputPath ?? path ?? '.github/workflows/release.yml';
      this.workflowSuccess = `Workflow generated at ${generatedPath}`;
    } catch (e: any) {
      this.error = e.message ?? 'Failed to generate release workflow';
    } finally {
      this.generatingWorkflow = false;
    }
  }

  private handleCancelRelease() {
    this.confirmRelease = false;
  }

  private async handleConfirmRelease() {
    this.confirmRelease = false;
    await this.doRelease(false);
  }

  private async handleDryRun() {
    await this.doRelease(true);
  }

  private async doRelease(dryRun: boolean) {
    this.releasing = true;
    this.error = '';
    this.releaseSuccess = '';
    try {
      const result = await this.api.release(dryRun);
      const prefix = dryRun ? 'Dry run complete' : 'Release published';
      this.releaseSuccess = `${prefix} — ${result.version} (${result.artifacts?.length ?? 0} artifact(s))`;
      await this.reload();
    } catch (e: any) {
      this.error = e.message ?? 'Release failed';
    } finally {
      this.releasing = false;
    }
  }

  render() {
    if (this.loading) {
      return html`<div class="loading">Loading release information\u2026</div>`;
    }

    return html`
      ${this.error ? html`<div class="error">${this.error}</div>` : nothing}
      ${this.releaseSuccess ? html`<div class="success">${this.releaseSuccess}</div>` : nothing}
      ${this.workflowSuccess ? html`<div class="success">${this.workflowSuccess}</div>` : nothing}

      <div class="version-bar">
        <div>
          <div class="version-label">Current Version</div>
          <div class="version-value">${this.version || 'unknown'}</div>
        </div>
        <div class="actions">
          <button
            class="dry-run"
            ?disabled=${this.releasing}
            @click=${this.handleDryRun}
          >
            Dry Run
          </button>
          <button
            class="release"
            ?disabled=${this.releasing}
            @click=${this.handleReleaseClick}
          >
            ${this.releasing ? 'Publishing\u2026' : 'Publish Release'}
          </button>
        </div>
      </div>

      <div class="workflow-section">
        <div class="workflow-label">Release Workflow</div>
        <div class="workflow-fields">
          <div class="workflow-field">
            <div class="workflow-field-label">Workflow Path</div>
            <input
              class="workflow-input"
              type="text"
              .value=${this.workflowPath}
              @input=${this.handleWorkflowPathInput}
              placeholder=".github/workflows/release.yml"
              aria-label="Workflow path"
            />
          </div>
          <div class="workflow-field">
            <div class="workflow-field-label">Workflow Output Path</div>
            <input
              class="workflow-input"
              type="text"
              .value=${this.workflowOutputPath}
              @input=${this.handleWorkflowOutputPathInput}
              placeholder="ci/release.yml"
              aria-label="Workflow output path"
            />
          </div>
        </div>
        <div class="workflow-row">
          <button
            class="workflow"
            ?disabled=${this.generatingWorkflow}
            @click=${this.handleGenerateWorkflow}
          >
            ${this.generatingWorkflow ? 'Generating…' : 'Generate Workflow'}
          </button>
        </div>
      </div>

      ${this.confirmRelease
        ? html`
            <div class="confirm">
              <span class="confirm-text">This will publish ${this.version} to all configured targets. This action cannot be undone. Continue?</span>
              <button class="confirm-yes" @click=${this.handleConfirmRelease}>Publish</button>
              <button class="confirm-no" @click=${this.handleCancelRelease}>Cancel</button>
            </div>
          `
        : nothing}

      ${this.changelog
        ? html`
            <div class="changelog-section">
              <div class="changelog-header">Changelog</div>
              <div class="changelog-content">${this.changelog}</div>
            </div>
          `
        : html`<div class="empty">No changelog available.</div>`}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'core-build-release': BuildRelease;
  }
}
