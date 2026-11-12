"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { CalendarDays, CloudSun, Luggage, MapPin, Newspaper } from "lucide-react";

import { useMe } from "@/hooks/auth-hooks";
import { cn } from "@/lib/utils";

const NAV = [
  { href: "/trips", label: "Trips", icon: Luggage },
  { href: "/calendar", label: "Calendar", icon: CalendarDays },
  { href: "/location", label: "Places", icon: MapPin },
  { href: "/weather", label: "Weather", icon: CloudSun },
  { href: "/feed", label: "Feed", icon: Newspaper },
] as const;

function useActiveHref() {
  const pathname = usePathname() ?? "";
  return NAV.find((n) => pathname === n.href || pathname.startsWith(`${n.href}/`))?.href;
}

export function DesktopNav() {
  const active = useActiveHref();
  return (
    <nav aria-label="Main" className="ml-4 hidden md:flex items-center gap-1 text-sm">
      {NAV.map((n) => {
        const on = n.href === active;
        return (
          <Link
            key={n.href}
            href={n.href}
            aria-current={on ? "page" : undefined}
            className={cn(
              "px-3 py-1.5 rounded-full transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
              on ? "bg-mint-soft text-mint-ink font-medium" : "text-foreground hover:bg-muted",
            )}
          >
            {n.label}
          </Link>
        );
      })}
    </nav>
  );
}

export function MobileTabBar() {
  const active = useActiveHref();
  const { data: user } = useMe();
  if (!user) return null;

  return (
    <nav aria-label="Main" className="mobile-tabbar md:hidden">
      <ul className="grid grid-cols-5 h-full">
        {NAV.map(({ href, label, icon: Icon }) => {
          const on = href === active;
          return (
            <li key={href} className="flex">
              <Link
                href={href}
                aria-current={on ? "page" : undefined}
                className={cn(
                  "flex-1 flex flex-col items-center justify-center gap-0.5 rounded-full text-[11px] outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset transition-colors",
                  on ? "text-mint-ink font-semibold" : "text-muted-foreground active:bg-muted",
                )}
              >
                <span
                  className={cn(
                    "grid place-items-center h-7 w-12 rounded-full transition-colors",
                    on && "bg-mint-soft",
                  )}
                >
                  <Icon className="size-[19px]" strokeWidth={on ? 2.2 : 1.8} aria-hidden />
                </span>
                {label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
