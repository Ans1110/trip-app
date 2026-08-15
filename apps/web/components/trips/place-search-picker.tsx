"use client";

import { useEffect, useRef, useState } from "react";
import { Loader2, Search } from "lucide-react";

import { useSearchPlaces } from "@/hooks/location-hooks";

export type PickedPlace = {
  name: string;
  address?: string;
  lat: number;
  lng: number;
  place_id: string;
};

export function PlaceSearchPicker({
  onPick,
  placeholder = "Find a place",
}: {
  onPick: (p: PickedPlace) => void;
  placeholder?: string;
}) {
  const [draft, setDraft] = useState("");
  const [debounced, setDebounced] = useState("");
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const t = setTimeout(() => {
      const q = draft.trim();
      setDebounced(q);
      if (q) setOpen(true);
    }, 250);
    return () => clearTimeout(t);
  }, [draft]);

  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      if (!wrapRef.current) return;
      if (!wrapRef.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, []);

  const search = useSearchPlaces({ q: debounced, limit: 6 });

  return (
    <div ref={wrapRef} className="relative flex flex-col gap-1.5">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 size-4 text-[#8B9A8E]" />
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onFocus={() => {
            if (debounced) setOpen(true);
          }}
          onKeyDown={(e) => {
            if (e.key === "Escape") setOpen(false);
          }}
          placeholder={placeholder}
          className="w-full h-9 pl-9 pr-9 rounded-lg bg-[#0B100D] text-sm text-[#ECEFEA] border border-[#1F2A24] focus:outline-none focus:border-white/20"
        />
        {search.isFetching && debounced && (
          <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 size-4 animate-spin text-[#8B9A8E]" />
        )}
      </div>

      {open && debounced && (
        <div className="absolute top-full left-0 right-0 mt-1 z-30 rounded-lg bg-[#121814] border border-[#1F2A24] shadow-lg max-h-64 overflow-auto">
          {search.isLoading && (
            <p className="text-xs px-3 py-2 text-[#8B9A8E]">Searching…</p>
          )}
          {search.isError && (
            <p className="text-xs px-3 py-2 text-[#FCA5A5]">Search failed.</p>
          )}
          {!search.isLoading &&
            !search.isError &&
            (search.data?.length ?? 0) === 0 && (
              <p className="text-xs px-3 py-2 text-[#8B9A8E]">No matches.</p>
            )}
          {search.data?.map((p) => (
            <button
              type="button"
              key={p.place_id}
              onClick={() => {
                onPick({
                  name: p.name,
                  address: p.address,
                  lat: p.location.lat,
                  lng: p.location.lng,
                  place_id: p.place_id,
                });
                setDraft("");
                setDebounced("");
                setOpen(false);
              }}
              className="w-full text-left px-3 py-2 hover:bg-white/5 border-b border-[#1F2A24] last:border-b-0"
            >
              <p className="text-sm text-[#ECEFEA] truncate">{p.name}</p>
              {p.address && (
                <p className="text-xs text-[#8B9A8E] truncate">{p.address}</p>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
