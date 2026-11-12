"use client";

import { useEffect, useId, useRef, useState } from "react";
import Link from "next/link";
import { ArrowRight, Compass, Menu, X } from "lucide-react";

import { useMe } from "@/hooks/auth-hooks";
import { UserMenu } from "@/components/user-menu";
import { cn } from "@/lib/utils";

const SECTIONS = [
  { id: "features", label: "How it feels" },
  { id: "preview", label: "Workspace" },
] as const;

const NAV_OFFSET = 96;

function useLandingScroll() {
  const [solid, setSolid] = useState(false);
  const [active, setActive] = useState<string | null>(null);

  useEffect(() => {
    let frame = 0;
    const measure = () => {
      frame = 0;
      const first = document.getElementById(SECTIONS[0].id);
      setSolid(first ? first.getBoundingClientRect().top <= NAV_OFFSET : window.scrollY > 0);

      let current: string | null = null;
      for (const s of SECTIONS) {
        const rect = document.getElementById(s.id)?.getBoundingClientRect();
        if (rect && rect.top <= NAV_OFFSET + 40 && rect.bottom > NAV_OFFSET + 40) current = s.id;
      }
      setActive(current);
    };
    const onScroll = () => {
      if (!frame) frame = requestAnimationFrame(measure);
    };
    measure();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll);
    return () => {
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onScroll);
      if (frame) cancelAnimationFrame(frame);
    };
  }, []);

  return { solid, active };
}

export function Nav() {
  const { data: user, isLoading } = useMe();
  const { solid, active } = useLandingScroll();
  const [open, setOpen] = useState(false);
  const sheetId = useId();
  const toggleRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        toggleRef.current?.focus();
      }
    };
    const onResize = () => {
      if (window.matchMedia("(min-width: 768px)").matches) setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("resize", onResize);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("resize", onResize);
    };
  }, [open]);

  const light = solid || open;

  return (
    <>
      <header className="fixed inset-x-0 top-3 md:top-4 z-50 flex justify-center px-3 md:px-6 pointer-events-none">
        <nav
          aria-label="Main"
          data-state={light ? "solid" : "clear"}
          className={cn(
            "landing-nav pointer-events-auto w-full max-w-5xl h-14 pl-4 pr-2 rounded-full flex items-center gap-2 border",
            light
              ? "text-foreground border-transparent"
              : "text-white border-[oklch(1_0_0/0.22)]",
          )}
        >
          <Link
            href="/"
            className="flex items-center gap-2 rounded-full pr-2 select-none outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <span
              className="grid place-items-center size-8 rounded-full transition-colors duration-300"
              style={{
                backgroundColor: light ? "var(--mint-soft)" : "oklch(1 0 0 / 0.16)",
              }}
            >
              <Compass
                className="size-[18px]"
                style={{ color: light ? "var(--mint-ink)" : "var(--mint)" }}
              />
            </span>
            <span className="font-display text-lg font-bold tracking-tight">TripCraft</span>
          </Link>

          <ul className="hidden md:flex items-center gap-1 mx-auto text-sm">
            {SECTIONS.map((s) => {
              const isActive = active === s.id;
              return (
                <li key={s.id}>
                  <a
                    href={`#${s.id}`}
                    aria-current={isActive ? "location" : undefined}
                    className={cn(
                      "inline-flex items-center h-9 px-4 rounded-full transition-colors duration-200 outline-none focus-visible:ring-2 focus-visible:ring-ring",
                      light
                        ? isActive
                          ? "bg-mint-soft text-mint-ink font-medium"
                          : "text-muted-foreground hover:text-foreground hover:bg-muted"
                        : isActive
                          ? "bg-white/20 text-white font-medium"
                          : "text-white/85 hover:text-white hover:bg-white/12",
                    )}
                  >
                    {s.label}
                  </a>
                </li>
              );
            })}
          </ul>

          <div className="ml-auto md:ml-0 flex items-center gap-1.5 min-h-9">
            {isLoading ? null : user ? (
              <>
                <Link
                  href="/trips"
                  className="hidden sm:inline-flex items-center gap-1.5 h-10 px-4 rounded-full text-sm font-medium bg-primary text-primary-foreground shadow-[0_6px_16px_-6px_var(--season-button-shadow)] transition-[filter,translate] duration-200 hover:brightness-[0.97] active:translate-y-px outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  Your trips <ArrowRight className="size-3.5" />
                </Link>
                <UserMenu />
              </>
            ) : (
              <>
                <Link
                  href="/sign-in"
                  className={cn(
                    "hidden md:inline-flex items-center h-10 px-4 rounded-full text-sm transition-colors duration-200 outline-none focus-visible:ring-2 focus-visible:ring-ring",
                    light ? "text-foreground hover:bg-muted" : "text-white/90 hover:text-white hover:bg-white/12",
                  )}
                >
                  Sign in
                </Link>
                <Link
                  href="/sign-up"
                  className="hidden sm:inline-flex items-center gap-1.5 h-10 px-4 rounded-full text-sm font-medium bg-primary text-primary-foreground shadow-[0_6px_16px_-6px_var(--season-button-shadow)] transition-[filter,translate] duration-200 hover:brightness-[0.97] active:translate-y-px outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  Get started <ArrowRight className="size-3.5" />
                </Link>
              </>
            )}

            <button
              ref={toggleRef}
              type="button"
              aria-label={open ? "Close menu" : "Open menu"}
              aria-expanded={open}
              aria-controls={sheetId}
              onClick={() => setOpen((v) => !v)}
              className={cn(
                "md:hidden grid place-items-center size-10 rounded-full transition-colors duration-200 outline-none focus-visible:ring-2 focus-visible:ring-ring",
                light ? "hover:bg-muted" : "hover:bg-white/12",
              )}
            >
              {open ? <X className="size-5" /> : <Menu className="size-5" />}
            </button>
          </div>
        </nav>
      </header>

      <div
        aria-hidden
        onClick={() => setOpen(false)}
        className={cn(
          "md:hidden fixed inset-0 z-40 transition-opacity duration-300",
          open ? "opacity-100" : "opacity-0 pointer-events-none",
        )}
        style={{ backgroundColor: "var(--overlay)" }}
      />

      <div
        id={sheetId}
        role="dialog"
        aria-modal="false"
        aria-label="Menu"
        inert={!open}
        className={cn(
          "landing-sheet md:hidden fixed inset-x-3 top-[76px] z-50 rounded-[1.75rem] bg-card p-3 shadow-soft-lg",
          open ? "is-open" : "",
        )}
      >
        <ul className="flex flex-col">
          {SECTIONS.map((s) => (
            <li key={s.id}>
              <a
                href={`#${s.id}`}
                onClick={() => setOpen(false)}
                className={cn(
                  "flex items-center justify-between h-12 px-4 rounded-2xl text-base transition-colors",
                  active === s.id ? "bg-mint-soft text-mint-ink font-medium" : "text-foreground hover:bg-muted",
                )}
              >
                {s.label}
                <ArrowRight className="size-4 opacity-50" />
              </a>
            </li>
          ))}
        </ul>
        {!isLoading && (
          <div className="mt-2 pt-3 border-t border-border flex flex-col gap-2">
            {user ? (
              <Link
                href="/trips"
                onClick={() => setOpen(false)}
                className="inline-flex items-center justify-center gap-1.5 h-12 rounded-full text-base font-medium bg-primary text-primary-foreground"
              >
                Your trips <ArrowRight className="size-4" />
              </Link>
            ) : (
              <>
                <Link
                  href="/sign-up"
                  onClick={() => setOpen(false)}
                  className="inline-flex items-center justify-center gap-1.5 h-12 rounded-full text-base font-medium bg-primary text-primary-foreground"
                >
                  Get started <ArrowRight className="size-4" />
                </Link>
                <Link
                  href="/sign-in"
                  onClick={() => setOpen(false)}
                  className="inline-flex items-center justify-center h-12 rounded-full text-base text-foreground hover:bg-muted"
                >
                  Sign in
                </Link>
              </>
            )}
          </div>
        )}
      </div>
    </>
  );
}
