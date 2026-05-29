import "server-only";
import {
  circuitOpenError,
  classifyHttpError,
  networkError,
  NormalizedError,
  timeoutError,
} from "./errors";
import { RequestContext } from "./request-context";
import { Logger, rootLogger } from "./logger";
import { CircuitBreaker, CircuitBreakerOptions } from "./circuit-breaker";
import { CSRF_COOKIE, UPSTREAM_API_BASE, UPSTREAM_TIMEOUT_MS } from "./config";

export type Envelope<T> = {
  code: number;
  message: string;
  data?: T;
};

type ResultBase = {
  status: number;
  payload: unknown;
  raw: string;
};

export type ApiOk<T> = ResultBase & {
  ok: true;
  data: T;
  envelope?: Envelope<T>;
};

export type ApiErr = ResultBase & {
  ok: false;
  code: number;
  message: string;
  envelope?: Envelope<unknown>;
  error: NormalizedError;
};

export type ApiResult<T> = ApiOk<T> | ApiErr;

export type RetryOutCome = {
  result: ApiResult<unknown>;
  threw: boolean;
  error?: unknown;
};

export type RetryDecision = boolean | { retry: boolean; reason?: string };

export type RetryPredicate = (
  outcome: RetryOutCome,
  attempt: number,
) => RetryDecision;

export type RetryConfig = {
  attempts?: number;
  baseBackoffMs?: number;
  maxBackoffMs?: number;
  jitter?: boolean;
  retryOn?: RetryPredicate;
};

export type HttpOptions = {
  ctx: RequestContext;
  accessToken?: string | null;
  csrfToken?: string | null;
  extraHeaders?: Record<string, string>;
  signal?: AbortSignal;
  timeoutMs?: number;
  retry?: RetryConfig | false;
};

type ResolvedRetry = Required<RetryConfig>;

const IDEMPOTENT_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);
const RETRYABLE_STATUSES = new Set([408, 425, 429, 502, 503, 504]);

const DEFAULT_BASE_BACKOFF_MS = 200;
const DEFAULT_MAX_BACKOFF_MS = 2_000;

export type HttpClientOptions = {
  baseUrl?: string;
  logger?: Logger;
  breaker?: CircuitBreaker | CircuitBreakerOptions | false;
};

export class HttpClient {
  private readonly baseUrl: string;
  private readonly logger: Logger;
  private readonly breaker?: CircuitBreaker;

  constructor(options: HttpClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? UPSTREAM_API_BASE;
    this.logger = options.logger ?? rootLogger.child({ module: "http-client" });
    this.breaker = resolveBreaker(options.breaker, (from, to, reason) => {
      this.logger.warn("upstream.breaker_state", { from, to, reason });
    });
  }

  get<T>(path: string, opts: HttpOptions): Promise<ApiResult<T>> {
    return this.request("GET", path, undefined, opts);
  }
  post<T>(
    path: string,
    body: unknown,
    opts: HttpOptions,
  ): Promise<ApiResult<T>> {
    return this.request("POST", path, body, opts);
  }
  put<T>(
    path: string,
    body: unknown,
    opts: HttpOptions,
  ): Promise<ApiResult<T>> {
    return this.request("PUT", path, body, opts);
  }
  patch<T>(
    path: string,
    body: unknown,
    opts: HttpOptions,
  ): Promise<ApiResult<T>> {
    return this.request("PATCH", path, body, opts);
  }
  delete<T>(path: string, opts: HttpOptions): Promise<ApiResult<T>> {
    return this.request("DELETE", path, undefined, opts);
  }

  async request<T>(
    method: string,
    path: string,
    body: unknown,
    opts: HttpOptions,
  ): Promise<ApiResult<T>> {
    const url = `${this.baseUrl}/${path.replace(/^\/+/, "")}`;
    const acquire = this.breaker?.acquire();
    if (acquire && !acquire.ok) {
      const errResult = makeApiErr(circuitOpenError());
      this.logger.warn("upstream.circuit_open", {
        method,
        path,
        request_id: opts.ctx.requestId,
        retry_after_ms: acquire.retryAfterMs,
      });
      return errResult as ApiResult<T>;
    }
    const probing = acquire?.probing ?? false;

    const retry = resolveRetry(method, opts.retry);
    const timeoutMs = opts.timeoutMs ?? UPSTREAM_TIMEOUT_MS;
    const headers = this.buildHeaders(opts, body !== undefined);
    const serialized = serializeBody(body);

    let lastOutcome: RetryOutCome | undefined;

    for (let attempt = 1; attempt <= retry.attempts; attempt++) {
      const signal = attemptSignal(opts.signal, timeoutMs);
      const start = Date.now();
      const outcome = await this.attemptOnce<T>(
        method,
        url,
        headers,
        serialized,
        signal,
      );
      const duration = Date.now() - start;

      this.logger.info("upstream.request", {
        method,
        path,
        status: outcome.result.status,
        attempt,
        duration_ms: duration,
        probing,
        request_id: opts.ctx.requestId,
        upstream_code: outcome.result.ok
          ? outcome.result.envelope?.code
          : outcome.result.code,
        err_category: outcome.result.ok
          ? undefined
          : outcome.result.error.category,
        threw: outcome.threw || undefined,
      });

      lastOutcome = outcome;
      if (attempt >= retry.attempts) break;
      const decision = retry.retryOn(outcome, attempt);
      if (!shouldRetry(decision)) break;

      const delay = computeBackoff(
        attempt,
        retry.baseBackoffMs,
        retry.maxBackoffMs,
        retry.jitter,
      );
      this.logger.warn("upstream.retry", {
        method,
        path,
        attempt,
        next_attempt: attempt + 1,
        delay_ms: delay,
        request_id: opts.ctx.requestId,
        reason: retryReason(decision, outcome),
      });
      await sleep(delay, opts.signal);
    }

    const final = lastOutcome!;
    this.recordBreaker(final, probing);
    return final.result as ApiResult<T>;
  }

  async fetchCsrfToken(ctx: RequestContext): Promise<string | null> {
    const r = await this.get<{ csrfToken: string }>("/csrf", { ctx });
    if (!r.ok) return null;
    const token = r.data?.csrfToken;
    return typeof token === "string" ? token : null;
  }

  private async attemptOnce<T>(
    method: string,
    url: string,
    headers: Headers,
    body: BodyInit | undefined,
    signal: AbortSignal,
  ): Promise<RetryOutCome> {
    try {
      const raw = await fetch(url, {
        method,
        headers,
        body,
        cache: "no-cache",
        redirect: "manual",
        signal,
      });
      const text = await raw.text();
      const parsed = parseJson(text);
      return { result: buildResult<T>(raw.status, parsed, text), threw: false };
    } catch (err) {
      const timeOut = isAbortError(err);
      const normalized = timeOut ? timeoutError() : networkError(err);
      return {
        result: makeApiErr(normalized),
        threw: true,
        error: err,
      };
    }
  }

  private recordBreaker(outcome: RetryOutCome, probing: boolean): void {
    if (!this.breaker) return;
    const result = outcome.result;
    if (result.ok) {
      this.breaker.recordSuccess(probing);
      return;
    }
    if (outcome.threw || result.status >= 500) {
      this.breaker.recordFailure(probing, result.error.category);
      return;
    }
    this.breaker.recordSuccess(probing);
  }

  private buildHeaders(opts: HttpOptions, hasBody: boolean): Headers {
    const h = new Headers();
    if (hasBody) h.set("content-type", "application/json");
    h.set("accept", "application/json");
    h.set("x-request-id", opts.ctx.requestId);
    if (opts.ctx.userAgent) h.set("user-agent", opts.ctx.userAgent);
    if (opts.ctx.clientIp) h.set("x-forwarded-for", opts.ctx.clientIp);
    if (opts.ctx.deviceName) h.set("x-device-name", opts.ctx.deviceName);
    if (opts.ctx.deviceType) h.set("x-device-type", opts.ctx.deviceType);
    if (opts.accessToken) h.set("authorization", `Bearer ${opts.accessToken}`);
    if (opts.csrfToken) {
      h.set("x-csrf-token", opts.csrfToken);
      h.set("cookie", `${CSRF_COOKIE}=${opts.csrfToken}`);
    }
    if (opts.extraHeaders) {
      for (const [k, v] of Object.entries(opts.extraHeaders)) h.set(k, v);
    }
    return h;
  }
}

const resolveBreaker = (
  cfg: CircuitBreaker | CircuitBreakerOptions | false | undefined,
  onStateChange: NonNullable<CircuitBreakerOptions["onStateChange"]>,
): CircuitBreaker | undefined => {
  if (cfg === false) return undefined;
  if (cfg instanceof CircuitBreaker) return cfg;
  return new CircuitBreaker({ ...(cfg ?? {}), onStateChange });
};

const makeApiErr = (error: NormalizedError): ApiErr => ({
  ok: false,
  status: error.status ?? 500,
  code: error.code ?? "UNKNOWN_ERROR",
  message: error.message,
  error,
  payload: null,
  raw: "",
});

const resolveRetry = (
  method: string,
  cfg: RetryConfig | false | undefined,
): ResolvedRetry => {
  if (cfg === false) {
    return {
      attempts: 1,
      baseBackoffMs: DEFAULT_BASE_BACKOFF_MS,
      maxBackoffMs: DEFAULT_MAX_BACKOFF_MS,
      jitter: true,
      retryOn: () => false,
    };
  }
  const defaultAttempts = IDEMPOTENT_METHODS.has(method) ? 2 : 1;
  return {
    attempts: Math.max(1, cfg?.attempts ?? defaultAttempts),
    baseBackoffMs: cfg?.baseBackoffMs ?? DEFAULT_BASE_BACKOFF_MS,
    maxBackoffMs: cfg?.maxBackoffMs ?? DEFAULT_MAX_BACKOFF_MS,
    jitter: cfg?.jitter ?? true,
    retryOn: cfg?.retryOn ?? defaultRetryPredicate(method),
  };
};

const defaultRetryPredicate = (method: string): RetryPredicate => {
  const isIdempotent = IDEMPOTENT_METHODS.has(method);
  return ({ result, threw }) => {
    if (!isIdempotent) return false;
    if (threw) return true;
    return RETRYABLE_STATUSES.has(result.status);
  };
};

const shouldRetry = (decision: RetryDecision): boolean => {
  return typeof decision === "boolean" ? decision : decision.retry;
};

const retryReason = (
  decision: RetryDecision,
  outcome: RetryOutCome,
): string => {
  if (typeof decision === "object" && decision.reason) return decision.reason;
  if (outcome.threw)
    return isAbortError(outcome.error) ? "timeout" : "network_error";
  return `status_${outcome.result.status}`;
};

const attemptSignal = (
  userSignal: AbortSignal | undefined,
  timeoutMs: number,
): AbortSignal => {
  const timeout = AbortSignal.timeout(timeoutMs);
  if (!userSignal) return timeout;
  const anyFn = (
    AbortSignal as unknown as { any?: (signals: AbortSignal[]) => AbortSignal }
  ).any;
  return typeof anyFn === "function" ? anyFn([userSignal, timeout]) : timeout;
};

const computeBackoff = (
  attempt: number,
  base: number,
  max: number,
  jitter: boolean,
): number => {
  const exp = Math.min(max, base * 2 ** (attempt - 1));
  return jitter ? Math.floor(Math.random() * exp) : exp;
};

const isAbortError = (error: unknown): boolean => {
  return (
    error instanceof Error &&
    (error.name === "AbortError" || error.message === "TimeoutError")
  );
};

const sleep = (ms: number, signal?: AbortSignal): Promise<void> => {
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    const onAbort = () => {
      clearTimeout(timer);
      resolve();
    };
    signal?.addEventListener("abort", onAbort, { once: true });
  });
};

const serializeBody = (body: unknown): BodyInit | undefined => {
  if (body === undefined || body == null) return undefined;
  if (typeof body === "string") return body;
  if (body instanceof ArrayBuffer || ArrayBuffer.isView(body))
    return body as BodyInit;
  return JSON.stringify(body);
};

const parseJson = (text: string): unknown => {
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
};

const isEnvelope = (v: unknown): v is Envelope<unknown> => {
  return (
    !!v &&
    typeof v === "object" &&
    "code" in v &&
    "message" in v &&
    typeof (v as { message: unknown }).message === "string"
  );
};

const buildResult = <T>(
  status: number,
  parsed: unknown,
  raw: string,
): ApiResult<T> => {
  const ok = status >= 200 && status < 300;
  const envelope = isEnvelope(parsed) ? (parsed as Envelope<T>) : undefined;
  const data = (envelope ? envelope.data : parsed) as T;

  if (ok) {
    return {
      ok: true,
      status,
      data,
      envelope,
      payload: parsed,
      raw,
    };
  }
  const error = classifyHttpError(status, envelope, raw);
  return {
    ok: false,
    status,
    code: error.code,
    message: error.message,
    envelope: envelope as Envelope<unknown> | undefined,
    error,
    payload: parsed,
    raw,
  };
};

export const httpClient = new HttpClient();
