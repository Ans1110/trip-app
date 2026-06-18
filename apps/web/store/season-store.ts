import { create } from "zustand";

import { seasons, type Season, type SeasonPalette } from "../lib/seasons";

type SeasonState = {
  active: number;
  setActive: (i: number) => void;
};

export const useSeasonStore = create<SeasonState>((set) => ({
  active: 0,
  setActive: (i) => set({ active: i }),
}));

export function useSeason(): {
  active: number;
  setActive: (i: number) => void;
  season: Season;
  palette: SeasonPalette;
  seasons: Season[];
} {
  const active = useSeasonStore((s) => s.active);
  const setActive = useSeasonStore((s) => s.setActive);
  const season = seasons[active];
  return { active, setActive, season, palette: season.palette, seasons };
}
