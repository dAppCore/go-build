// SPDX-Licence-Identifier: EUPL-1.2

export interface BuildEvent {
  type: string;
  channel?: string;
  data?: any;
  timestamp?: string;
}

/**
 * Connects to a WebSocket endpoint and dispatches build events to a handler.
 * Returns the WebSocket instance for lifecycle management.
 */
export function connectBuildEvents(
  wsUrl: string,
  handler: (event: BuildEvent) => void,
): WebSocket {
  const ws = new WebSocket(wsUrl);

  ws.onmessage = (e: MessageEvent) => {
    try {
      const event: BuildEvent = JSON.parse(e.data);
      if (
        event.type?.startsWith?.('build.') ||
        event.type?.startsWith?.('release.') ||
        event.type?.startsWith?.('sdk.') ||
        event.channel?.startsWith?.('build.') ||
        event.channel?.startsWith?.('release.') ||
        event.channel?.startsWith?.('sdk.')
      ) {
        handler(event);
      }
    } catch {
      // Ignore malformed messages
    }
  };

  return ws;
}
