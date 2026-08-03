"use client";

import Link from "next/link";
import { ArrowRight, Compass } from "lucide-react";

import { useMe } from "@/hooks/auth-hooks";
import { UserMenu } from "@/components/user-menu";

export function Nav() {
  const { data: user, isLoading } = useMe();

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
        {isLoading ? null : user ? (
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
            <Link
              href="/feed"
              className="hidden sm:inline-flex items-center px-3 py-1.5 text-sm rounded-lg transition-colors hover:bg-white/5"
              style={{ color: "#ECEFEA" }}
            >
              Feed
            </Link>
            <UserMenu />
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
