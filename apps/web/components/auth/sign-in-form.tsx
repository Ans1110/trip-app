"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, Loader2 } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { signInSchema, type SignInInput } from "@/lib/schemas/auth-schemas";
import { errorMessage, useSignIn } from "@/hooks/auth-hooks";

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

  const form = useForm<SignInInput>({
    resolver: zodResolver(signInSchema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useSignIn({
    onSuccess: () => window.location.assign(next),
  });

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
      </form>
    </FormProvider>
  );
}
