import type { PlaceHours, PlaceHoursPoint } from "@/lib/apis/location-api";

const WEEK = 7 * 24 * 60;
const DAY_NAMES = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export type OpenStatus = {
  open: boolean;
  tone: "open" | "closing" | "closed";
  label: string;
  detail?: string;
};

const toWeekMinute = (p: PlaceHoursPoint) => p.day * 1440 + p.hour * 60 + p.minute;

const fmtTime = (weekMinute: number) => {
  const m = ((weekMinute % 1440) + 1440) % 1440;
  return `${String(Math.floor(m / 60)).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`;
};

const dayOf = (weekMinute: number) => Math.floor((((weekMinute % WEEK) + WEEK) % WEEK) / 1440);

/** Evaluates opening hours in the place's own timezone (via its UTC offset). */
export function openStatus(hours: PlaceHours | undefined, nowMs = Date.now()): OpenStatus | null {
  if (!hours || hours.periods.length === 0) return null;

  if (hours.periods.length === 1 && !hours.periods[0].close) {
    const p = hours.periods[0].open;
    if (p.day === 0 && p.hour === 0 && p.minute === 0) {
      return { open: true, tone: "open", label: "Open 24 hours" };
    }
  }

  const local = new Date(nowMs + hours.utc_offset_minutes * 60_000);
  const now = local.getUTCDay() * 1440 + local.getUTCHours() * 60 + local.getUTCMinutes();

  let closesIn: number | null = null;
  let opensIn: number | null = null;
  let opensAt = 0;

  for (const { open, close } of hours.periods) {
    const start = toWeekMinute(open);
    let end = close ? toWeekMinute(close) : start + WEEK;
    if (end <= start) end += WEEK;

    for (const shift of [0, WEEK]) {
      const t = now + shift;
      if (t >= start && t < end) {
        const left = end - t;
        if (closesIn === null || left > closesIn) closesIn = left;
      }
    }
    const until = (((start - now) % WEEK) + WEEK) % WEEK;
    if (until > 0 && (opensIn === null || until < opensIn)) {
      opensIn = until;
      opensAt = start;
    }
  }

  if (closesIn !== null) {
    if (closesIn >= WEEK) return { open: true, tone: "open", label: "Open 24 hours" };
    const at = fmtTime(now + closesIn);
    return closesIn <= 60
      ? { open: true, tone: "closing", label: "Closes soon", detail: at }
      : { open: true, tone: "open", label: "Open", detail: `Closes ${at}` };
  }

  if (opensIn === null) return { open: false, tone: "closed", label: "Closed" };
  const sameDay = dayOf(now) === dayOf(opensAt) && opensIn < 1440;
  const at = fmtTime(opensAt);
  return {
    open: false,
    tone: "closed",
    label: "Closed",
    detail: sameDay ? `Opens ${at}` : `Opens ${DAY_NAMES[dayOf(opensAt)]} ${at}`,
  };
}
