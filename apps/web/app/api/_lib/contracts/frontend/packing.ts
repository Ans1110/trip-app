import "server-only";
import { UpstreamPackingItem } from "../upstream";

export type PackingItemView = {
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

export const toPackingItemView = (i: UpstreamPackingItem): PackingItemView => ({
  id: i.id,
  trip_id: i.trip_id,
  created_by: i.created_by,
  name: i.name,
  quantity: i.quantity,
  category: i.category,
  note: i.note,
  packed_by_me: i.packed_by_me,
  packed_count: i.packed_count,
  sort_order: i.sort_order,
  created_at: i.created_at,
  updated_at: i.updated_at,
});
