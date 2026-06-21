"use client";

import { useMemo, useState, type FormEvent } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";

import {
  errorMessage,
  useCreateItinerary,
  useDeleteItinerary,
  useItinerary,
  useUpdateItinerary,
} from "@/hooks/trip-hooks";
import type { Itinerary } from "@/lib/trip-api";

export function TripItinerary({ tripId }: { tripId: string }) {
  const query = useItinerary(tripId);

  const grouped = useMemo(() => {
    const items = query.data ?? [];
    const byDay = new Map<number, Itinerary[]>();
    for (const item of items) {
      const list = byDay.get(item.day) ?? [];
      list.push(item);
      byDay.set(item.day, list);
    }
    for (const list of byDay.values()) {
      list.sort((a, b) => a.sort_order - b.sort_order);
    }
    return [...byDay.entries()].sort(([a], [b]) => a - b);
  }, [query.data]);

  const itemCount = query.data?.length ?? 0;

  return (
    <section className="flex flex-col gap-6">
      <NewItemForm tripId={tripId} />

      {query.isLoading && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          Loading…
        </p>
      )}
      {query.isError && (
        <p className="text-sm" style={{ color: "#FCA5A5" }}>
          {errorMessage(query.error)}
        </p>
      )}
      {!query.isLoading && !query.isError && itemCount === 0 && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          No itinerary items yet. Add the first stop above.
        </p>
      )}

      {grouped.map(([day, list]) => (
        <div key={day} className="flex flex-col gap-3">
          <h3
            className="text-sm font-medium tracking-wider uppercase"
            style={{ color: "var(--season-button)" }}
          >
            Day {day}
          </h3>
          <div className="flex flex-col gap-2">
            {list.map((item) => (
              <ItemRow key={item.id} tripId={tripId} item={item} />
            ))}
          </div>
        </div>
      ))}
    </section>
  );
}

function NewItemForm({ tripId }: { tripId: string }) {
  const create = useCreateItinerary();
  const [day, setDay] = useState("1");
  const [title, setTitle] = useState("");
  const [location, setLocation] = useState("");

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const parsedDay = Number(day);
    if (!Number.isFinite(parsedDay) || parsedDay < 1) return;
    create.mutate(
      {
        id: tripId,
        input: {
          day: parsedDay,
          title: title.trim() || undefined,
          location: location.trim() || undefined,
        },
      },
      {
        onSuccess: () => {
          setTitle("");
          setLocation("");
        },
      },
    );
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-3 p-4 rounded-xl"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <div className="grid grid-cols-1 sm:grid-cols-[80px_1fr_1fr_auto] gap-2 items-end">
        <Field label="Day">
          <input
            type="number"
            min={1}
            value={day}
            onChange={(e) => setDay(e.target.value)}
            className={inputClass}
            style={inputStyle}
            required
          />
        </Field>
        <Field label="Title">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Senso-ji visit"
            className={inputClass}
            style={inputStyle}
          />
        </Field>
        <Field label="Location">
          <input
            value={location}
            onChange={(e) => setLocation(e.target.value)}
            placeholder="Tokyo"
            className={inputClass}
            style={inputStyle}
          />
        </Field>
        <button
          type="submit"
          disabled={create.isPending}
          className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60"
          style={{
            backgroundColor: "var(--season-button)",
            color: "#0B100D",
          }}
        >
          {create.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Plus className="size-4" />
          )}
          Add
        </button>
      </div>
      {create.isError && (
        <p className="text-xs" style={{ color: "#FCA5A5" }}>
          {errorMessage(create.error)}
        </p>
      )}
    </form>
  );
}

function ItemRow({ tripId, item }: { tripId: string; item: Itinerary }) {
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(item.title);
  const [description, setDescription] = useState(item.description);
  const [location, setLocation] = useState(item.location);
  const update = useUpdateItinerary();
  const remove = useDeleteItinerary();

  const handleSave = () => {
    update.mutate(
      {
        id: tripId,
        itemId: item.id,
        input: {
          title: title.trim(),
          description: description.trim(),
          location: location.trim(),
        },
      },
      { onSuccess: () => setEditing(false) },
    );
  };

  return (
    <div
      className="px-4 py-3 rounded-lg flex items-start gap-3"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <div className="flex-1 min-w-0">
        {editing ? (
          <div className="flex flex-col gap-2">
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Title"
              className={inputClass}
              style={inputStyle}
            />
            <input
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="Location"
              className={inputClass}
              style={inputStyle}
            />
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Description"
              rows={2}
              className={`${inputClass} resize-none`}
              style={inputStyle}
            />
          </div>
        ) : (
          <>
            <p className="text-sm" style={{ color: "#ECEFEA" }}>
              {item.title || "Untitled"}
            </p>
            {item.location && (
              <p className="text-xs mt-0.5" style={{ color: "#8B9A8E" }}>
                {item.location}
              </p>
            )}
            {item.description && (
              <p className="text-xs mt-1 whitespace-pre-wrap" style={{ color: "#6B7A6F" }}>
                {item.description}
              </p>
            )}
          </>
        )}
      </div>
      <div className="flex items-center gap-1.5 shrink-0">
        {editing ? (
          <>
            <button
              type="button"
              onClick={() => setEditing(false)}
              className="text-xs px-2 py-1 rounded-full hover:bg-white/5"
              style={{ color: "#8B9A8E" }}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={update.isPending}
              className="season-transition inline-flex items-center gap-1 text-xs px-3 py-1 rounded-full disabled:opacity-60"
              style={{
                backgroundColor: "var(--season-button)",
                color: "#0B100D",
              }}
            >
              {update.isPending && <Loader2 className="size-3 animate-spin" />}
              Save
            </button>
          </>
        ) : (
          <>
            <button
              type="button"
              onClick={() => setEditing(true)}
              className="text-xs px-2 py-1 rounded-full hover:bg-white/5"
              style={{ color: "#8B9A8E" }}
            >
              Edit
            </button>
            <button
              type="button"
              onClick={() => remove.mutate({ id: tripId, itemId: item.id })}
              disabled={remove.isPending}
              aria-label="Delete"
              className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 disabled:opacity-60"
              style={{ color: "#FCA5A5" }}
            >
              {remove.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Trash2 className="size-3.5" />
              )}
            </button>
          </>
        )}
      </div>
    </div>
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
    <label className="flex flex-col gap-1">
      <span className="text-[11px] font-medium" style={{ color: "#8B9A8E" }}>
        {label}
      </span>
      {children}
    </label>
  );
}

const inputClass =
  "w-full px-3 py-2 rounded-lg text-sm outline-none focus:border-[color:var(--season-button)]";
const inputStyle: React.CSSProperties = {
  backgroundColor: "#161E19",
  border: "1px solid #1F2A24",
  color: "#ECEFEA",
};
