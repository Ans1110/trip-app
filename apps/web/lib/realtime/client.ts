import { isServerMsg, type ClientMsg, type ServerMsg } from "./protocol";

export type RealtimeState =
  | "idle"
  | "connecting"
  | "open"
  | "reconnecting"
  | "closed";

export type RealtimeClientConfig = {
  wsUrl: string;

  // Called every time open a socket. Tickets are single-use
  // so this MUST return a fresh ticket per connect attempt — never cache.
  fetchTicket: (signal: AbortSignal) => Promise<string>;

  onMessage: (msg: ServerMsg) => void;
  onStateChange?: (state: RealtimeState) => void;
  onError?: (err: unknown) => void;

  // If false, skip the document.visibilityState auto-pause. Default: true.
  pauseOnHidden?: boolean;
};

const BACKOFF_BASE_MS = 500;
const BACKOFF_CAP_MS = 30_000;
const BACKOFF_JITTER = 0.3;

// RealtimeClient owns one WebSocket lifecycle: connect -> (open|close) ->
// exponential backoff reconnect. tickets are single-use, every
// reconnect must call fetchTicket() again — never reuse a URL.
//
// While document.visibilityState === "hidden", drop the socket (React Query
// will re-hydrate on focus anyway) and re-establish on visible.
export class RealtimeClient {
  private ws: WebSocket | null = null;
  private state: RealtimeState = "idle";
  private shouldRun = false;
  private pausedByHidden = false;
  private attempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private ticketAbort: AbortController | null = null;
  private visibilityHandler: (() => void) | null = null;

  constructor(private readonly cfg: RealtimeClientConfig) {}

  getState(): RealtimeState {
    return this.state;
  }

  connect(): void {
    if (this.shouldRun) return;
    this.shouldRun = true;
    this.attempt = 0;
    if (this.cfg.pauseOnHidden !== false) {
      this.installVisibilityHandler();
    }
    if (this.isDocumentHidden()) {
      this.pausedByHidden = true;
      this.setState("idle");
      return;
    }
    this.openSocket();
  }

  disconnect(): void {
    this.shouldRun = false;
    this.pausedByHidden = false;
    this.clearReconnectTimer();
    this.abortTicket();
    this.uninstallVisibilityHandler();
    this.closeSocket();
    this.setState("closed");
  }

  send(msg: ClientMsg): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return false;
    try {
      this.ws.send(JSON.stringify(msg));
      return true;
    } catch (err) {
      this.cfg.onError?.(err);
      return false;
    }
  }

  private async openSocket(): Promise<void> {
    this.clearReconnectTimer();
    this.closeSocket();
    this.setState(this.attempt === 0 ? "connecting" : "reconnecting");

    const abort = new AbortController();
    this.ticketAbort = abort;

    let ticket: string;
    try {
      ticket = await this.cfg.fetchTicket(abort.signal);
    } catch (err) {
      if (abort.signal.aborted) return;
      this.cfg.onError?.(err);
      this.scheduleReconnect();
      return;
    } finally {
      if (this.ticketAbort === abort) this.ticketAbort = null;
    }

    if (!this.shouldRun || this.pausedByHidden) return;

    const url = this.buildUrl(ticket);
    let ws: WebSocket;
    try {
      ws = new WebSocket(url);
    } catch (err) {
      this.cfg.onError?.(err);
      this.scheduleReconnect();
      return;
    }
    this.ws = ws;

    ws.onopen = () => {
      this.attempt = 0;
      this.setState("open");
    };

    ws.onmessage = (ev) => {
      let parsed: unknown;
      try {
        parsed = JSON.parse(typeof ev.data === "string" ? ev.data : "");
      } catch (err) {
        this.cfg.onError?.(err);
        return;
      }
      if (isServerMsg(parsed)) this.cfg.onMessage(parsed);
    };

    ws.onerror = (ev) => {
      this.cfg.onError?.(ev);
    };

    ws.onclose = () => {
      this.ws = null;
      if (!this.shouldRun || this.pausedByHidden) {
        // intentional close (disconnect / hidden) — don't reconnect
        return;
      }
      this.scheduleReconnect();
    };
  }

  private scheduleReconnect(): void {
    if (!this.shouldRun || this.pausedByHidden) return;
    const delay = this.nextBackoff();
    this.attempt += 1;
    this.setState("reconnecting");
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      void this.openSocket();
    }, delay);
  }

  private nextBackoff(): number {
    const raw = Math.min(BACKOFF_CAP_MS, BACKOFF_BASE_MS * 2 ** this.attempt);
    const jitter = 1 + (Math.random() * 2 - 1) * BACKOFF_JITTER;
    return Math.max(BACKOFF_BASE_MS, Math.floor(raw * jitter));
  }

  private buildUrl(ticket: string): string {
    const sep = this.cfg.wsUrl.includes("?") ? "&" : "?";
    return `${this.cfg.wsUrl}${sep}ticket=${encodeURIComponent(ticket)}`;
  }

  private closeSocket(): void {
    const ws = this.ws;
    if (!ws) return;
    this.ws = null;
    // Detach handlers so a late close/error doesn't trigger reconnect.
    ws.onopen = null;
    ws.onmessage = null;
    ws.onerror = null;
    ws.onclose = null;
    try {
      ws.close();
    } catch {
      // ignore
    }
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private abortTicket(): void {
    if (this.ticketAbort) {
      this.ticketAbort.abort();
      this.ticketAbort = null;
    }
  }

  private setState(next: RealtimeState): void {
    if (this.state === next) return;
    this.state = next;
    this.cfg.onStateChange?.(next);
  }

  private isDocumentHidden(): boolean {
    return (
      typeof document !== "undefined" && document.visibilityState === "hidden"
    );
  }

  private installVisibilityHandler(): void {
    if (this.visibilityHandler || typeof document === "undefined") return;
    const handler = () => {
      if (!this.shouldRun) return;
      if (document.visibilityState === "hidden") {
        this.pausedByHidden = true;
        this.clearReconnectTimer();
        this.abortTicket();
        this.closeSocket();
        this.setState("idle");
      } else if (this.pausedByHidden) {
        this.pausedByHidden = false;
        this.attempt = 0;
        void this.openSocket();
      }
    };
    document.addEventListener("visibilitychange", handler);
    this.visibilityHandler = handler;
  }

  private uninstallVisibilityHandler(): void {
    if (!this.visibilityHandler || typeof document === "undefined") return;
    document.removeEventListener("visibilitychange", this.visibilityHandler);
    this.visibilityHandler = null;
  }
}
