"use client";

import { useState } from "react";
import { Check, Loader2, MailCheck, X } from "lucide-react";

import { errorMessage, useMe, useResendVerification } from "@/hooks/auth-hooks";

export function VerifyEmailBanner() {
  const [dismissed, setDismissed] = useState(false);
  const meQuery = useMe();
  const resend = useResendVerification();

  if (!meQuery.isFetched || dismissed) return null;
  const me = meQuery.data;
  if (!me || me.is_verified) return null;

  const errorMsg = resend.isError ? errorMessage(resend.error) : null;

  return (
    <div
      className="fixed top-16 inset-x-0 z-40 px-4 md:px-8 pt-3"
      style={{ pointerEvents: "none" }}
    >
      <div
        className="season-transition mx-auto max-w-3xl flex items-center gap-3 px-4 py-3 rounded-2xl backdrop-blur-md"
        style={{
          pointerEvents: "auto",
          backgroundColor:
            "color-mix(in srgb, var(--season-button) 14%, transparent)",
          border:
            "1px solid color-mix(in srgb, var(--season-button) 30%, transparent)",
          color: "#ECEFEA",
        }}
        role="status"
      >
        <MailCheck
          className="season-transition size-4 shrink-0"
          style={{ color: "var(--season-accent)" }}
        />
        <p className="text-sm flex-1 min-w-0">
          {resend.isSuccess ? (
            <>
              Verification email re-sent to{" "}
              <strong className="truncate">{me.email}</strong>. Check your
              inbox.
            </>
          ) : errorMsg ? (
            <>
              Couldn&apos;t resend:{" "}
              <span style={{ color: "#FCA5A5" }}>{errorMsg}</span>
            </>
          ) : (
            <>
              Verify your email to unlock all features. We sent a link to{" "}
              <strong className="truncate">{me.email}</strong>.
            </>
          )}
        </p>
        <button
          type="button"
          onClick={() => resend.mutate(me.email)}
          disabled={resend.isPending || resend.isSuccess}
          className="season-transition hidden sm:inline-flex items-center gap-1.5 text-xs tracking-wide px-3 py-1.5 rounded-full disabled:opacity-60"
          style={{
            backgroundColor:
              "color-mix(in srgb, var(--season-button) 22%, transparent)",
            color: "#ECEFEA",
            border:
              "1px solid color-mix(in srgb, var(--season-button) 32%, transparent)",
          }}
        >
          {resend.isPending ? (
            <>
              <Loader2 className="size-3.5 animate-spin" /> Sending
            </>
          ) : resend.isSuccess ? (
            <>
              <Check className="size-3.5" /> Sent
            </>
          ) : (
            "Resend"
          )}
        </button>
        <button
          type="button"
          onClick={() => setDismissed(true)}
          aria-label="Dismiss"
          className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/10 transition-colors"
          style={{ color: "#8B9A8E" }}
        >
          <X className="size-3.5" />
        </button>
      </div>
    </div>
  );
}
