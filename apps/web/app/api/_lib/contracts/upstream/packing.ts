import "server-only";

export type UpstreamPackingItem = {
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

export type UpstreamCreatePackingItemPayload = {
  name: string;
  quantity?: number;
  category?: string;
  note?: string;
  sort_order?: number;
};

export type UpstreamUpdatePackingItemPayload = {
  name?: string;
  quantity?: number;
  category?: string;
  note?: string;
  sort_order?: number;
};
