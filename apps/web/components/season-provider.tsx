"use client";

import { useEffect, type ReactNode } from "react";

import { useSeasonStore } from "@/store/season-store";
import { seasons } from "@/lib/seasons";

export { useSeason } from "@/store/season-store";

const AUTO_ROTATE_MS = 30 * 60 * 1000;

export function SeasonProvider({ children }: { children: ReactNode }) {
  const active = useSeasonStore((s) => s.active);
  const setActive = useSeasonStore((s) => s.setActive);

  useEffect(() => {
    const palette = seasons[active].palette;
    const root = document.documentElement;
    root.style.setProperty("--season-accent", palette.accent);
    root.style.setProperty("--season-heading", palette.heading);
    root.style.setProperty("--season-body", palette.body);
    root.style.setProperty("--season-button", palette.button);
    root.style.setProperty("--season-button-shadow", palette.buttonShadow);
  }, [active]);

  useEffect(() => {
    const id = window.setTimeout(() => {
      setActive((active + 1) % seasons.length);
    }, AUTO_ROTATE_MS);
    return () => window.clearTimeout(id);
  }, [active, setActive]);

  return <>{children}</>;
}
