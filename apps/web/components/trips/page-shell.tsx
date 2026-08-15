import Link from "next/link";
import { ChevronLeft, Compass } from "lucide-react";
import type { ReactNode } from "react";

import { UserMenu } from "@/components/user-menu";

export function PageShell({
  back,
  children,
}: {
  back?: { href: string; label: string };
  children: ReactNode;
}) {
  return (
    <main
      className="season-transition relative min-h-screen flex flex-col"
      style={{
        backgroundColor: "#0B100D",
        color: "#ECEFEA",
        backgroundImage:
          "radial-gradient(circle 600px at 80% -10%, color-mix(in srgb, var(--season-button) 18%, transparent), transparent 70%), radial-gradient(circle 500px at 10% 110%, color-mix(in srgb, var(--season-accent) 12%, transparent), transparent 70%)",
      }}
    >
      <header className="px-6 md:px-12 h-16 flex items-center gap-4">
        <Link href="/" className="flex items-center gap-2 select-none">
          <Compass
            className="season-transition size-5"
            style={{ color: "var(--season-button)" }}
          />
          <span
            className="text-xl tracking-tight"
            style={{ fontFamily: "var(--font-display, Georgia, serif)" }}
          >
            TripCraft
          </span>
        </Link>
        <nav className="ml-4 hidden md:flex items-center gap-1 text-sm">
          <Link
            href="/trips"
            className="px-3 py-1.5 rounded-lg hover:bg-white/5"
            style={{ color: "#ECEFEA" }}
          >
            Trips
          </Link>
          <Link
            href="/calendar"
            className="px-3 py-1.5 rounded-lg hover:bg-white/5"
            style={{ color: "#ECEFEA" }}
          >
            Calendar
          </Link>
          <Link
            href="/location"
            className="px-3 py-1.5 rounded-lg hover:bg-white/5"
            style={{ color: "#ECEFEA" }}
          >
            Places
          </Link>
          <Link
            href="/weather"
            className="px-3 py-1.5 rounded-lg hover:bg-white/5"
            style={{ color: "#ECEFEA" }}
          >
            Weather
          </Link>
          <Link
            href="/feed"
            className="px-3 py-1.5 rounded-lg hover:bg-white/5"
            style={{ color: "#ECEFEA" }}
          >
            Feed
          </Link>
        </nav>
        <div className="ml-auto flex items-center gap-3">
          {back && (
            <Link
              href={back.href}
              className="inline-flex items-center gap-1 text-sm hover:bg-white/5 rounded-lg px-2 py-1"
              style={{ color: "#8B9A8E" }}
            >
              <ChevronLeft className="size-4" />
              {back.label}
            </Link>
          )}
          <UserMenu />
        </div>
      </header>
      {children}
    </main>
  );
}
