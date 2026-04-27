// SPDX-Licence-Identifier: EUPL-1.2

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { connectBuildEvents, type BuildEvent } from './shared/events.js';

// Side-effect imports to register child elements
import './build-config.js';
import './build-artifacts.js';
import './build-release.js';
import './build-sdk.js';

type TabId = 'config' | 'build' | 'release' | 'sdk';

/**
 * <core-build-panel> — Top-level HLCRF panel with tabs.
 *
 * Arranges child elements in HLCRF layout:
 * - H: Title bar with refresh button
 * - H-L: Navigation tabs
 * - C: Active tab content (one of the child elements)
 * - F: Status bar (connection state, last event)
 */
@customElement('core-build-panel')
export class BuildPanel extends LitElement {
  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      font-family: system-ui, -apple-system, sans-serif;
      height: 100%;
      background: #fafafa;
    }

    /* H — Header */
    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0.75rem 1rem;
      background: #fff;
      border-bottom: 1px solid #e5e7eb;
    }

    .title {
      font-weight: 700;
      font-size: 1rem;
      color: #111827;
    }

    .refresh-btn {
      padding: 0.375rem 0.75rem;
      border: 1px solid #d1d5db;
      border-radius: 0.375rem;
      background: #fff;
      font-size: 0.8125rem;
      cursor: pointer;
      transition: background 0.15s;
    }

    .refresh-btn:hover {
      background: #f3f4f6;
    }

    /* H-L — Tabs */
    .tabs {
      display: flex;
      gap: 0;
      background: #fff;
      border-bottom: 1px solid #e5e7eb;
      padding: 0 1rem;
    }

    .tab {
      padding: 0.625rem 1rem;
      font-size: 0.8125rem;
      font-weight: 500;
      color: #6b7280;
      cursor: pointer;
      border-bottom: 2px solid transparent;
      transition: all 0.15s;
      background: none;
      border-top: none;
      border-left: none;
      border-right: none;
    }

    .tab:hover {
      color: #374151;
    }

    .tab.active {
      color: #6366f1;
      border-bottom-color: #6366f1;
    }

    /* C — Content */
    .content {
      flex: 1;
      padding: 1rem;
      overflow-y: auto;
    }

    /* F — Footer / Status bar */
    .footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0.5rem 1rem;
      background: #fff;
      border-top: 1px solid #e5e7eb;
      font-size: 0.75rem;
      color: #9ca3af;
    }

    .ws-status {
      display: flex;
      align-items: center;
      gap: 0.375rem;
    }

    .ws-dot {
      width: 0.5rem;
      height: 0.5rem;
      border-radius: 50%;
    }

    .ws-dot.connected {
      background: #22c55e;
    }

    .ws-dot.disconnected {
      background: #ef4444;
    }

    .ws-dot.idle {
      background: #d1d5db;
    }
  `;

  @property({ attribute: 'api-url' }) apiUrl = '';
  @property({ attribute: 'ws-url' }) wsUrl = '';

  @state() private activeTab: TabId = 'config';
  @state() private wsConnected = false;
  @state() private lastEvent = '';

  private ws: WebSocket | null = null;

  connectedCallback() {
    super.connectedCallback();
    if (this.wsUrl) {
      this.connectWs();
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private connectWs() {
    this.ws = connectBuildEvents(this.wsUrl, (event: BuildEvent) => {
      this.lastEvent = event.channel ?? event.type ?? '';
      this.requestUpdate();
    });
    this.ws.onopen = () => {
      this.wsConnected = true;
    };
    this.ws.onclose = () => {
      this.wsConnected = false;
    };
  }

  private handleTabClick(tab: TabId) {
    this.activeTab = tab;
  }

  private handleRefresh() {
    const content = this.shadowRoot?.querySelector('.content');
    if (content) {
      const child = content.firstElementChild;
      if (child && 'reload' in child) {
        (child as any).reload();
      }
    }
  }

  private renderContent() {
    switch (this.activeTab) {
      case 'config':
        return html`<core-build-config api-url=${this.apiUrl}></core-build-config>`;
      case 'build':
        return html`<core-build-artifacts api-url=${this.apiUrl}></core-build-artifacts>`;
      case 'release':
        return html`<core-build-release api-url=${this.apiUrl}></core-build-release>`;
      case 'sdk':
        return html`<core-build-sdk api-url=${this.apiUrl}></core-build-sdk>`;
      default:
        return nothing;
    }
  }

  private tabs: { id: TabId; label: string }[] = [
    { id: 'config', label: 'Config' },
    { id: 'build', label: 'Build' },
    { id: 'release', label: 'Release' },
    { id: 'sdk', label: 'SDK' },
  ];

  render() {
    const wsState = this.wsUrl
      ? this.wsConnected
        ? 'connected'
        : 'disconnected'
      : 'idle';

    return html`
      <div class="header">
        <span class="title">Build</span>
        <button class="refresh-btn" @click=${this.handleRefresh}>Refresh</button>
      </div>

      <div class="tabs">
        ${this.tabs.map(
          (tab) => html`
            <button
              class="tab ${this.activeTab === tab.id ? 'active' : ''}"
              @click=${() => this.handleTabClick(tab.id)}
            >
              ${tab.label}
            </button>
          `,
        )}
      </div>

      <div class="content">${this.renderContent()}</div>

      <div class="footer">
        <div class="ws-status">
          <span class="ws-dot ${wsState}"></span>
          <span>${wsState === 'connected' ? 'Connected' : wsState === 'disconnected' ? 'Disconnected' : 'No WebSocket'}</span>
        </div>
        ${this.lastEvent ? html`<span>Last: ${this.lastEvent}</span>` : nothing}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'core-build-panel': BuildPanel;
  }
}
