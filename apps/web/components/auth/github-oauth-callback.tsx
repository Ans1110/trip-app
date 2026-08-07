"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ArrowRight, Check, Loader2, X } from "lucide-react";

import { errorMessage, useOAuthGithub } from "@/hooks/auth-hooks";
import { VerifyEmailPending } from "@/components/auth/verify-email-pending";

const GITHUB_STATE_KEY = "oauth:github:state";
const GITHUB_NEXT_KEY = "oauth:github:next";

const REDIRECT_DELAY_MS = 800;

const safeNext = (raw: string | null): string => {
  if (!raw) return "/trips";
  if (!raw.startsWith("/") || raw.startsWith("//")) return "/trips";
  return raw;
};

type Phase =
  | { kind: "checking" }
  | { kind: "exchanging" }
  | { kind: "success"; next: string }
  | { kind: "requires_verification"; email: string }
  | { kind: "error"; message: string };

type Initial =
  | { kind: "ready"; code: string }
  | { kind: "error"; message: string };

const evaluate = (
  providerError: string | null,
  code: string | null,
  returnedState: string | null,
): Initial => {
  if (providerError) return { kind: "error", message: providerError };
  if (!code || !returnedState) {
    return { kind: "error", message: "Missing code or state from GitHub." };
  }
  const savedState = sessionStorage.getItem(GITHUB_STATE_KEY);
  if (!savedState || savedState !== returnedState) {
    return {
      kind: "error",
      message: "Sign-in state mismatch. Please try again.",
    };
  }
  return { kind: "ready", code };
};

export function GithubOAuthCallback() {
  const searchParams = useSearchParams();
  const ran = useRef(false);

  const providerError =
    searchParams.get("error_description") ?? searchParams.get("error");
  const code = searchParams.get("code");
  const returnedState = searchParams.get("state");

  const [phase, setPhase] = useState<Phase>(() => {
    if (typeof window === "undefined") return { kind: "checking" };
    const initial = evaluate(providerError, code, returnedState);
    if (initial.kind === "error") return initial;
    return { kind: "exchanging" };
  });

  const github = useOAuthGithub({
    onSuccess: (session) => {
      const next = safeNext(sessionStorage.getItem(GITHUB_NEXT_KEY));
      sessionStorage.removeItem(GITHUB_STATE_KEY);
      sessionStorage.removeItem(GITHUB_NEXT_KEY);
      if (session?.requires_verification) {
        setPhase({
          kind: "requires_verification",
          email: session.user?.email ?? "",
        });
        return;
      }
      setPhase({ kind: "success", next });
    },
    onError: (err) => {
      setPhase({
        kind: "error",
        message: errorMessage(err) ?? "Sign-in failed.",
      });
    },
  });

  const mutate = github.mutate;

  useEffect(() => {
    if (ran.current) return;
    ran.current = true;
    const initial = evaluate(providerError, code, returnedState);
    if (initial.kind === "ready") mutate(initial.code);
  }, [code, mutate, providerError, returnedState]);

  useEffect(() => {
    if (phase.kind !== "success") return;
    const id = window.setTimeout(() => {
      window.location.assign(phase.next);
    }, REDIRECT_DELAY_MS);
    return () => window.clearTimeout(id);
  }, [phase]);

  const view = useMemo(() => {
    switch (phase.kind) {
      case "checking":
      case "exchanging":
        return (
          <Card tone="info" eyebrow="Signing you in">
            <p
              className="text-sm flex items-center gap-2"
              style={{ color: "#ECEFEA" }}
            >
              <Loader2 className="size-4 animate-spin" />
              Finishing sign-in with GitHub…
            </p>
          </Card>
        );
      case "success":
        return (
          <Card tone="success" eyebrow="Signed in">
            <p className="text-sm" style={{ color: "#ECEFEA" }}>
              Redirecting you now…
            </p>
          </Card>
        );
      case "requires_verification":
        return <VerifyEmailPending email={phase.email} />;
      case "error":
        return (
          <Card tone="error" eyebrow="Sign-in failed">
            <p className="text-sm" style={{ color: "#ECEFEA" }}>
              {phase.message}
            </p>
            <Link
              href="/sign-in"
              className="season-transition inline-flex items-center gap-2 mt-2 text-sm"
              style={{ color: "var(--season-accent)" }}
            >
              Back to sign in <ArrowRight className="size-3.5" />
            </Link>
          </Card>
        );
    }
  }, [phase]);

  return view;
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
