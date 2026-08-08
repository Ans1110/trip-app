"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, Check, Loader2, Mail } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import {
  mfaChallengeSchema,
  signInSchema,
  type MFAChallengeInput,
  type SignInInput,
} from "@/lib/schemas/auth-schemas";
import { errorMessage, useSignIn } from "@/hooks/auth-hooks";
import { OAuthButtons } from "@/components/auth/oauth-buttons";
import { VerifyEmailPending } from "@/components/auth/verify-email-pending";
import type { SessionView } from "@/lib/apis/auth-api";

const AUTH_LABEL = "text-[11px] tracking-[0.2em] uppercase";
const AUTH_INPUT = "px-4 py-3";

const safeNext = (raw: string | null): string => {
  if (!raw) return "/trips";
  if (!raw.startsWith("/") || raw.startsWith("//")) return "/trips";
  return raw;
};

export function SignInForm() {
  const searchParams = useSearchParams();
  const next = safeNext(searchParams.get("next"));
  const [pendingEmail, setPendingEmail] = useState<string | null>(null);
  const [challenge, setChallenge] = useState<SignInInput | null>(null);

  const form = useForm<SignInInput>({
    resolver: zodResolver(signInSchema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useSignIn({
    onSuccess: (session, variables) => {
      if (session?.requires_mfa) {
        setChallenge(variables);
        return;
      }
      if (session?.requires_verification) {
        setPendingEmail(session.user?.email ?? variables.email);
        return;
      }
      window.location.assign(next);
    },
  });

  const handleOAuthSession = (session: SessionView) => {
    if (session?.requires_verification) {
      setPendingEmail(session.user?.email ?? null);
      return;
    }
    window.location.assign(next);
  };

  if (pendingEmail) return <VerifyEmailPending email={pendingEmail} />;

  if (challenge) {
    return (
      <MFAChallengeForm
        email={challenge.email}
        onCancel={() => {
          setChallenge(null);
          mutation.reset();
        }}
        onSubmit={(code) => mutation.mutate({ ...challenge, mfa_code: code })}
        onResend={() => mutation.mutate({ ...challenge, mfa_code: undefined })}
        submitError={errorMessage(mutation.error)}
        pending={mutation.isPending}
      />
    );
  }

  const submitError = errorMessage(mutation.error);

  return (
    <FormProvider {...form}>
      <form
        onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
        className="flex flex-col gap-5"
        noValidate
      >
        <ControlledInput<SignInInput>
          name="email"
          label="Email"
          type="email"
          autoComplete="email"
          labelClassName={AUTH_LABEL}
          className={AUTH_INPUT}
        />
        <ControlledInput<SignInInput>
          name="password"
          label="Password"
          type="password"
          autoComplete="current-password"
          labelClassName={AUTH_LABEL}
          className={AUTH_INPUT}
          labelTrailing={
            <Link
              href="/forgot-password"
              className="season-transition text-xs tracking-wide hover:underline"
              style={{ color: "var(--season-button)" }}
            >
              Forgot?
            </Link>
          }
        />

        {submitError && (
          <p
            className="text-sm rounded-lg px-3 py-2"
            style={{
              color: "#FCA5A5",
              backgroundColor: "rgba(220,38,38,0.08)",
              border: "1px solid rgba(220,38,38,0.25)",
            }}
            role="alert"
          >
            {submitError}
          </p>
        )}

        <button
          type="submit"
          disabled={mutation.isPending}
          className="season-transition inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-60 disabled:translate-y-0"
          style={{
            backgroundColor: "var(--season-button)",
            color: "#0B100D",
            boxShadow: "0 8px 24px var(--season-button-shadow)",
          }}
        >
          {mutation.isPending ? (
            <>
              <Loader2 className="size-4 animate-spin" />
              Signing in
            </>
          ) : (
            <>
              Sign in <ArrowRight className="size-4" />
            </>
          )}
        </button>

        <OAuthButtons next={next} onSession={handleOAuthSession} />
      </form>
    </FormProvider>
  );
}

function MFAChallengeForm({
  email,
  onSubmit,
  onResend,
  onCancel,
  submitError,
  pending,
}: {
  email: string;
  onSubmit: (code: string) => void;
  onResend: () => void;
  onCancel: () => void;
  submitError: string | null;
  pending: boolean;
}) {
  const form = useForm<MFAChallengeInput>({
    resolver: zodResolver(mfaChallengeSchema),
    defaultValues: { mfa_code: "" },
  });
  const { cooldownLeft, canResend, markSent } = useResendCooldown(30);

  const handleResend = () => {
    if (!canResend) return;
    markSent();
    onResend();
  };

  return (
    <FormProvider {...form}>
      <form
        onSubmit={form.handleSubmit((v) => onSubmit(v.mfa_code))}
        className="flex flex-col gap-5"
        noValidate
      >
        <div
          className="flex items-start gap-3 rounded-lg px-3 py-2"
          style={{
            backgroundColor: "rgba(127,182,138,0.08)",
            border: "1px solid rgba(127,182,138,0.25)",
          }}
        >
          <Mail className="size-5 mt-0.5" style={{ color: "var(--season-button)" }} />
          <div>
            <h2 className="text-sm font-medium" style={{ color: "#ECEFEA" }}>
              Code sent to your email
            </h2>
            <p className="text-xs" style={{ color: "#8B9A8E" }}>
              We sent a 6-digit code to <span style={{ color: "#ECEFEA" }}>{email}</span>. It expires in a few minutes.
            </p>
          </div>
        </div>

        <ControlledInput<MFAChallengeInput>
          name="mfa_code"
          label="Verification code"
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          labelClassName={AUTH_LABEL}
          className={`${AUTH_INPUT} tracking-[0.4em] text-center`}
          autoFocus
        />

        {submitError && (
          <p
            className="text-sm rounded-lg px-3 py-2"
            style={{
              color: "#FCA5A5",
              backgroundColor: "rgba(220,38,38,0.08)",
              border: "1px solid rgba(220,38,38,0.25)",
            }}
            role="alert"
          >
            {submitError}
          </p>
        )}

        <button
          type="submit"
          disabled={pending}
          className="season-transition inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-60 disabled:translate-y-0"
          style={{
            backgroundColor: "var(--season-button)",
            color: "#0B100D",
            boxShadow: "0 8px 24px var(--season-button-shadow)",
          }}
        >
          {pending ? (
            <>
              <Loader2 className="size-4 animate-spin" />
              Verifying
            </>
          ) : (
            <>
              Verify <ArrowRight className="size-4" />
            </>
          )}
        </button>

        <div className="flex items-center justify-between text-xs" style={{ color: "#8B9A8E" }}>
          {canResend ? (
            <button
              type="button"
              onClick={handleResend}
              className="hover:underline"
              style={{ color: "var(--season-button)" }}
            >
              Resend code
            </button>
          ) : (
            <span
              className="inline-flex items-center gap-1"
              style={{ color: "#7FB68A" }}
            >
              <Check className="size-3" />
              Sent · resend in {cooldownLeft}s
            </span>
          )}
          <button
            type="button"
            onClick={onCancel}
            className="hover:underline"
          >
            Use a different account
          </button>
        </div>
      </form>
    </FormProvider>
  );
}

function useResendCooldown(cooldownSec: number) {
  const [sentAt, setSentAt] = useState<number>(() => Date.now());
  const [now, setNow] = useState<number>(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);
  const elapsedSec = Math.floor((now - sentAt) / 1000);
  const cooldownLeft = Math.max(0, cooldownSec - elapsedSec);
  return {
    cooldownLeft,
    canResend: cooldownLeft === 0,
    markSent: () => setSentAt(Date.now()),
  };
}
