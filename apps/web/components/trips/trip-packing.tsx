"use client";

import { useMemo, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Check, Loader2, Pencil, Plus, Trash2, X } from "lucide-react";

import { ControlledInput } from "@/components/ui/controlled-input";
import { Checkbox } from "@/components/ui/checkbox";
import { useMe } from "@/hooks/auth-hooks";
import {
  errorMessage,
  useCreatePackingItem,
  useDeletePackingItem,
  useSetPackingItemPacked,
  useUpdatePackingItem,
  usePackingItems,
} from "@/hooks/packing-hooks";
import { useRoom, useTrip } from "@/hooks/trip-hooks";
import { useTripRealtime } from "@/hooks/realtime-hooks";
import type {
  PackingItem,
  UpdatePackingItemInput,
} from "@/lib/apis/packing-api";
import type { UserSummary } from "@/lib/apis/trip-api";
import {
  createPackingItemSchema,
  type CreatePackingItemFormInput,
} from "@/lib/schemas/packing-schemas";

const UNCATEGORIZED = "Uncategorized";

export function TripPacking({ tripId }: { tripId: string }) {
  const query = usePackingItems(tripId);
  const roomQuery = useRoom(tripId);
  const tripQuery = useTrip(tripId);
  const meQuery = useMe();
  const create = useCreatePackingItem();

  useTripRealtime(tripId);

  const form = useForm<CreatePackingItemFormInput>({
    resolver: zodResolver(createPackingItemSchema),
    defaultValues: { name: "", quantity: "", category: "", note: "" },
  });
  const { handleSubmit, reset, watch } = form;
  const nameValue = watch("name");

  const [filter, setFilter] = useState<string>("all");

  const items = query.data ?? [];
  const me = meQuery.data;
  const isTripOwner = !!me && tripQuery.data?.owner.id === me.id;

  const memberById = useMemo(() => {
    const map = new Map<string, UserSummary>();
    for (const m of roomQuery.data?.members ?? []) map.set(m.user.id, m.user);
    return map;
  }, [roomQuery.data]);

  const categories = useMemo(() => {
    const set = new Set<string>();
    for (const i of items) set.add(i.category.trim() || UNCATEGORIZED);
    return [...set].sort((a, b) => a.localeCompare(b));
  }, [items]);

  const visible = useMemo(
    () =>
      items.filter((i) => {
        if (filter === "all") return true;
        if (filter === "mine") return !!me && i.created_by === me.id;
        return (i.category.trim() || UNCATEGORIZED) === filter;
      }),
    [items, filter, me],
  );

  const grouped = useMemo(() => {
    const buckets = new Map<string, PackingItem[]>();
    for (const item of visible) {
      const key = item.category.trim() || UNCATEGORIZED;
      const list = buckets.get(key) ?? [];
      list.push(item);
      buckets.set(key, list);
    }
    return [...buckets.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [visible]);

  const total = items.length;
  const packed = items.filter((i) => i.packed_by_me).length;

  const onSubmit = (v: CreatePackingItemFormInput) => {
    const qty = v.quantity.trim() === "" ? undefined : Number(v.quantity);
    create.mutate(
      {
        tripId,
        input: {
          name: v.name,
          quantity: qty,
          category: v.category || undefined,
          note: v.note || undefined,
        },
      },
      {
        onSuccess: () =>
          reset({ name: "", quantity: "", category: "", note: "" }),
      },
    );
  };

  return (
    <section className="flex flex-col gap-4">
      <FormProvider {...form}>
        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          className="flex flex-col gap-2.5 p-4 rounded-xl bg-[#121814] border border-[#1F2A24]"
        >
          <div className="flex items-end gap-2 flex-wrap">
            <ControlledInput<CreatePackingItemFormInput>
              name="name"
              placeholder="Add an item…"
              containerClassName="flex-1 min-w-[180px]"
            />
            <ControlledInput<CreatePackingItemFormInput>
              name="quantity"
              placeholder="Qty"
              inputMode="numeric"
              containerClassName="w-20"
            />
            <ControlledInput<CreatePackingItemFormInput>
              name="category"
              placeholder="Category"
              containerClassName="w-32"
            />
            <button
              type="submit"
              disabled={create.isPending || !nameValue.trim()}
              className="season-transition inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-full disabled:opacity-60 text-[#0B100D]"
              style={{ backgroundColor: "var(--season-button)" }}
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
            <p className="text-xs text-[#FCA5A5]">
              {errorMessage(create.error)}
            </p>
          )}
        </form>
      </FormProvider>

      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-xs" style={{ color: "#6B7A6F" }}>
          {packed}/{total} packed
        </span>
        <span className="text-xs" style={{ color: "#6B7A6F" }}>
          ·
        </span>
        <FilterChip
          label="All"
          active={filter === "all"}
          onClick={() => setFilter("all")}
        />
        <FilterChip
          label="Mine"
          active={filter === "mine"}
          onClick={() => setFilter("mine")}
        />
        {categories.map((c) => (
          <FilterChip
            key={c}
            label={c}
            active={filter === c}
            onClick={() => setFilter(c)}
          />
        ))}
      </div>

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
      {!query.isLoading && !query.isError && visible.length === 0 && (
        <p className="text-sm" style={{ color: "#8B9A8E" }}>
          {total === 0 ? "Nothing packed yet." : "No items match the filter."}
        </p>
      )}

      <div className="flex flex-col gap-4">
        {grouped.map(([category, rows]) => (
          <div key={category} className="flex flex-col gap-2">
            <h4
              className="text-xs uppercase tracking-wide"
              style={{ color: "#6B7A6F" }}
            >
              {category}
            </h4>
            <div className="flex flex-col gap-2">
              {rows.map((item) => (
                <PackingRow
                  key={item.id}
                  tripId={tripId}
                  item={item}
                  meId={me?.id}
                  isTripOwner={isTripOwner}
                  creator={memberById.get(item.created_by)}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="text-xs px-2 py-1 rounded-full season-transition"
      style={{
        backgroundColor: active ? "rgba(181,208,134,0.18)" : "transparent",
        color: active ? "#B5D086" : "#8B9A8E",
        border: `1px solid ${active ? "#B5D086" : "#1F2A24"}`,
      }}
    >
      {label}
    </button>
  );
}

function PackingRow({
  tripId,
  item,
  meId,
  isTripOwner,
  creator,
}: {
  tripId: string;
  item: PackingItem;
  meId: string | undefined;
  isTripOwner: boolean;
  creator: UserSummary | undefined;
}) {
  const update = useUpdatePackingItem();
  const remove = useDeletePackingItem();
  const setPacked = useSetPackingItemPacked();

  const canEdit = isTripOwner || (!!meId && item.created_by === meId);

  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState({
    name: item.name,
    quantity: String(item.quantity),
    category: item.category,
  });

  const startEdit = () => {
    setDraft({
      name: item.name,
      quantity: String(item.quantity),
      category: item.category,
    });
    setIsEditing(true);
  };

  const cancelEdit = () => setIsEditing(false);

  const save = () => {
    const name = draft.name.trim();
    const qty = Number(draft.quantity);
    if (!name || !Number.isInteger(qty) || qty < 1 || qty > 999) return;
    const patch: UpdatePackingItemInput = {};
    if (name !== item.name) patch.name = name;
    if (qty !== item.quantity) patch.quantity = qty;
    const nextCategory = draft.category.trim();
    if (nextCategory !== item.category) patch.category = nextCategory;
    if (Object.keys(patch).length === 0) {
      setIsEditing(false);
      return;
    }
    update.mutate(
      { tripId, itemId: item.id, input: patch },
      { onSuccess: () => setIsEditing(false) },
    );
  };

  return (
    <div
      className="flex items-center gap-3 px-4 py-3 rounded-lg"
      style={{ backgroundColor: "#121814", border: "1px solid #1F2A24" }}
    >
      <Checkbox
        checked={item.packed_by_me}
        disabled={!meId || setPacked.isPending}
        onCheckedChange={(v) =>
          setPacked.mutate({
            tripId,
            itemId: item.id,
            packed: v === true,
          })
        }
        aria-label={item.packed_by_me ? "Mark not packed" : "Mark packed"}
      />
      {isEditing ? (
        <>
          <input
            value={draft.name}
            onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
            onKeyDown={(e) => {
              if (e.key === "Enter") save();
              if (e.key === "Escape") cancelEdit();
            }}
            placeholder="Name"
            aria-label="Name"
            className="flex-1 min-w-0 bg-transparent outline-none text-sm border-b border-[#1F2A24] pb-0.5"
            style={{ color: "#ECEFEA" }}
          />
          <input
            value={draft.category}
            onChange={(e) =>
              setDraft((d) => ({ ...d, category: e.target.value }))
            }
            onKeyDown={(e) => {
              if (e.key === "Enter") save();
              if (e.key === "Escape") cancelEdit();
            }}
            placeholder="Category"
            aria-label="Category"
            className="w-28 bg-transparent outline-none text-xs border-b border-[#1F2A24] pb-0.5"
            style={{ color: "#9CB0A3" }}
          />
          <input
            value={draft.quantity}
            onChange={(e) =>
              setDraft((d) => ({ ...d, quantity: e.target.value }))
            }
            onKeyDown={(e) => {
              if (e.key === "Enter") save();
              if (e.key === "Escape") cancelEdit();
            }}
            inputMode="numeric"
            aria-label="Quantity"
            className="w-12 bg-transparent outline-none text-xs text-right border-b border-[#1F2A24] pb-0.5"
            style={{ color: "#9CB0A3" }}
          />
        </>
      ) : (
        <>
          <span
            className={`flex-1 min-w-0 truncate text-sm ${
              item.packed_by_me ? "line-through" : ""
            }`}
            style={{ color: item.packed_by_me ? "#6B7A6F" : "#ECEFEA" }}
          >
            {item.name}
          </span>
          <span
            className="w-12 text-xs text-right"
            style={{ color: "#9CB0A3" }}
          >
            {item.quantity}
          </span>
        </>
      )}
      {!isEditing && item.packed_count > 0 && (
        <span
          className="text-[10px] tabular-nums"
          style={{ color: "#6B7A6F" }}
          title={`${item.packed_count} member${
            item.packed_count === 1 ? "" : "s"
          } packed`}
        >
          {item.packed_count}✓
        </span>
      )}
      {!isEditing && <MiniAvatar user={creator} />}
      {canEdit && !isEditing && (
        <>
          <button
            type="button"
            onClick={startEdit}
            aria-label="Edit"
            className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5"
            style={{ color: "#9CB0A3" }}
          >
            <Pencil className="size-3.5" />
          </button>
          <button
            type="button"
            onClick={() => remove.mutate({ tripId, itemId: item.id })}
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
      {isEditing && (
        <>
          <button
            type="button"
            onClick={save}
            disabled={update.isPending || !draft.name.trim()}
            aria-label="Save"
            className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5 disabled:opacity-60"
            style={{ color: "#B5D086" }}
          >
            {update.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Check className="size-3.5" />
            )}
          </button>
          <button
            type="button"
            onClick={cancelEdit}
            aria-label="Cancel"
            className="inline-flex items-center justify-center size-7 rounded-full hover:bg-white/5"
            style={{ color: "#8B9A8E" }}
          >
            <X className="size-3.5" />
          </button>
        </>
      )}
    </div>
  );
}

function MiniAvatar({ user }: { user: UserSummary | undefined }) {
  const initial = (user?.name || user?.email || "?").charAt(0).toUpperCase();
  const title = user?.name || user?.email || "unknown";
  return (
    <span
      title={title}
      aria-label={`Added by ${title}`}
      className="size-6 rounded-full inline-flex items-center justify-center shrink-0 text-[10px] font-medium overflow-hidden"
      style={{
        backgroundColor: user?.avatar_url
          ? "transparent"
          : "color-mix(in srgb, var(--season-button) 22%, transparent)",
        color: "#ECEFEA",
      }}
    >
      {user?.avatar_url ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={user.avatar_url} alt="" className="size-full object-cover" />
      ) : (
        <span aria-hidden>{initial}</span>
      )}
    </span>
  );
}
