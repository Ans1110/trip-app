"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowRight, Compass, Loader2 } from "lucide-react";

type Me = { id: string; email: string; name: string; avatar_url?: string };
type MeResponse = { code: number; message: string; data: Me | null };

export function Nav() {
  const [user, setUser] = useState<Me | null>(null);
  const [status, setStatus] = useState<"loading" | "ready">("loading");
  const [signingOut, setSigningOut] = useState(false);

  useEffect(() => {
    const ac = new AbortController();
    (async () => {
      try {
        const res = await fetch("/api/auth/me", {
          credentials: "include",
          signal: ac.signal,
        });
        if (res.ok) {
          const body = (await res.json()) as MeResponse;
          setUser(body.data);
        }
      } catch {
        // ignore — treated as signed-out
      } finally {
        setStatus("ready");
      }
    })();
    return () => ac.abort();
  }, []);

  async function signOut() {
    setSigningOut(true);
    try {
      await fetch("/api/auth/logout", {
        method: "POST",
        credentials: "include",
      });
    } finally {
      window.location.assign("/");
    }
  }

  return (
    <nav
      className="fixed top-0 inset-x-0 z-50 flex items-center justify-between px-6 md:px-12 h-16 backdrop-blur-md"
      style={{ backgroundColor: "rgba(11,16,13,0.55)" }}
    >
      <Link href="/" className="flex items-center gap-2 select-none">
        <Compass
          className="season-transition size-5"
          style={{ color: "var(--season-button)" }}
        />
        <span
          className="text-xl tracking-tight"
          style={{
            fontFamily: "var(--font-display, Georgia, serif)",
            color: "#ECEFEA",
          }}
        >
          TripCraft
        </span>
      </Link>

      <div
        className="hidden md:flex items-center gap-8 text-sm tracking-wide"
        style={{ color: "#8B9A8E" }}
      >
        <Link href="#features" className="hover:text-[#ECEFEA] transition-colors">
          Routes
        </Link>
        <Link href="#preview" className="hover:text-[#ECEFEA] transition-colors">
          Itinerary
        </Link>
        <Link href="#about" className="hover:text-[#ECEFEA] transition-colors">
          About
        </Link>
      </div>

      <div className="flex items-center gap-3 min-h-[36px]">
        {status === "loading" ? null : user ? (
          <>
            <Link
              href="/trips"
              className="hidden sm:inline-flex items-center px-3 py-1.5 text-sm rounded-lg transition-colors hover:bg-white/5"
              style={{ color: "#ECEFEA" }}
            >
              Trips
            </Link>
            <Link
              href="/calendar"
              className="hidden sm:inline-flex items-center px-3 py-1.5 text-sm rounded-lg transition-colors hover:bg-white/5"
              style={{ color: "#ECEFEA" }}
            >
              Calendar
            </Link>
            <span
              className="hidden sm:inline text-sm"
              style={{ color: "#ECEFEA" }}
            >
              {user.name || user.email}
            </span>
            <button
              type="button"
              onClick={signOut}
              disabled={signingOut}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg transition-colors hover:bg-white/5 disabled:opacity-60"
              style={{ color: "#ECEFEA" }}
            >
              {signingOut && <Loader2 className="size-3.5 animate-spin" />}
              Sign out
            </button>
          </>
        ) : (
          <>
            <Link
              href="/sign-in"
              className="hidden sm:inline-flex items-center px-3 py-1.5 text-sm rounded-lg transition-colors hover:bg-white/5"
              style={{ color: "#ECEFEA" }}
            >
              Sign in
            </Link>
            <Link
              href="/sign-up"
              className="season-transition inline-flex items-center gap-1.5 px-4 py-2 rounded-full text-sm font-medium hover:opacity-90 active:scale-95"
              style={{
                backgroundColor: "var(--season-button)",
                color: "#0B100D",
              }}
            >
              Get started <ArrowRight className="size-3.5" />
            </Link>
          </>
        )}
      </div>
    </nav>
  );
}
