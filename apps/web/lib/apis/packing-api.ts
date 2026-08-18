import { request } from "../utils";

export type PackingItem = {
  id: string;
  trip_id: string;
  created_by: string;
  name: string;
  quantity: number;
  category: string;
  note?: string;
  packed_by_me: boolean;
  packed_count: number;
  sort_order: number;
  created_at: string;
  updated_at: string;
};

export type CreatePackingItemInput = {
  name: string;
  quantity?: number;
  category?: string;
  note?: string;
  sort_order?: number;
};

export type UpdatePackingItemInput = {
  name?: string;
  quantity?: number;
  category?: string;
  note?: string;
  sort_order?: number;
};

export const packingApi = {
  listItems: (tripId: string, signal?: AbortSignal) =>
    request<PackingItem[]>(
      `/api/packing/trips/${encodeURIComponent(tripId)}/items`,
      { signal },
    ),
  createItem: (
    tripId: string,
    input: CreatePackingItemInput,
    signal?: AbortSignal,
  ) =>
    request<PackingItem>(
      `/api/packing/trips/${encodeURIComponent(tripId)}/items`,
      { method: "POST", json: input, signal },
    ),
  updateItem: (
    itemId: string,
    input: UpdatePackingItemInput,
    signal?: AbortSignal,
  ) =>
    request<PackingItem>(`/api/packing/items/${encodeURIComponent(itemId)}`, {
      method: "PATCH",
      json: input,
      signal,
    }),
  deleteItem: (itemId: string, signal?: AbortSignal) =>
    request<null>(`/api/packing/items/${encodeURIComponent(itemId)}`, {
      method: "DELETE",
      signal,
    }),
  packItem: (itemId: string, signal?: AbortSignal) =>
    request<PackingItem>(
      `/api/packing/items/${encodeURIComponent(itemId)}/pack`,
      { method: "POST", signal },
    ),
  unpackItem: (itemId: string, signal?: AbortSignal) =>
    request<PackingItem>(
      `/api/packing/items/${encodeURIComponent(itemId)}/pack`,
      { method: "DELETE", signal },
    ),
};
