import type { Metadata } from "next";
import Link from "next/link";
import { Compass } from "lucide-react";

import { SignUpForm } from "@/components/auth/sign-up-form";

export const metadata: Metadata = {
  title: "Create account — TripCraft",
};

export default function SignUpPage() {
  return (
    <main
      className="bg-wash season-transition relative min-h-screen flex flex-col"
      style={{
        color: "var(--foreground)",
      }}
    >
      <header className="px-6 md:px-12 h-16 flex items-center">
        <Link href="/" className="flex items-center gap-2 select-none">
          <Compass
            className="season-transition size-5"
            style={{ color: "var(--mint-ink)" }}
          />
          <span
            className="text-xl tracking-tight"
            style={{ fontFamily: "var(--font-display), system-ui, sans-serif" }}
          >
            TripCraft
          </span>
        </Link>
      </header>

      <section className="flex-1 flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-md">
          <div className="mb-8">
            <h1
              className="season-transition leading-[1.05] tracking-tight mb-3"
              style={{
                fontFamily: "var(--font-display), system-ui, sans-serif",
                fontSize: "clamp(2rem, 4vw, 2.75rem)",
                color: "var(--foreground)",
              }}
            >
              Start planning.
            </h1>
            <p className="text-sm" style={{ color: "var(--muted-foreground)" }}>
              A quiet workspace for the part of travel before the travel.
            </p>
          </div>

          <SignUpForm />

          <p className="mt-8 text-sm" style={{ color: "var(--muted-foreground)" }}>
            Already have an account?{" "}
            <Link
              href="/sign-in"
              className="season-transition underline-offset-4 hover:underline"
              style={{ color: "var(--pink-ink)" }}
            >
              Sign in
            </Link>
          </p>
        </div>
      </section>
    </main>
  );
}
