import type { Metadata } from "next";
import Link from "next/link";
import { Compass } from "lucide-react";

import { ForgotPasswordForm } from "@/components/auth/forgot-password-form";

export const metadata: Metadata = {
  title: "Reset your password — TripCraft",
};

export default function ForgotPasswordPage() {
  return (
    <main
      className="relative min-h-screen flex flex-col"
      style={{
        backgroundColor: "#0B100D",
        color: "#ECEFEA",
        backgroundImage:
          "radial-gradient(circle 600px at 80% -10%, rgba(127,182,138,0.18), transparent 70%), radial-gradient(circle 500px at 10% 110%, rgba(168,224,180,0.10), transparent 70%)",
      }}
    >
      <header className="px-6 md:px-12 h-16 flex items-center">
        <Link href="/" className="flex items-center gap-2 select-none">
          <Compass className="size-5" style={{ color: "#7FB68A" }} />
          <span
            className="text-xl tracking-tight"
            style={{ fontFamily: "var(--font-display, Georgia, serif)" }}
          >
            TripCraft
          </span>
        </Link>
      </header>

      <section className="flex-1 flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-md">
          <div className="mb-8">
            <h1
              className="leading-[1.05] tracking-tight mb-3"
              style={{
                fontFamily: "var(--font-display, Georgia, serif)",
                fontSize: "clamp(2rem, 4vw, 2.75rem)",
              }}
            >
              Forgot your password?
            </h1>
            <p className="text-sm" style={{ color: "#8B9A8E" }}>
              Enter the email on your account and we&apos;ll send a reset link.
            </p>
          </div>

          <ForgotPasswordForm />

          <p className="mt-8 text-sm" style={{ color: "#8B9A8E" }}>
            Remembered it?{" "}
            <Link
              href="/sign-in"
              className="underline-offset-4 hover:underline"
              style={{ color: "#A8E0B4" }}
            >
              Back to sign in
            </Link>
          </p>
        </div>
      </section>
    </main>
  );
}
