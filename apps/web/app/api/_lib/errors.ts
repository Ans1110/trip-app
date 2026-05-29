import "server-only";

type EnvelopeLike = { code: number; message: string };

export type ErrorCategory =
  | "network"
  | "timeout"
  | "circuit_open"
  | "validation"
  | "auth"
  | "not_found"
  | "conflict"
  | "rate_limit"
  | "service_unavailable"
  | "upstream_4xx"
  | "upstream_5xx"
  | "unknown";

export type NormalizedError = {
  category: ErrorCategory;
  status: number;
  code: number;
  message: string;
  retryable: boolean;
  upstream?: { code: number; message: string };
  raw?: string;
};

export function classifyHttpError(
  status: number,
  envelope: EnvelopeLike | undefined,
  raw: string,
): NormalizedError {
  const upstream =
    envelope && typeof envelope.code === "number"
      ? { code: envelope.code, message: envelope.message }
      : undefined;
  const code = envelope?.code ?? status;
  const message = envelope?.message ?? defaultStatusMessage(status, raw);
  return {
    category: httpCategory(status),
    status,
    code,
    message,
    retryable: isRetryableStatus(status),
    upstream,
    raw: envelope ? undefined : raw || undefined,
  };
}

export function timeoutError(): NormalizedError {
  return {
    category: "timeout",
    status: 504,
    code: 504,
    message: "upstream timeout",
    retryable: true,
  };
}

export function networkError(err: unknown): NormalizedError {
  const detail = err instanceof Error ? err.message : "upstream unavailable";
  return {
    category: "network",
    status: 502,
    code: 502,
    message: `upstream unavailable: ${detail}`,
    retryable: true,
  };
}

export function circuitOpenError(): NormalizedError {
  return {
    category: "circuit_open",
    status: 503,
    code: 503,
    message: "upstream circuit open",
    retryable: true,
  };
}

function httpCategory(status: number): ErrorCategory {
  if (status === 400) return "validation";
  if (status === 401 || status === 403) return "auth";
  if (status === 404) return "not_found";
  if (status === 409) return "conflict";
  if (status === 429) return "rate_limit";
  if (status === 503) return "service_unavailable";
  if (status >= 500) return "upstream_5xx";
  if (status >= 400) return "upstream_4xx";
  return "unknown";
}

function isRetryableStatus(status: number): boolean {
  return status === 408 || status === 425 || status === 429 || status >= 500;
}

function defaultStatusMessage(status: number, raw: string): string {
  if (status === 400) return "bad request";
  if (status === 401) return "unauthorized";
  if (status === 403) return "forbidden";
  if (status === 404) return "not found";
  if (status === 409) return "conflict";
  if (status === 429) return "too many requests";
  if (status >= 500) return "upstream error";
  return raw || `http ${status}`;
}
