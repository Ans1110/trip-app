"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { Loader2 } from "lucide-react";

import { errorMessage, useJoinRoom } from "@/hooks/trip-hooks";

import { Modal } from "./modal";

export function JoinRoomDialog({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const join = useJoinRoom();
  const [code, setCode] = useState("");

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const trimmed = code.trim().toUpperCase();
    if (!trimmed) return;
    join.mutate(
      { code: trimmed },
      {
        onSuccess: (result) => {
          onClose();
          router.push(`/trips/${result.trip_id}`);
        },
      },
    );
  };

  return (
    <Modal title="Join a trip" onClose={onClose}>
      <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Enter the room code shared with you, or scan a friend&rsquo;s QR code.
        </p>

        <label className="flex flex-col gap-1.5">
          <span className="text-xs font-medium" style={{ color: "#8B9A8E" }}>
            Room code
          </span>
          <input
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="ABCD1234"
            maxLength={16}
            autoFocus
            className="w-full px-3 py-2 rounded-lg text-sm uppercase tracking-widest outline-none focus:border-[color:var(--season-button)]"
            style={{
              backgroundColor: "#161E19",
              border: "1px solid #1F2A24",
              color: "#ECEFEA",
            }}
          />
        </label>

        {join.isError && (
          <p className="text-sm" style={{ color: "#FCA5A5" }}>
            {errorMessage(join.error)}
          </p>
        )}

        <div className="flex items-center justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={join.isPending}
            className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60"
            style={{ color: "#8B9A8E" }}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={join.isPending || !code.trim()}
            className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            {join.isPending && <Loader2 className="size-3.5 animate-spin" />}
            Join
          </button>
        </div>
      </form>
    </Modal>
  );
}
