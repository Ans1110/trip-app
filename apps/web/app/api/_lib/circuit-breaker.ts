import "server-only";

export type BreakerState = "closed" | "open" | "half_open";

export type CircuitBreakerOptions = {
  failureThreshold?: number;
  rollingWindowMs?: number;
  recoveryTimeoutMs?: number;
  halfOpenMaxConcurrent?: number;
  halfOpenSuccessThreshold?: number;
  onStateChange?: (
    from: BreakerState,
    to: BreakerState,
    reason: string,
  ) => void;
};

export type AcquireResult =
  | { ok: true; probing: boolean }
  | { ok: false; reason: "open"; retryAfterMs: number };

const DEFAULTS = {
  failureThreshold: 5,
  rollingWindowMs: 10_000,
  recoveryTimeoutMs: 30_000,
  halfOpenMaxConcurrent: 1,
  halfOpenSuccessThreshold: 2,
};

type Config = Required<Omit<CircuitBreakerOptions, "onStateChange">>;

export class CircuitBreaker {
  private state: BreakerState = "closed";
  private failures: number[] = [];
  private openAt = 0;
  private halfOpenInflight = 0;
  private halfOpenSuccesses = 0;
  private readonly cfg: Config;
  private readonly onStateChange?: CircuitBreakerOptions["onStateChange"];
  private readonly clock: () => number;

  constructor(
    opts: CircuitBreakerOptions = {},
    clock: () => number = () => Date.now(),
  ) {
    this.cfg = {
      failureThreshold: opts.failureThreshold ?? DEFAULTS.failureThreshold,
      rollingWindowMs: opts.rollingWindowMs ?? DEFAULTS.rollingWindowMs,
      recoveryTimeoutMs: opts.recoveryTimeoutMs ?? DEFAULTS.recoveryTimeoutMs,
      halfOpenMaxConcurrent:
        opts.halfOpenMaxConcurrent ?? DEFAULTS.halfOpenMaxConcurrent,
      halfOpenSuccessThreshold:
        opts.halfOpenSuccessThreshold ?? DEFAULTS.halfOpenSuccessThreshold,
    };
    this.onStateChange = opts.onStateChange;
    this.clock = clock;
  }

  getState(): BreakerState {
    return this.state;
  }

  acquire(): AcquireResult {
    const now = this.clock();
    if (this.state === "open") {
      const elapsed = now - this.openAt;
      if (elapsed < this.cfg.recoveryTimeoutMs) {
        return {
          ok: false,
          reason: "open",
          retryAfterMs: this.cfg.recoveryTimeoutMs - elapsed,
        };
      }
      this.transition("half_open", "recovery_timeout_elapsed");
      this.halfOpenInflight = 0;
      this.halfOpenSuccesses = 0;
    }
    if (this.state === "half_open") {
      if (this.halfOpenInflight >= this.cfg.halfOpenMaxConcurrent) {
        return {
          ok: false,
          reason: "open",
          retryAfterMs: this.cfg.recoveryTimeoutMs,
        };
      }
      this.halfOpenInflight++;
      return { ok: true, probing: true };
    }
    return { ok: true, probing: false };
  }

  recordSuccess(probing: boolean): void {
    if (this.state !== "half_open" || !probing) return;
    this.halfOpenInflight = Math.max(0, this.halfOpenInflight - 1);
    this.halfOpenSuccesses++;
    if (this.halfOpenSuccesses >= this.cfg.halfOpenSuccessThreshold) {
      this.failures = [];
      this.transition("closed", "half_open_success");
    }
  }

  recordFailure(probing: boolean, reason: string): void {
    const now = this.clock();
    if (this.state === "half_open" && probing) {
      this.halfOpenInflight = Math.max(0, this.halfOpenInflight - 1);
      this.openAt = now;
      this.transition("open", `half_open_failure: ${reason}`);
      return;
    }
    this.failures.push(now);
    const cutoff = now - this.cfg.rollingWindowMs;
    this.failures = this.failures.filter((t) => t >= cutoff);
    if (this.failures.length >= this.cfg.failureThreshold) {
      this.openAt = now;
      this.transition("open", `failure_threshold_exceeded: ${reason}`);
    }
  }

  private transition(to: BreakerState, reason: string): void {
    const from = this.state;
    if (from == to) return;
    this.state = to;
    this.onStateChange?.(from, to, reason);
  }
}
