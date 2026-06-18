"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, CheckCircle2, Loader2, MailCheck } from "lucide-react";

import { signUpSchema, type SignUpInput } from "@/lib/auth-schemas";
import {
  errorMessage,
  useResendVerification,
  useSignUp,
} from "@/hooks/auth-hooks";

export function SignUpForm() {
  const [pendingEmail, setPendingEmail] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SignUpInput>({
    resolver: zodResolver(signUpSchema),
    defaultValues: { name: "", email: "", password: "" },
  });

  const mutation = useSignUp({
    onSuccess: (session, vars) => {
      if (session?.requires_verification) {
        setPendingEmail(session.user?.email ?? vars.email);
        return;
      }
      window.location.assign("/");
    },
  });

  if (pendingEmail) return <CheckYourEmail email={pendingEmail} />;

  const submitError = errorMessage(mutation.error);

  return (
    <form
      onSubmit={handleSubmit((v) => mutation.mutate(v))}
      className="flex flex-col gap-5"
      noValidate
    >
      <Field
        id="name"
        label="Name"
        type="text"
        autoComplete="name"
        error={errors.name?.message}
        {...register("name")}
      />
      <Field
        id="email"
        label="Email"
        type="email"
        autoComplete="email"
        error={errors.email?.message}
        {...register("email")}
      />
      <Field
        id="password"
        label="Password"
        type="password"
        autoComplete="new-password"
        hint="At least 8 characters."
        error={errors.password?.message}
        {...register("password")}
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
  );
}

type FieldProps = React.InputHTMLAttributes<HTMLInputElement> & {
  id: string;
  label: string;
  hint?: string;
  error?: string;
};

function Field({ id, label, hint, error, ...input }: FieldProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <label
        htmlFor={id}
        className="text-[11px] tracking-[0.2em] uppercase font-medium"
        style={{ color: "#8B9A8E" }}
      >
        {label}
      </label>
      <input
        id={id}
        {...input}
        className="season-transition px-4 py-3 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]"
        style={{
          backgroundColor: "#161E19",
          border: `1px solid ${error ? "rgba(220,38,38,0.4)" : "#1F2A24"}`,
          color: "#ECEFEA",
        }}
        aria-invalid={error ? true : undefined}
      />
      {error ? (
        <p className="text-[11px]" style={{ color: "#FCA5A5" }}>
          {error}
        </p>
      ) : hint ? (
        <p className="text-[11px]" style={{ color: "#6B7A6F" }}>
          {hint}
        </p>
      ) : null}
    </div>
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
