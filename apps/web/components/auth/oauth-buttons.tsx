"use client";

import { useCallback, useEffect, useRef } from "react";
import Image from "next/image";
import Script from "next/script";
import { Loader2 } from "lucide-react";

import {
  errorMessage,
  useOAuthGithub,
  useOAuthGoogle,
} from "@/hooks/auth-hooks";
import type { SessionView } from "@/lib/apis/auth-api";

const GITHUB_AUTHORIZE_URL = "https://github.com/login/oauth/authorize";
const GITHUB_STATE_KEY = "oauth:github:state";
const GITHUB_NEXT_KEY = "oauth:github:next";

const CIRCLE_SIZE = 44;

type GoogleCredentialResponse = { credential?: string };

type GoogleIdApi = {
  initialize: (config: {
    client_id: string;
    callback: (response: GoogleCredentialResponse) => void;
    use_fedcm_for_prompt?: boolean;
    ux_mode?: "popup" | "redirect";
  }) => void;
  renderButton: (
    parent: HTMLElement,
    options: {
      theme?: "outline" | "filled_blue" | "filled_black";
      size?: "small" | "medium" | "large";
      type?: "standard" | "icon";
      shape?: "rectangular" | "pill" | "circle" | "square";
    },
  ) => void;
};

declare global {
  interface Window {
    google?: { accounts?: { id?: GoogleIdApi } };
  }
}

export type OAuthButtonsProps = {
  next: string;
  onSession: (session: SessionView) => void;
};

export function OAuthButtons({ next, onSession }: OAuthButtonsProps) {
  const googleClientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;
  const githubClientId = process.env.NEXT_PUBLIC_GITHUB_CLIENT_ID;

  const google = useOAuthGoogle({ onSuccess: onSession });
  const github = useOAuthGithub({ onSuccess: onSession });

  const errorMsg =
    errorMessage(google.error) ?? errorMessage(github.error) ?? null;

  if (!googleClientId && !githubClientId) return null;

  const disabled = google.isPending || github.isPending;

  return (
    <div className="flex flex-col items-center gap-3">
      <div className="flex items-center gap-3">
        {googleClientId && (
          <GoogleButton
            clientId={googleClientId}
            disabled={disabled}
            pending={google.isPending}
            onCredential={(idToken) => google.mutate(idToken)}
          />
        )}
        {githubClientId && (
          <GithubButton
            clientId={githubClientId}
            next={next}
            disabled={disabled}
            pending={github.isPending}
          />
        )}
      </div>

      {errorMsg && (
        <p
          className="text-xs rounded-lg px-3 py-2 self-stretch"
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

function GoogleButton({
  clientId,
  disabled,
  pending,
  onCredential,
}: {
  clientId: string;
  disabled: boolean;
  pending: boolean;
  onCredential: (idToken: string) => void;
}) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const callbackRef = useRef(onCredential);

  useEffect(() => {
    callbackRef.current = onCredential;
  });

  const render = useCallback(() => {
    const parent = containerRef.current;
    if (!parent) return;
    const id = window.google?.accounts?.id;
    if (!id) return;
    id.initialize({
      client_id: clientId,
      callback: (res) => {
        if (res.credential) callbackRef.current(res.credential);
      },
      use_fedcm_for_prompt: true,
      ux_mode: "popup",
    });
    parent.replaceChildren();
    id.renderButton(parent, {
      theme: "outline",
      size: "large",
      type: "icon",
      shape: "circle",
    });
  }, [clientId]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (window.google?.accounts?.id) render();
  }, [render]);

  return (
    <>
      <Script
        src="https://accounts.google.com/gsi/client"
        strategy="afterInteractive"
        onLoad={render}
      />
      <div
        className="relative"
        style={{
          width: CIRCLE_SIZE,
          height: CIRCLE_SIZE,
          opacity: disabled ? 0.6 : 1,
        }}
      >
        <div
          aria-hidden="true"
          className="season-transition absolute inset-0 rounded-full flex items-center justify-center pointer-events-none"
          style={{
            backgroundColor: "#ECEFEA",
            border:
              "1px solid color-mix(in srgb, var(--season-accent) 32%, transparent)",
          }}
        >
          {pending ? (
            <Loader2
              className="size-4 animate-spin"
              style={{ color: "#0B100D" }}
            />
          ) : (
            <GoogleMark />
          )}
        </div>
        <div
          ref={containerRef}
          aria-label="Continue with Google"
          className="absolute inset-0 overflow-hidden rounded-full"
          style={{
            opacity: 0.001,
            pointerEvents: disabled ? "none" : "auto",
            cursor: disabled ? "not-allowed" : "pointer",
          }}
        />
      </div>
    </>
  );
}

function GoogleMark() {
  return (
    <Image
      src="/google-color.svg"
      alt=""
      width={20}
      height={20}
      className="size-5"
      aria-hidden="true"
    />
  );
}

function GithubButton({
  clientId,
  next,
  disabled,
  pending,
}: {
  clientId: string;
  next: string;
  disabled: boolean;
  pending: boolean;
}) {
  const handleClick = () => {
    if (typeof window === "undefined") return;
    const state = crypto.randomUUID();
    sessionStorage.setItem(GITHUB_STATE_KEY, state);
    sessionStorage.setItem(GITHUB_NEXT_KEY, next);
    const redirectUri = `${window.location.origin}/oauth/github/callback`;
    const url = new URL(GITHUB_AUTHORIZE_URL);
    url.searchParams.set("client_id", clientId);
    url.searchParams.set("scope", "read:user user:email");
    url.searchParams.set("state", state);
    url.searchParams.set("redirect_uri", redirectUri);
    window.location.assign(url.toString());
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={disabled}
      aria-label="Continue with GitHub"
      title="Continue with GitHub"
      className="season-transition inline-flex items-center justify-center rounded-full disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
      style={{
        width: CIRCLE_SIZE,
        height: CIRCLE_SIZE,
        backgroundColor: "#0B100D",
        color: "#ECEFEA",
        border:
          "1px solid color-mix(in srgb, var(--season-accent) 32%, transparent)",
      }}
    >
      {pending ? <Loader2 className="size-4 animate-spin" /> : <GithubMark />}
    </button>
  );
}

function GithubMark() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="size-5"
      aria-hidden="true"
      fill="currentColor"
    >
      <path d="M12 .5C5.65.5.5 5.65.5 12a11.5 11.5 0 007.86 10.92c.58.1.79-.25.79-.56v-2c-3.2.7-3.88-1.36-3.88-1.36-.53-1.34-1.3-1.7-1.3-1.7-1.06-.72.08-.7.08-.7 1.17.08 1.79 1.2 1.79 1.2 1.04 1.78 2.73 1.27 3.4.97.11-.76.41-1.27.74-1.56-2.55-.29-5.24-1.28-5.24-5.7 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.47.11-3.06 0 0 .98-.31 3.2 1.18a11.06 11.06 0 015.83 0c2.22-1.5 3.2-1.18 3.2-1.18.63 1.6.23 2.77.11 3.06.74.81 1.19 1.84 1.19 3.1 0 4.44-2.7 5.4-5.26 5.69.42.36.79 1.08.79 2.18v3.23c0 .31.21.67.8.56A11.5 11.5 0 0023.5 12C23.5 5.65 18.35.5 12 .5z" />
    </svg>
  );
}
