"use client";

import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Check, Loader2, Mail, Shield, ShieldOff } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import {
  errorMessage,
  useDisableMFA,
  useEnableMFA,
  useMe,
  useRequestMFAEnableCode,
  useRequestMFADisableCode,
} from "@/hooks/auth-hooks";
import {
  mfaChallengeSchema,
  type MFAChallengeInput,
} from "@/lib/schemas/auth-schemas";

const LABEL = "text-[11px] tracking-[0.2em] uppercase";
const INPUT = "px-4 py-3";

type Phase = "idle" | "awaitingCode";
type Mode = "enable" | "disable";

export function MFASettings() {
  const { data: me, isLoading } = useMe();
  const [phase, setPhase] = useState<Phase>("idle");
  const [mode, setMode] = useState<Mode>("enable");

  const requestEnable = useRequestMFAEnableCode({
    onSuccess: () => {
      setMode("enable");
      setPhase("awaitingCode");
    },
  });
  const requestDisable = useRequestMFADisableCode({
    onSuccess: () => {
      setMode("disable");
      setPhase("awaitingCode");
    },
  });
  const enable = useEnableMFA({ onSuccess: () => setPhase("idle") });
  const disable = useDisableMFA({ onSuccess: () => setPhase("idle") });

  if (isLoading) {
    return (
      <div
        className="flex items-center gap-2 text-sm"
        style={{ color: "#8B9A8E" }}
      >
        <Loader2 className="size-4 animate-spin" />
        Loading security settings
      </div>
    );
  }

  const mfaEnabled = !!me?.mfa_enabled;
  const requestPending = requestEnable.isPending || requestDisable.isPending;
  const verifyPending = enable.isPending || disable.isPending;
  const requestError = errorMessage(
    mode === "enable" ? requestEnable.error : requestDisable.error,
  );

  if (phase === "awaitingCode") {
    return (
      <MFACodeForm
        email={me?.email ?? ""}
        mode={mode}
        pending={verifyPending}
        submitError={errorMessage(mode === "enable" ? enable.error : disable.error)}
        onCancel={() => {
          setPhase("idle");
          enable.reset();
          disable.reset();
        }}
        onResend={() =>
          mode === "enable"
            ? requestEnable.mutate()
            : requestDisable.mutate()
        }
        onSubmit={(code) =>
          mode === "enable" ? enable.mutate(code) : disable.mutate(code)
        }
      />
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-start gap-3">
        {mfaEnabled ? (
          <Shield className="size-5 mt-0.5" style={{ color: "var(--season-button)" }} />
        ) : (
          <ShieldOff className="size-5 mt-0.5" style={{ color: "#8B9A8E" }} />
        )}
        <div className="flex-1">
          <h3 className="text-sm font-medium" style={{ color: "#ECEFEA" }}>
            Email two-factor
          </h3>
          <p className="text-xs mt-0.5" style={{ color: "#8B9A8E" }}>
            {mfaEnabled
              ? "You'll receive a 6-digit code by email on sign-in."
              : "Add a second step at sign-in by receiving a code by email."}
          </p>
        </div>
      </div>

      {requestError && (
        <p
          className="text-sm rounded-lg px-3 py-2"
          style={{
            color: "#FCA5A5",
            backgroundColor: "rgba(220,38,38,0.08)",
            border: "1px solid rgba(220,38,38,0.25)",
          }}
          role="alert"
        >
          {requestError}
        </p>
      )}

      <div>
        <button
          type="button"
          disabled={requestPending}
          onClick={() =>
            mfaEnabled ? requestDisable.mutate() : requestEnable.mutate()
          }
          className="season-transition inline-flex items-center gap-2 px-4 py-2 rounded-full text-xs font-medium disabled:opacity-60"
          style={
            mfaEnabled
              ? {
                  color: "#FCA5A5",
                  border: "1px solid rgba(220,38,38,0.35)",
                  backgroundColor: "transparent",
                }
              : {
                  backgroundColor: "var(--season-button)",
                  color: "#0B100D",
                  boxShadow: "0 6px 18px var(--season-button-shadow)",
                }
          }
        >
          {requestPending ? (
            <>
              <Loader2 className="size-3.5 animate-spin" />
              Sending code
            </>
          ) : mfaEnabled ? (
            "Disable two-factor"
          ) : (
            "Enable two-factor"
          )}
        </button>
      </div>
    </div>
  );
}

function MFACodeForm({
  email,
  mode,
  onSubmit,
  onResend,
  onCancel,
  submitError,
  pending,
}: {
  email: string;
  mode: Mode;
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
        className="flex flex-col gap-4"
        noValidate
      >
        <div
          className="flex items-start gap-3 rounded-lg px-3 py-2"
          style={{
            backgroundColor: "rgba(127,182,138,0.08)",
            border: "1px solid rgba(127,182,138,0.25)",
          }}
        >
          <Mail
            className="size-5 mt-0.5"
            style={{ color: "var(--season-button)" }}
          />
          <div>
            <h3 className="text-sm font-medium" style={{ color: "#ECEFEA" }}>
              Code sent to your email
            </h3>
            <p className="text-xs" style={{ color: "#8B9A8E" }}>
              We sent a 6-digit code to{" "}
              <span style={{ color: "#ECEFEA" }}>{email}</span> to{" "}
              {mode === "enable" ? "enable" : "disable"} two-factor. It expires in a few minutes.
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
          labelClassName={LABEL}
          className={`${INPUT} tracking-[0.4em] text-center`}
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

        <div className="flex items-center gap-2">
          <button
            type="submit"
            disabled={pending}
            className="season-transition inline-flex items-center gap-2 px-4 py-2 rounded-full text-xs font-medium disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
              boxShadow: "0 6px 18px var(--season-button-shadow)",
            }}
          >
            {pending ? (
              <>
                <Loader2 className="size-3.5 animate-spin" />
                Verifying
              </>
            ) : (
              <>
                <Check className="size-3.5" />
                Confirm
              </>
            )}
          </button>
          {canResend ? (
            <button
              type="button"
              onClick={handleResend}
              className="text-xs hover:underline"
              style={{ color: "var(--season-button)" }}
            >
              Resend code
            </button>
          ) : (
            <span
              className="inline-flex items-center gap-1 text-xs"
              style={{ color: "#7FB68A" }}
            >
              <Check className="size-3" />
              Sent · resend in {cooldownLeft}s
            </span>
          )}
          <button
            type="button"
            onClick={onCancel}
            className="ml-auto text-xs hover:underline"
            style={{ color: "#8B9A8E" }}
          >
            Cancel
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
