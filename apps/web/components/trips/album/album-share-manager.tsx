"use client";

import { useMemo, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Check, Copy, Link2, Loader2, Trash2 } from "lucide-react";

import { Modal } from "@/components/trips/modal";
import { ControlledDatePicker } from "@/components/ui/controlled-date-picker";
import { ISO_DATE_FORMAT } from "@/components/ui/date-picker";
import { format } from "date-fns";
import {
  errorMessage,
  useAlbumShares,
  useCreateAlbumShare,
  useRevokeAlbumShare,
} from "@/hooks/album-hooks";
import type { AlbumShareToken } from "@/lib/apis/album-api";
import {
  albumShareCreateSchema,
  parseExpiryEndOfDay,
  type AlbumShareCreateFormInput,
} from "@/lib/schemas/album-schemas";

export function AlbumShareManagerModal({
  tripId,
  onClose,
}: {
  tripId: string;
  onClose: () => void;
}) {
  return (
    <Modal title="Share this album" onClose={onClose}>
      <AlbumShareManager tripId={tripId} />
    </Modal>
  );
}

export function AlbumShareManager({ tripId }: { tripId: string }) {
  const shares = useAlbumShares(tripId);
  const create = useCreateAlbumShare();
  const revoke = useRevokeAlbumShare();

  // Backend returns the plaintext token exactly once; hold it locally so the
  // user can copy the URL before it becomes unrecoverable.
  const [freshToken, setFreshToken] = useState<string | null>(null);

  const form = useForm<AlbumShareCreateFormInput>({
    resolver: zodResolver(albumShareCreateSchema),
    defaultValues: { expires_at: "" },
  });

  const onSubmit = (v: AlbumShareCreateFormInput) => {
    const picked = v.expires_at ? parseExpiryEndOfDay(v.expires_at) : null;
    create.mutate(
      {
        tripId,
        input: picked ? { expires_at: picked.toISOString() } : {},
      },
      {
        onSuccess: (data) => {
          setFreshToken(data.token);
          form.reset({ expires_at: "" });
        },
      },
    );
  };

  const todayISO = format(new Date(), ISO_DATE_FORMAT);

  const rows = useMemo(
    () => (shares.data ? sortShares(shares.data) : []),
    [shares.data],
  );

  return (
    <div className="flex flex-col gap-5">
      {freshToken && <FreshTokenBanner token={freshToken} />}

      <section
        className="rounded-xl p-4"
        style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
      >
        <FormProvider {...form}>
          <form
            className="flex flex-col gap-3"
            onSubmit={form.handleSubmit(onSubmit)}
            noValidate
          >
            <p className="text-[11px]" style={{ color: "#8B9A8E" }}>
              Anyone with the link can view this album. Leave the expiry blank
              for no expiration.
            </p>
            <ControlledDatePicker<AlbumShareCreateFormInput>
              name="expires_at"
              label="Expires at (optional)"
              placeholder="Pick a date"
              min={todayISO}
              hint="Link stays live through the end of the selected day."
            />
            {create.error && (
              <p className="text-xs text-[#FCA5A5]">
                {errorMessage(create.error)}
              </p>
            )}
            <button
              type="submit"
              disabled={create.isPending}
              className="season-transition inline-flex items-center gap-2 self-start px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
            >
              {create.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              <Link2 size={14} /> Create share link
            </button>
          </form>
        </FormProvider>
      </section>

      <section className="flex flex-col gap-2">
        <h3
          className="text-xs uppercase tracking-wider"
          style={{ color: "#8B9A8E" }}
        >
          Existing links
        </h3>

        {shares.isLoading && (
          <p className="text-xs" style={{ color: "#8B9A8E" }}>
            Loading…
          </p>
        )}
        {shares.isError && (
          <p className="text-xs text-[#FCA5A5]">
            {errorMessage(shares.error) ?? "Failed to load share links."}
          </p>
        )}
        {shares.data && rows.length === 0 && (
          <p className="text-xs" style={{ color: "#6B7A6F" }}>
            No share links yet.
          </p>
        )}

        <ul className="flex flex-col gap-2">
          {rows.map((s) => (
            <ShareRow
              key={s.id}
              share={s}
              onRevoke={() => revoke.mutate({ tokenId: s.id, tripId })}
              revoking={revoke.isPending && revoke.variables?.tokenId === s.id}
            />
          ))}
        </ul>
        {revoke.error && (
          <p className="text-xs text-[#FCA5A5]">{errorMessage(revoke.error)}</p>
        )}
      </section>
    </div>
  );
}

function FreshTokenBanner({ token }: { token: string }) {
  const url = buildPublicUrl(token);
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div
      className="rounded-xl p-4 flex flex-col gap-3"
      style={{
        backgroundColor: "rgba(134, 239, 172, 0.08)",
        border: "1px solid rgba(134, 239, 172, 0.28)",
      }}
    >
      <div>
        <p className="text-xs font-medium" style={{ color: "#86EFAC" }}>
          New share link ready
        </p>
        <p className="text-[11px] mt-1" style={{ color: "#8B9A8E" }}>
          Copy this URL now — for security we only store a hash of the token, so
          it can&apos;t be shown again after you close this dialog.
        </p>
      </div>
      <div className="flex items-center gap-2">
        <input
          readOnly
          value={url}
          className="flex-1 truncate rounded-md px-2 py-1.5 text-xs"
          style={{
            backgroundColor: "#0B100D",
            border: "1px solid #1F2A24",
            color: "#ECEFEA",
          }}
          onFocus={(e) => e.currentTarget.select()}
        />
        <button
          type="button"
          onClick={copy}
          className="inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-xs"
          style={{
            backgroundColor:
              "color-mix(in srgb, var(--season-button) 22%, transparent)",
            color: "#ECEFEA",
            border: "1px solid #1F2A24",
          }}
        >
          {copied ? <Check size={12} /> : <Copy size={12} />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
    </div>
  );
}

function ShareRow({
  share,
  onRevoke,
  revoking,
}: {
  share: AlbumShareToken;
  onRevoke: () => void;
  revoking: boolean;
}) {
  const state = shareState(share);
  return (
    <li
      className="flex items-center gap-3 rounded-lg px-3 py-2"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <div className="flex flex-col flex-1 min-w-0">
        <span
          className="text-xs truncate font-mono"
          style={{ color: "#ECEFEA" }}
        >
          {share.id}
        </span>
        <span className="text-[10px]" style={{ color: "#8B9A8E" }}>
          {formatShareMeta(share)}
        </span>
      </div>
      <span
        className="text-[10px] uppercase tracking-wider"
        style={{ color: state.color }}
      >
        {state.label}
      </span>
      {state.kind === "active" && (
        <button
          type="button"
          onClick={onRevoke}
          disabled={revoking}
          aria-label="Revoke link"
          title="Revoke link"
          className="inline-flex items-center gap-1 rounded-full px-2 py-1 text-[11px] disabled:opacity-60"
          style={{ color: "#FCA5A5" }}
        >
          {revoking ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <Trash2 size={12} />
          )}
          Revoke
        </button>
      )}
    </li>
  );
}

type ShareState =
  | { kind: "active"; label: string; color: string }
  | { kind: "revoked"; label: "REVOKED"; color: string }
  | { kind: "expired"; label: "EXPIRED"; color: string };

function shareState(s: AlbumShareToken): ShareState {
  if (s.revoked_at)
    return { kind: "revoked", label: "REVOKED", color: "#FCA5A5" };
  if (s.expires_at && new Date(s.expires_at).getTime() <= Date.now()) {
    return { kind: "expired", label: "EXPIRED", color: "#F59E0B" };
  }
  return { kind: "active", label: "Active", color: "#86EFAC" };
}

function formatShareMeta(s: AlbumShareToken): string {
  const created = `Created ${formatDate(s.created_at)}`;
  if (s.revoked_at) return `${created} · revoked ${formatDate(s.revoked_at)}`;
  if (s.expires_at) return `${created} · expires ${formatDate(s.expires_at)}`;
  return created;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

function sortShares(list: AlbumShareToken[]): AlbumShareToken[] {
  return [...list].sort((a, b) => {
    const av = a.revoked_at ? 1 : 0;
    const bv = b.revoked_at ? 1 : 0;
    if (av !== bv) return av - bv;
    return b.created_at.localeCompare(a.created_at);
  });
}

// Plain helper with no logging so the token never lands in an analytics pipeline.
export function buildPublicUrl(token: string): string {
  if (typeof window === "undefined")
    return `/albums/${encodeURIComponent(token)}`;
  return `${window.location.origin}/albums/${encodeURIComponent(token)}`;
}
