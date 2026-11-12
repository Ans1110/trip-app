import Link from "next/link";
import { ChevronLeft, Compass } from "lucide-react";
import type { ReactNode } from "react";

import { DesktopNav, MobileTabBar } from "@/components/app-nav";
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
      className="bg-wash season-transition relative min-h-screen flex flex-col"
      style={{
        color: "var(--foreground)",
      }}
    >
      <header className="px-4 sm:px-6 md:px-12 h-16 flex items-center gap-4">
        <Link href="/" className="flex items-center gap-2 select-none">
          <Compass
            className="season-transition size-5"
            style={{ color: "var(--mint-ink)" }}
          />
          <span
            className="text-xl tracking-tight"
            style={{ fontFamily: "var(--font-display), system-ui, sans-serif" }}
          >
            TripCraft
          </span>
        </Link>
        <DesktopNav />
        <div className="ml-auto flex items-center gap-3">
          {back && (
            <Link
              href={back.href}
              className="inline-flex items-center gap-1 text-sm hover:bg-muted rounded-lg px-2 py-1"
              style={{ color: "var(--muted-foreground)" }}
            >
              <ChevronLeft className="size-4" />
              {back.label}
            </Link>
          )}
          <UserMenu />
        </div>
      </header>
      {children}
      <MobileTabBar />
    </main>
  );
}
