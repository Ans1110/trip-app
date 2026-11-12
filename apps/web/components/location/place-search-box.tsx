"use client";

import { useEffect, useId, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronLeft, Loader2, MapPin, Search, X } from "lucide-react";

import { useSearchPlaces } from "@/hooks/location-hooks";
import type { PlaceSummary } from "@/lib/apis/location-api";
import { searchSchema, type SearchFormInput } from "@/lib/schemas/location-schemas";
import { cn } from "@/lib/utils";

export type SearchBias = { lat: number; lng: number; radius: number };

const MIN_CHARS = 3;
// One CJK character carries a whole syllable, so 台中 is already a useful query.
const CJK = /[\u3040-\u30ff\u3400-\u9fff\uac00-\ud7af]/;
const minChars = (q: string) => (CJK.test(q) ? 2 : MIN_CHARS);
const SUGGESTION_LIMIT = 6;

export function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const id = window.setTimeout(() => setV(value), ms);
    return () => window.clearTimeout(id);
  }, [value, ms]);
  return v;
}

export function usePlaceSuggestions(q: string, bias: SearchBias | null, enabled: boolean) {
  const debounced = useDebounced(q.trim(), 350);
  const active = enabled && debounced.length >= minChars(debounced);
  const query = useSearchPlaces(
    { q: debounced, lat: bias?.lat, lng: bias?.lng, radius: bias?.radius, limit: 20 },
    { enabled: active, staleTime: 5 * 60_000 },
  );
  return {
    active,
    pending: active && (query.isFetching || debounced !== q.trim()),
    items: active ? (query.data ?? []).slice(0, SUGGESTION_LIMIT) : [],
  };
}

type Props = {
  bias: SearchBias | null;
  displayValue: string;
  showBack: boolean;
  placeholder?: string;
  onSubmit: (q: string) => void;
  onPick: (place: PlaceSummary) => void;
  onBack: () => void;
  onClear: () => void;
};

export function PlaceSearchBox({
  bias,
  displayValue,
  showBack,
  placeholder = "Search places, food, sights…",
  onSubmit,
  onPick,
  onBack,
  onClear,
}: Props) {
  const form = useForm<SearchFormInput>({
    resolver: zodResolver(searchSchema),
    defaultValues: { q: displayValue },
  });
  const { setValue, watch, handleSubmit, register } = form;
  const q = watch("q") ?? "";
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const listId = useId();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const { ref: rhfRef, ...field } = register("q");

  useEffect(() => {
    setValue("q", displayValue);
  }, [displayValue, setValue]);

  const suggestions = usePlaceSuggestions(q, bias, open);
  const items = suggestions.items;

  useEffect(() => setActiveIndex(-1), [q]);

  const pick = (p: PlaceSummary) => {
    setOpen(false);
    setValue("q", p.name);
    inputRef.current?.blur();
    onPick(p);
  };

  const submit = handleSubmit((v) => {
    if (activeIndex >= 0 && items[activeIndex]) {
      pick(items[activeIndex]);
      return;
    }
    setOpen(false);
    inputRef.current?.blur();
    onSubmit(v.q);
  });

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowDown" && items.length) {
      e.preventDefault();
      setOpen(true);
      setActiveIndex((i) => (i + 1) % items.length);
    } else if (e.key === "ArrowUp" && items.length) {
      e.preventDefault();
      setActiveIndex((i) => (i <= 0 ? items.length - 1 : i - 1));
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  };

  const showList = open && suggestions.active && (items.length > 0 || suggestions.pending);

  return (
    <form onSubmit={submit} role="search" className="relative">
      <div
        className={cn(
          "flex items-center gap-1 h-12 pl-2 pr-1.5 bg-card shadow-soft-lg transition-[border-radius] duration-200",
          showList ? "rounded-t-3xl" : "rounded-full",
        )}
      >
        {showBack ? (
          <button
            type="button"
            onClick={onBack}
            aria-label="Back"
            className="grid place-items-center size-9 rounded-full text-muted-foreground hover:bg-muted hover:text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <ChevronLeft className="size-5" />
          </button>
        ) : (
          <span className="grid place-items-center size-9 text-muted-foreground" aria-hidden>
            <Search className="size-[18px]" />
          </span>
        )}
        <input
          {...field}
          ref={(el) => {
            rhfRef(el);
            inputRef.current = el;
          }}
          type="text"
          role="combobox"
          aria-label="Search places"
          aria-autocomplete="list"
          aria-expanded={showList}
          aria-controls={listId}
          aria-activedescendant={activeIndex >= 0 ? `${listId}-${activeIndex}` : undefined}
          autoComplete="off"
          enterKeyHint="search"
          placeholder={placeholder}
          onFocus={() => setOpen(true)}
          onBlur={(e) => {
            field.onBlur(e);
            window.setTimeout(() => setOpen(false), 120);
          }}
          onKeyDown={onKeyDown}
          className="flex-1 min-w-0 h-full bg-transparent text-[15px] text-foreground placeholder:text-subtle-foreground outline-none"
        />
        {suggestions.pending && <Loader2 className="size-4 mr-1 animate-spin text-subtle-foreground" aria-hidden />}
        {q && (
          <button
            type="button"
            aria-label="Clear search"
            onClick={() => {
              setValue("q", "");
              setOpen(false);
              onClear();
              inputRef.current?.focus();
            }}
            className="grid place-items-center size-9 rounded-full text-muted-foreground hover:bg-muted hover:text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <X className="size-[18px]" />
          </button>
        )}
      </div>

      {showList && (
        <ul
          id={listId}
          role="listbox"
          aria-label="Suggestions"
          className="absolute inset-x-0 top-full z-10 bg-card rounded-b-3xl shadow-soft-lg pb-2 border-t border-border overflow-hidden"
        >
          {items.length === 0 && (
            <li className="px-5 py-3 text-sm text-muted-foreground" role="presentation">
              Searching…
            </li>
          )}
          {items.map((p, i) => (
            <li
              key={p.place_id}
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === activeIndex}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => pick(p)}
              onMouseEnter={() => setActiveIndex(i)}
              className={cn(
                "flex items-center gap-3 px-4 py-2.5 cursor-pointer",
                i === activeIndex && "bg-muted",
              )}
            >
              <span className="grid place-items-center size-8 rounded-full bg-pink-soft text-pink-ink shrink-0">
                <MapPin className="size-4" />
              </span>
              <span className="min-w-0">
                <span className="block text-sm font-medium text-foreground truncate">{p.name}</span>
                {p.address && <span className="block text-xs text-muted-foreground truncate">{p.address}</span>}
              </span>
            </li>
          ))}
        </ul>
      )}
    </form>
  );
}
