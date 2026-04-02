// SPDX-Licence-Identifier: EUPL-1.2

/**
 * BuildApi provides a typed fetch wrapper for the /api/v1/build/* endpoints.
 */
export class BuildApi {
  constructor(private baseUrl: string = '') {}

  private get base(): string {
    return `${this.baseUrl}/api/v1/build`;
  }

  private async request<T>(path: string, opts?: RequestInit): Promise<T> {
    const res = await fetch(`${this.base}${path}`, opts);
    const json = await res.json();
    if (!json.success) {
      throw new Error(json.error?.message ?? 'Request failed');
    }
    return json.data as T;
  }

  // -- Build ------------------------------------------------------------------

  config() {
    return this.request<any>('/config');
  }

  discover() {
    return this.request<any>('/discover');
  }

  build() {
    return this.request<any>('/build', { method: 'POST' });
  }

  artifacts() {
    return this.request<any>('/artifacts');
  }

  // -- Release ----------------------------------------------------------------

  version() {
    return this.request<any>('/release/version');
  }

  changelog(from?: string, to?: string) {
    const params = new URLSearchParams();
    if (from) params.set('from', from);
    if (to) params.set('to', to);
    const qs = params.toString();
    return this.request<any>(`/release/changelog${qs ? `?${qs}` : ''}`);
  }

  release(dryRun = false) {
    const qs = dryRun ? '?dry_run=true' : '';
    return this.request<any>(`/release${qs}`, { method: 'POST' });
  }

  releaseWorkflow(request: {
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
  } = {}) {
    return this.request<any>('/release/workflow', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
  }

  // -- SDK --------------------------------------------------------------------

  sdkDiff(base: string, revision: string) {
    const params = new URLSearchParams({ base, revision });
    return this.request<any>(`/sdk/diff?${params.toString()}`);
  }

  sdkGenerate(language?: string) {
    const body = language ? JSON.stringify({ language }) : undefined;
    return this.request<any>('/sdk/generate', {
      method: 'POST',
      headers: body ? { 'Content-Type': 'application/json' } : undefined,
      body,
    });
  }
}
