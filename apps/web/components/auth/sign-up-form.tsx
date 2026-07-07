"use client";

import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, CheckCircle2, Loader2, MailCheck } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { signUpSchema, type SignUpInput } from "@/lib/schemas/auth-schemas";
import {
  errorMessage,
  useResendVerification,
  useSignUp,
} from "@/hooks/auth-hooks";

const AUTH_LABEL = "text-[11px] tracking-[0.2em] uppercase";
const AUTH_INPUT = "px-4 py-3";

export function SignUpForm() {
  const [pendingEmail, setPendingEmail] = useState<string | null>(null);
  const form = useForm<SignUpInput>({
    resolver: zodResolver(signUpSchema),
    defaultValues: { name: "", email: "", password: "" },
  });

  const mutation = useSignUp({
    onSuccess: (session, vars) => {
      if (session?.requires_verification) {
        setPendingEmail(session.user?.email ?? vars.email);
        return;
      }
      window.location.assign("/trips");
    },
  });

  if (pendingEmail) return <CheckYourEmail email={pendingEmail} />;

  const submitError = errorMessage(mutation.error);

  return (
    <FormProvider {...form}>
      <form
        onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
        className="flex flex-col gap-5"
        noValidate
      >
        <ControlledInput<SignUpInput>
          name="name"
          label="Name"
          type="text"
          autoComplete="name"
          labelClassName={AUTH_LABEL}
          className={AUTH_INPUT}
        />
        <ControlledInput<SignUpInput>
          name="email"
          label="Email"
          type="email"
          autoComplete="email"
          labelClassName={AUTH_LABEL}
          className={AUTH_INPUT}
        />
        <ControlledInput<SignUpInput>
          name="password"
          label="Password"
          type="password"
          autoComplete="new-password"
          hint="At least 8 characters."
          labelClassName={AUTH_LABEL}
          className={AUTH_INPUT}
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
              Creating account
            </>
          ) : (
            <>
              Create account <ArrowRight className="size-4" />
            </>
          )}
        </button>
      </form>
    </FormProvider>
  );
}

function CheckYourEmail({ email }: { email: string }) {
  const resend = useResendVerification();
  const errorMsg = resend.isError ? errorMessage(resend.error) : null;

  return (
    <div
      className="season-transition flex flex-col gap-5 rounded-2xl px-5 py-6"
      style={{
        backgroundColor:
          "color-mix(in srgb, var(--season-accent) 8%, transparent)",
        border:
          "1px solid color-mix(in srgb, var(--season-accent) 28%, transparent)",
      }}
      role="status"
    >
      <div className="flex items-center gap-2">
        <MailCheck
          className="season-transition size-4"
          style={{ color: "var(--season-accent)" }}
        />
        <p
          className="season-transition text-[11px] tracking-[0.2em] uppercase font-medium"
          style={{ color: "var(--season-accent)" }}
        >
          Check your email
        </p>
      </div>

      <p className="text-sm leading-relaxed" style={{ color: "#ECEFEA" }}>
        We sent a verification link to{" "}
        <span
          className="season-transition"
          style={{ color: "var(--season-accent)" }}
        >
          {email}
        </span>
        . Click it to finish signing up — you&apos;ll be signed in
        automatically.
      </p>

      <p className="text-xs" style={{ color: "#8B9A8E" }}>
        Didn&apos;t get it? Check spam, or resend below.
      </p>

      <button
        type="button"
        onClick={() => resend.mutate(email)}
        disabled={resend.isPending || resend.isSuccess}
        className="season-transition inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide disabled:opacity-60"
        style={{
          backgroundColor: "transparent",
          color: "var(--season-accent)",
          border:
            "1px solid color-mix(in srgb, var(--season-accent) 38%, transparent)",
        }}
      >
        {resend.isPending && (
          <>
            <Loader2 className="size-4 animate-spin" />
            Resending
          </>
        )}
        {resend.isSuccess && (
          <>
            <CheckCircle2 className="size-4" />
            Email sent
          </>
        )}
        {(resend.isIdle || resend.isError) && <>Resend verification email</>}
      </button>

      {errorMsg && (
        <p
          className="text-xs rounded-lg px-3 py-2"
          style={{
            color: "#FCA5A5",
            backgroundColor: "rgba(220,38,38,0.08)",
            border: "1px solid rgba(220,38,38,0.25)",
          }}
          role="alert"
        >
          {errorMsg}
        </p>
      )}
    </div>
  );
}
