"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { Loader2 } from "lucide-react";

import { errorMessage, useCreateTrip } from "@/hooks/trip-hooks";

import { Modal } from "./modal";

export function CreateTripDialog({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const create = useCreateTrip();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!title.trim() || !startDate || !endDate) return;
    create.mutate(
      {
        title: title.trim(),
        description: description.trim() || undefined,
        start_date: startDate,
        end_date: endDate,
      },
      {
        onSuccess: (trip) => {
          onClose();
          router.push(`/trips/${trip.id}`);
        },
      },
    );
  };

  return (
    <Modal title="Create a trip" onClose={onClose}>
      <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
        <Field label="Title">
          <input
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Japan · Spring 2026"
            className="w-full px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]"
            style={{
              backgroundColor: "#161E19",
              border: "1px solid #1F2A24",
              color: "#ECEFEA",
            }}
          />
        </Field>

        <Field label="Description (optional)">
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="What's this trip about?"
            rows={3}
            className="w-full px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)] resize-none"
            style={{
              backgroundColor: "#161E19",
              border: "1px solid #1F2A24",
              color: "#ECEFEA",
            }}
          />
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Start date">
            <input
              required
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="w-full px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]"
              style={{
                backgroundColor: "#161E19",
                border: "1px solid #1F2A24",
                color: "#ECEFEA",
              }}
            />
          </Field>
          <Field label="End date">
            <input
              required
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              min={startDate || undefined}
              className="w-full px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]"
              style={{
                backgroundColor: "#161E19",
                border: "1px solid #1F2A24",
                color: "#ECEFEA",
              }}
            />
          </Field>
        </div>

        {create.isError && (
          <p className="text-sm" style={{ color: "#FCA5A5" }}>
            {errorMessage(create.error)}
          </p>
        )}

        <div className="flex items-center justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={create.isPending}
            className="px-4 py-2 text-sm rounded-full hover:bg-white/5 disabled:opacity-60"
            style={{ color: "#8B9A8E" }}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={create.isPending}
            className="season-transition inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
            style={{
              backgroundColor: "var(--season-button)",
              color: "#0B100D",
            }}
          >
            {create.isPending && <Loader2 className="size-3.5 animate-spin" />}
            Create
          </button>
        </div>
      </form>
    </Modal>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="text-xs font-medium" style={{ color: "#8B9A8E" }}>
        {label}
      </span>
      {children}
    </label>
  );
}
