import { describe, expect, test } from "bun:test";

import type { PlaceHours } from "@/lib/apis/location-api";
import { openStatus } from "./place-hours";

// 2026-11-11 is a Wednesday (day 3).
const at = (iso: string) => Date.parse(iso);

const weekdays9to18: PlaceHours = {
  utc_offset_minutes: 540,
  periods: [1, 2, 3, 4, 5].map((day) => ({
    open: { day, hour: 9, minute: 0 },
    close: { day, hour: 18, minute: 0 },
  })),
};

describe("openStatus", () => {
  test("returns null without hours", () => {
    expect(openStatus(undefined)).toBeNull();
  });

  test("open during the day in the place's timezone", () => {
    // 03:00Z = 12:00 in UTC+9
    const s = openStatus(weekdays9to18, at("2026-11-11T03:00:00Z"));
    expect(s).toEqual({ open: true, tone: "open", label: "Open", detail: "Closes 18:00" });
  });

  test("closing soon within the last hour", () => {
    const s = openStatus(weekdays9to18, at("2026-11-11T08:30:00Z"));
    expect(s?.tone).toBe("closing");
    expect(s?.detail).toBe("18:00");
  });

  test("closed before opening the same day", () => {
    const s = openStatus(weekdays9to18, at("2026-11-10T22:00:00Z")); // Wed 07:00 local
    expect(s).toEqual({ open: false, tone: "closed", label: "Closed", detail: "Opens 09:00" });
  });

  test("closed on the weekend names the next opening day", () => {
    const s = openStatus(weekdays9to18, at("2026-11-14T03:00:00Z")); // Sat 12:00 local
    expect(s?.detail).toBe("Opens Mon 09:00");
  });

  test("overnight period spanning midnight", () => {
    const bar: PlaceHours = {
      utc_offset_minutes: 0,
      periods: [{ open: { day: 5, hour: 20, minute: 0 }, close: { day: 6, hour: 2, minute: 0 } }],
    };
    expect(openStatus(bar, at("2026-11-14T01:00:00Z"))?.open).toBe(true); // Sat 01:00
    expect(openStatus(bar, at("2026-11-14T03:00:00Z"))?.open).toBe(false);
  });

  test("period wrapping the end of the week", () => {
    const late: PlaceHours = {
      utc_offset_minutes: 0,
      periods: [{ open: { day: 6, hour: 22, minute: 0 }, close: { day: 0, hour: 3, minute: 0 } }],
    };
    expect(openStatus(late, at("2026-11-15T01:00:00Z"))?.open).toBe(true); // Sun 01:00
  });

  test("always open", () => {
    const allDay: PlaceHours = { utc_offset_minutes: 0, periods: [{ open: { day: 0, hour: 0, minute: 0 } }] };
    expect(openStatus(allDay)?.label).toBe("Open 24 hours");
  });
});
