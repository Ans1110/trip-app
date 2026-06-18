"use client";

import Link from "next/link";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, Loader2 } from "lucide-react";

import { signInSchema, type SignInInput } from "@/lib/auth-schemas";
import { errorMessage, useSignIn } from "@/hooks/auth-hooks";

export function SignInForm() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SignInInput>({
    resolver: zodResolver(signInSchema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useSignIn({
    onSuccess: () => window.location.assign("/"),
  });

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
      <Field
        id="password"
        label="Password"
        type="password"
        autoComplete="current-password"
        error={errors.password?.message}
        trailing={
          <Link
            href="/forgot-password"
            className="text-xs tracking-wide hover:underline"
            style={{ color: "#7FB68A" }}
          >
            Forgot?
          </Link>
        }
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
        className="inline-flex items-center justify-center gap-2 px-5 py-3 rounded-full text-sm font-medium tracking-wide transition-all hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-60 disabled:translate-y-0"
        style={{
          backgroundColor: "#7FB68A",
          color: "#0B100D",
          boxShadow: "0 8px 24px rgba(127,182,138,0.18)",
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
    </form>
  );
}

type FieldProps = React.InputHTMLAttributes<HTMLInputElement> & {
  id: string;
  label: string;
  trailing?: React.ReactNode;
  error?: string;
};

function Field({ id, label, trailing, error, ...input }: FieldProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between">
        <label
          htmlFor={id}
          className="text-[11px] tracking-[0.2em] uppercase font-medium"
          style={{ color: "#8B9A8E" }}
        >
          {label}
        </label>
        {trailing}
      </div>
      <input
        id={id}
        {...input}
        className="px-4 py-3 rounded-lg text-sm outline-none transition-colors focus:border-[#7FB68A]"
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
