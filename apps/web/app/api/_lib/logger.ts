import "server-only";

type Fields = Record<string, unknown>;
type Level = "debug" | "info" | "warn" | "error";

const LEVELS: Record<Level, number> = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40,
};

const envLevel = (process.env.BFF_LOG_LEVEL ?? "info") as Level;
const MIN_LEVEL = LEVELS[envLevel] ?? LEVELS.info;

export type Logger = {
  child(fields: Fields): Logger;
  debug(msg: string, fields?: Fields): void;
  info(msg: string, fields?: Fields): void;
  warn(msg: string, fields?: Fields): void;
  error(msg: string, fields?: Fields): void;
};

const emit = (
  level: Level,
  base: Fields,
  msg: string,
  fields?: Fields,
): void => {
  if (LEVELS[level] < MIN_LEVEL) return;
  const record = {
    level,
    time: new Date().toISOString(),
    msg,
    ...base,
    ...fields,
  };
  const line = safeStringify(record);
  if (level === "error" || level === "warn") {
    console.error(line);
  } else {
    console.log(line);
  }
};

const safeStringify = (obj: unknown): string => {
  try {
    return JSON.stringify(obj);
  } catch {
    return JSON.stringify({ msg: "log_serialization_error" });
  }
};

export const createLogger = (base: Fields = {}): Logger => {
  return {
    child(fields) {
      return createLogger({ ...base, ...fields });
    },
    debug(msg, fields) {
      emit("debug", base, msg, fields);
    },
    info(msg, fields) {
      emit("info", base, msg, fields);
    },
    warn(msg, fields) {
      emit("warn", base, msg, fields);
    },
    error(msg, fields) {
      emit("error", base, msg, fields);
    },
  };
};

export const rootLogger: Logger = createLogger({ service: "web-bff" });
