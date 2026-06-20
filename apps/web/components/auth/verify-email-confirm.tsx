"use client";

import { useEffect, useRef } from "react";
import Link from "next/link";
import { ArrowRight, Check, Loader2, X } from "lucide-react";

import { errorMessage, useVerifyEmail } from "@/hooks/auth-hooks";

const REDIRECT_DELAY_MS = 1500;

export function VerifyEmailConfirm({ token }: { token: string | null }) {
  const ran = useRef(false);
  const verify = useVerifyEmail();
  const { mutate } = verify;

  useEffect(() => {
    if (!token || ran.current) return;
    ran.current = true;
    mutate(token);
  }, [token, mutate]);

  useEffect(() => {
    if (!verify.isSuccess) return;
    const id = window.setTimeout(() => {
      window.location.assign("/trips");
    }, REDIRECT_DELAY_MS);
    return () => window.clearTimeout(id);
  }, [verify.isSuccess]);

  if (!token) {
    return (
      <Card tone="error" eyebrow="Link incomplete">
        <p className="text-sm" style={{ color: "#ECEFEA" }}>
          This page needs a verification token. Open the link from your email,
          or request a new one from your account.
        </p>
        <Link
          href="/"
          className="season-transition inline-flex items-center gap-2 mt-2 text-sm"
          style={{ color: "var(--season-accent)" }}
        >
          Back to home <ArrowRight className="size-3.5" />
        </Link>
      </Card>
    );
  }

  if (verify.isIdle || verify.isPending) {
    return (
      <Card tone="info" eyebrow="Working">
        <p
          className="text-sm flex items-center gap-2"
          style={{ color: "#ECEFEA" }}
        >
          <Loader2 className="size-4 animate-spin" />
          Confirming your token…
        </p>
      </Card>
    );
  }

  if (verify.isSuccess) {
    return (
      <Card tone="success" eyebrow="Email verified">
        <p className="text-sm" style={{ color: "#ECEFEA" }}>
          Your email is confirmed. Taking you to your trips…
        </p>
        <p className="text-xs" style={{ color: "#8B9A8E" }}>
          Not redirecting?{" "}
          <Link
            href="/trips"
            className="season-transition underline-offset-4 hover:underline"
            style={{ color: "var(--season-accent)" }}
          >
            Go to trips
          </Link>
        </p>
      </Card>
    );
  }

  const message =
    errorMessage(verify.error) ??
    "Your link may have expired or already been used.";

  return (
    <Card tone="error" eyebrow="Verification failed">
      <p className="text-sm" style={{ color: "#ECEFEA" }}>
        {message}
      </p>
      <Link
        href="/"
        className="season-transition inline-flex items-center gap-2 mt-2 text-sm"
        style={{ color: "var(--season-accent)" }}
      >
        Back to home <ArrowRight className="size-3.5" />
      </Link>
    </Card>
  );
}

type Tone = "info" | "success" | "error";

function Card({
  tone,
  eyebrow,
  children,
}: {
  tone: Tone;
  eyebrow: string;
  children: React.ReactNode;
}) {
  const palette =
    tone === "error"
      ? {
          bg: "rgba(220,38,38,0.06)",
          border: "rgba(220,38,38,0.25)",
          accent: "#FCA5A5",
        }
      : tone === "success"
        ? {
            bg: "color-mix(in srgb, var(--season-accent) 8%, transparent)",
            border:
              "color-mix(in srgb, var(--season-accent) 28%, transparent)",
            accent: "var(--season-accent)",
          }
        : {
            bg: "color-mix(in srgb, var(--season-accent) 6%, transparent)",
            border:
              "color-mix(in srgb, var(--season-accent) 20%, transparent)",
            accent: "var(--season-accent)",
          };
  const Icon = tone === "success" ? Check : tone === "error" ? X : Loader2;

  return (
    <div
      className="season-transition flex flex-col gap-3 rounded-2xl px-5 py-6"
      style={{
        backgroundColor: palette.bg,
        border: `1px solid ${palette.border}`,
      }}
      role="status"
    >
      <div className="flex items-center gap-2">
        <Icon
          className={`season-transition size-4 ${tone === "info" ? "animate-spin" : ""}`}
          style={{ color: palette.accent }}
        />
        <p
          className="season-transition text-[11px] tracking-[0.2em] uppercase font-medium"
          style={{ color: palette.accent }}
        >
          {eyebrow}
        </p>
      </div>
      {children}
    </div>
  );
}
