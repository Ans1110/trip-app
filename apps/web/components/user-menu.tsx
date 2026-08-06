"use client";

import Link from "next/link";
import { HoverCard as RxHoverCard } from "radix-ui";
import { Bookmark, LogOut, User as UserIcon, Loader2 } from "lucide-react";
import { useState } from "react";

import { useMe } from "@/hooks/auth-hooks";
import { useMyProfile } from "@/hooks/profile-hooks";
import { cn } from "@/lib/utils";

type Size = "sm" | "md";

const SIZE_CLASSES: Record<Size, string> = {
  sm: "size-8 text-xs",
  md: "size-9 text-sm",
};

export function UserMenu({
  size = "md",
  className,
}: {
  size?: Size;
  className?: string;
}) {
  const { data: user, isLoading: userLoading } = useMe();
  const { data: profile } = useMyProfile({ enabled: !!user });
  const [signingOut, setSigningOut] = useState(false);

  if (userLoading) return <div className={cn(SIZE_CLASSES[size])} />;
  if (!user) return null;

  const displayName = profile?.name?.trim() || user.name?.trim() || user.email;
  const username = profile?.username || null;
  const avatarUrl = profile?.avatar_url || user.avatar_url || "";
  const initial = pickInitial(username, displayName, user.email);

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
    <RxHoverCard.Root openDelay={80} closeDelay={120}>
      <RxHoverCard.Trigger asChild>
        <Link
          href={username ? `/profile/${username}` : "/"}
          aria-label="Open user menu"
          className={cn(
            "season-transition rounded-full overflow-hidden inline-flex items-center justify-center font-semibold select-none outline-none focus:ring-2 focus:ring-[var(--season-accent)]/60",
            SIZE_CLASSES[size],
            className,
          )}
          style={{
            backgroundColor: avatarUrl
              ? "transparent"
              : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
            color: "#ECEFEA",
          }}
        >
          {avatarUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={avatarUrl}
              alt={displayName}
              className="size-full object-cover"
            />
          ) : (
            <span aria-hidden>{initial}</span>
          )}
        </Link>
      </RxHoverCard.Trigger>

      <RxHoverCard.Portal>
        <RxHoverCard.Content
          align="end"
          sideOffset={8}
          className={cn(
            "z-50 w-60 rounded-xl border shadow-lg outline-none p-1",
            "data-[state=open]:animate-in data-[state=closed]:animate-out",
            "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
            "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
          )}
          style={{
            backgroundColor: "#121814",
            borderColor: "#1F2A24",
            color: "#ECEFEA",
          }}
        >
          <div className="px-3 py-2.5 flex items-center gap-3">
            <div
              className="size-9 rounded-full overflow-hidden inline-flex items-center justify-center text-sm font-semibold shrink-0"
              style={{
                backgroundColor: avatarUrl
                  ? "transparent"
                  : "color-mix(in srgb, var(--season-accent) 24%, #1F2A24)",
              }}
            >
              {avatarUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={avatarUrl}
                  alt=""
                  className="size-full object-cover"
                />
              ) : (
                <span aria-hidden>{initial}</span>
              )}
            </div>
            <div className="min-w-0">
              <p
                className="text-sm font-semibold truncate"
                style={{ color: "#ECEFEA" }}
              >
                {displayName}
              </p>
              <p className="text-[11px] truncate" style={{ color: "#8B9A8E" }}>
                {username ? `@${username}` : user.email}
              </p>
            </div>
          </div>

          <div
            className="my-1 h-px"
            style={{ backgroundColor: "#1F2A24" }}
            aria-hidden
          />

          {username && (
            <MenuLink
              href={`/profile/${username}`}
              icon={<UserIcon className="size-4" />}
            >
              View profile
            </MenuLink>
          )}
          <MenuLink href="/bookmarks" icon={<Bookmark className="size-4" />}>
            Bookmarks
          </MenuLink>
          <button
            type="button"
            onClick={signOut}
            disabled={signingOut}
            className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-left hover:bg-white/5 transition-colors disabled:opacity-60"
            style={{ color: "#ECEFEA" }}
          >
            {signingOut ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <LogOut className="size-4" />
            )}
            <span>Sign out</span>
          </button>
        </RxHoverCard.Content>
      </RxHoverCard.Portal>
    </RxHoverCard.Root>
  );
}

function MenuLink({
  href,
  icon,
  children,
}: {
  href: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm hover:bg-white/5 transition-colors"
      style={{ color: "#ECEFEA" }}
    >
      {icon}
      <span>{children}</span>
    </Link>
  );
}

function pickInitial(
  username: string | null,
  name: string,
  email: string,
): string {
  const src = username || name || email;
  const trimmed = src.trim();
  if (!trimmed) return "U";
  return trimmed[0]!.toUpperCase();
}
