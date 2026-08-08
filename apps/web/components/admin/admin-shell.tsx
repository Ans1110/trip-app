"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { FileText, Flag, LogOut, Users } from "lucide-react";

import { cn } from "@/lib/utils";
import { useAdminLogout } from "@/hooks/admin-hooks";

const TABS = [
  { href: "/admin/users", label: "Users", Icon: Users },
  { href: "/admin/posts", label: "Posts", Icon: FileText },
  { href: "/admin/reports", label: "Reports", Icon: Flag },
];

export function AdminShell({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const logout = useAdminLogout({
    onSuccess: () => window.location.assign("/"),
  });

  return (
    <div className="px-6 md:px-12 py-10 w-full max-w-5xl mx-auto">
      <header className="flex items-center justify-between mb-8">
        <div>
          <p
            className="season-transition text-[11px] font-medium tracking-[0.25em] uppercase mb-2"
            style={{ color: "var(--season-button)" }}
          >
            Admin
          </p>
          <h1
            className="season-transition leading-tight tracking-tight"
            style={{
              fontFamily: "var(--font-display, Georgia, serif)",
              fontSize: "clamp(1.5rem, 2.5vw, 2rem)",
              color: "var(--season-heading)",
            }}
          >
            {title}
          </h1>
        </div>
        <button
          type="button"
          onClick={() => logout.mutate()}
          disabled={logout.isPending}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full border hover:bg-white/5 disabled:opacity-60"
          style={{ borderColor: "#1F2A24", color: "#ECEFEA" }}
        >
          <LogOut className="size-4" />
          End admin
        </button>
      </header>

      <nav
        className="mb-8 flex items-center gap-1 border-b"
        style={{ borderColor: "#1F2A24" }}
      >
        {TABS.map(({ href, label, Icon }) => {
          const active = pathname?.startsWith(href) ?? false;
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "inline-flex items-center gap-1.5 px-3 py-2 text-sm -mb-px border-b-2",
                active
                  ? "border-[var(--season-button)]"
                  : "border-transparent hover:bg-white/5",
              )}
              style={{
                color: active ? "var(--season-heading)" : "#8B9A8E",
              }}
            >
              <Icon className="size-4" />
              {label}
            </Link>
          );
        })}
      </nav>

      {children}
    </div>
  );
}
