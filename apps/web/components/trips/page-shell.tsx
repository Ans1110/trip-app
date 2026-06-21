import Link from "next/link";
import { ChevronLeft, Compass } from "lucide-react";
import type { ReactNode } from "react";

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
        {back && (
          <Link
            href={back.href}
            className="ml-2 inline-flex items-center gap-1 text-sm hover:bg-white/5 rounded-lg px-2 py-1"
            style={{ color: "#8B9A8E" }}
          >
            <ChevronLeft className="size-4" />
            {back.label}
          </Link>
        )}
      </header>
      {children}
    </main>
  );
}
