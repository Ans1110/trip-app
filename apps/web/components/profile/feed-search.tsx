"use client";

import { Search as SearchIcon } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";

import { FeedView, type FeedTab } from "@/components/profile/feed-view";

const TABS: { key: FeedTab; label: string }[] = [
  { key: "all", label: "All" },
  { key: "following", label: "Following" },
  { key: "bookmarks", label: "Bookmarks" },
];

const parseTab = (raw: string | null): FeedTab => {
  if (raw === "all" || raw === "following" || raw === "bookmarks") return raw;
  return "following";
};

export function FeedSearch() {
  return (
    <Suspense fallback={<FeedView />}>
      <FeedSearchInner />
    </Suspense>
  );
}

function FeedSearchInner() {
  const router = useRouter();
  const params = useSearchParams();
  const urlQ = (params.get("q") ?? "").trim();
  const tab = parseTab(params.get("tab"));
  const [input, setInput] = useState(urlQ);

  const lastPushed = useRef(urlQ);
  useEffect(() => {
    if (urlQ !== lastPushed.current) {
      lastPushed.current = urlQ;
      setInput(urlQ);
    }
  }, [urlQ]);

  useEffect(() => {
    const next = input.trim();
    if (next === lastPushed.current) return;
    const t = setTimeout(() => {
      lastPushed.current = next;
      const qs = new URLSearchParams();
      if (next) qs.set("q", next);
      if (tab !== "following") qs.set("tab", tab);
      const s = qs.toString();
      router.replace(s ? `/feed?${s}` : "/feed");
    }, 250);
    return () => clearTimeout(t);
  }, [input, router, tab]);

  const selectTab = (next: FeedTab) => {
    if (next === tab) return;
    const qs = new URLSearchParams();
    if (urlQ) qs.set("q", urlQ);
    if (next !== "following") qs.set("tab", next);
    const s = qs.toString();
    router.replace(s ? `/feed?${s}` : "/feed");
  };

  return (
    <div>
      <div
        className="mb-6 flex items-center gap-2 rounded-full px-4 py-2"
        style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
      >
        <SearchIcon className="size-4" style={{ color: "#8B9A8E" }} />
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Search posts, tags, or travelers…"
          className="flex-1 bg-transparent outline-none text-sm"
          style={{ color: "#ECEFEA" }}
          aria-label="Search feed"
          autoComplete="off"
        />
        {input && (
          <button
            type="button"
            onClick={() => setInput("")}
            className="text-xs px-2 py-0.5 rounded-full hover:bg-white/5"
            style={{ color: "#8B9A8E" }}
          >
            Clear
          </button>
        )}
      </div>

      {!urlQ && (
        <div
          className="mb-6 inline-flex items-center gap-1 p-1 rounded-full"
          style={{ border: "1px solid #1F2A24" }}
          role="tablist"
          aria-label="Feed tabs"
        >
          {TABS.map((t) => {
            const active = t.key === tab;
            return (
              <button
                key={t.key}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => selectTab(t.key)}
                className="text-xs px-3 py-1.5 rounded-full season-transition"
                style={{
                  backgroundColor: active
                    ? "color-mix(in srgb, var(--season-accent) 18%, transparent)"
                    : "transparent",
                  color: active ? "var(--season-accent)" : "#8B9A8E",
                }}
              >
                {t.label}
              </button>
            );
          })}
        </div>
      )}

      <FeedView query={urlQ} tab={tab} />
    </div>
  );
}
