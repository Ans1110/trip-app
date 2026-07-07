"use client";

import { Check, Loader2, ShieldOff, X } from "lucide-react";

import {
  errorMessage,
  useAcceptRequest,
  useBlockUser,
  useCancelRequest,
  useDeclineRequest,
  useIncomingRequests,
  useOutgoingRequests,
} from "@/hooks/friend-hooks";
import type { FriendRequest, UserSummary } from "@/lib/apis/friend-api";

import { Avatar, Section } from "./friend-list";

export function RequestsPanel() {
  const incoming = useIncomingRequests();
  const outgoing = useOutgoingRequests();

  return (
    <div className="flex flex-col gap-8">
      <Section
        title="Incoming requests"
        empty="No one has asked to connect yet."
        loading={incoming.isLoading}
        error={errorMessage(incoming.error)}
        items={incoming.data ?? []}
        renderItem={(r) => <IncomingRow key={r.id} request={r} />}
      />
      <Section
        title="Sent requests"
        empty="You haven't sent any requests."
        loading={outgoing.isLoading}
        error={errorMessage(outgoing.error)}
        items={outgoing.data ?? []}
        renderItem={(r) => <OutgoingRow key={r.id} request={r} />}
      />
    </div>
  );
}

function IncomingRow({ request }: { request: FriendRequest }) {
  const accept = useAcceptRequest();
  const decline = useDeclineRequest();
  const block = useBlockUser();
  const busy = accept.isPending || decline.isPending || block.isPending;

  return (
    <RequestRow user={request.sender} message={request.message}>
      <button
        type="button"
        onClick={() => accept.mutate(request.id)}
        disabled={busy}
        className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full disabled:opacity-60"
        style={{
          backgroundColor:
            "color-mix(in srgb, var(--season-button) 22%, transparent)",
          color: "#ECEFEA",
          border:
            "1px solid color-mix(in srgb, var(--season-button) 32%, transparent)",
        }}
      >
        {accept.isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <Check className="size-3.5" />
        )}
        Accept
      </button>
      <button
        type="button"
        onClick={() => decline.mutate(request.id)}
        disabled={busy}
        className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
        style={{ color: "#8B9A8E" }}
      >
        {decline.isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <X className="size-3.5" />
        )}
        Decline
      </button>
      <button
        type="button"
        onClick={() => {
          if (
            window.confirm(
              `Block ${request.sender.name || request.sender.email}? Their request will be declined.`,
            )
          ) {
            block.mutate({ target_id: request.sender.id });
          }
        }}
        disabled={busy}
        aria-label="Block"
        title="Block"
        className="season-transition inline-flex items-center justify-center size-8 rounded-full hover:bg-white/5 disabled:opacity-60"
        style={{ color: "#8B9A8E" }}
      >
        {block.isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <ShieldOff className="size-3.5" />
        )}
      </button>
    </RequestRow>
  );
}

function OutgoingRow({ request }: { request: FriendRequest }) {
  const cancel = useCancelRequest();
  return (
    <RequestRow user={request.receiver} message={request.message}>
      <button
        type="button"
        onClick={() => cancel.mutate(request.id)}
        disabled={cancel.isPending}
        className="season-transition inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
        style={{ color: "#FCA5A5" }}
      >
        {cancel.isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <X className="size-3.5" />
        )}
        Cancel
      </button>
    </RequestRow>
  );
}

function RequestRow({
  user,
  message,
  children,
}: {
  user: UserSummary;
  message?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <Avatar user={user} />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate" style={{ color: "#ECEFEA" }}>
          {user.name || user.email}
        </p>
        <p className="text-xs truncate" style={{ color: "#8B9A8E" }}>
          {message ? message : user.email}
        </p>
      </div>
      <div className="flex items-center gap-1.5 shrink-0">{children}</div>
    </div>
  );
}
