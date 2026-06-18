"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, Check, Loader2 } from "lucide-react";

import {
  forgotPasswordSchema,
  type ForgotPasswordInput,
} from "@/lib/auth-schemas";
import { errorMessage, useForgotPassword } from "@/hooks/auth-hooks";

export function ForgotPasswordForm() {
  const {
    register,
    handleSubmit,
    getValues,
    reset,
    formState: { errors },
  } = useForm<ForgotPasswordInput>({
    resolver: zodResolver(forgotPasswordSchema),
    defaultValues: { email: "" },
  });

  const mutation = useForgotPassword();

  if (mutation.isSuccess) {
    const email = getValues("email");
    return (
      <div
        className="season-transition flex flex-col gap-3 rounded-2xl px-5 py-6"
        style={{
          backgroundColor:
            "color-mix(in srgb, var(--season-accent) 8%, transparent)",
          border:
            "1px solid color-mix(in srgb, var(--season-accent) 28%, transparent)",
        }}
        role="status"
      >
        <div className="flex items-center gap-2">
          <Check
            className="season-transition size-4"
            style={{ color: "var(--season-accent)" }}
          />
          <p
            className="season-transition text-[11px] tracking-[0.2em] uppercase font-medium"
            style={{ color: "var(--season-accent)" }}
          >
            Check your inbox
          </p>
        </div>
        <p className="text-sm" style={{ color: "#ECEFEA" }}>
          If an account exists for <strong>{email}</strong>, we&apos;ve sent a
          link to reset its password. It expires in 30 minutes.
        </p>
        <p className="text-xs" style={{ color: "#8B9A8E" }}>
          Didn&apos;t get it? Check spam, or{" "}
          <button
            type="button"
            onClick={() => {
              mutation.reset();
              reset({ email: "" });
            }}
            className="season-transition underline-offset-4 hover:underline"
            style={{ color: "var(--season-accent)" }}
          >
            try again
          </button>
          .
        </p>
      </div>
    );
  }

  const submitError = errorMessage(mutation.error);

  return (
    <form
      onSubmit={handleSubmit((v) => mutation.mutate(v))}
      className="flex flex-col gap-5"
      noValidate
    >
      <Field
        id="email"
        label="Email"
        type="email"
        autoComplete="email"
        error={errors.email?.message}
        {...register("email")}
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
            Sending link
          </>
        ) : (
          <>
            Send reset link <ArrowRight className="size-4" />
          </>
        )}
      </button>
    </form>
  );
}

type FieldProps = React.InputHTMLAttributes<HTMLInputElement> & {
  id: string;
  label: string;
  error?: string;
};

function Field({ id, label, error, ...input }: FieldProps) {
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
      {error && (
        <p className="text-[11px]" style={{ color: "#FCA5A5" }}>
          {error}
        </p>
      )}
    </div>
  );
}
