"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import { ApiError } from "@/lib/utils";
import {
  packingApi,
  type CreatePackingItemInput,
  type PackingItem,
  type UpdatePackingItemInput,
} from "@/lib/apis/packing-api";

export { errorMessage } from "@/hooks/trip-hooks";

export const packingKeys = {
  items: (tripId: string) => ["packing", "items", tripId] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function usePackingItems(
  tripId: string | undefined,
  options?: QueryOpts<PackingItem[]>,
) {
  return useQuery<PackingItem[], ApiError>({
    queryKey: packingKeys.items(tripId ?? ""),
    queryFn: ({ signal }) => packingApi.listItems(tripId!, signal),
    enabled: !!tripId,
    ...options,
  });
}

type CreateItemVars = { tripId: string; input: CreatePackingItemInput };

export function useCreatePackingItem(
  options?: MutOpts<PackingItem, CreateItemVars>,
) {
  const qc = useQueryClient();
  return useMutation<PackingItem, ApiError, CreateItemVars>({
    mutationFn: ({ tripId, input }) => packingApi.createItem(tripId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: packingKeys.items(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type UpdateItemVars = {
  tripId: string;
  itemId: string;
  input: UpdatePackingItemInput;
};

export function useUpdatePackingItem(
  options?: MutOpts<PackingItem, UpdateItemVars>,
) {
  const qc = useQueryClient();
  return useMutation<PackingItem, ApiError, UpdateItemVars>({
    mutationFn: ({ itemId, input }) => packingApi.updateItem(itemId, input),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: packingKeys.items(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type DeleteItemVars = { tripId: string; itemId: string };

export function useDeletePackingItem(options?: MutOpts<null, DeleteItemVars>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, DeleteItemVars>({
    mutationFn: ({ itemId }) => packingApi.deleteItem(itemId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: packingKeys.items(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}

type PackItemVars = { tripId: string; itemId: string; packed: boolean };

export function useSetPackingItemPacked(
  options?: MutOpts<PackingItem, PackItemVars>,
) {
  const qc = useQueryClient();
  return useMutation<PackingItem, ApiError, PackItemVars>({
    mutationFn: ({ itemId, packed }) =>
      packed ? packingApi.packItem(itemId) : packingApi.unpackItem(itemId),
    ...options,
    onSuccess: (data, vars, ...rest) => {
      qc.invalidateQueries({ queryKey: packingKeys.items(vars.tripId) });
      return options?.onSuccess?.(data, vars, ...rest);
    },
  });
}
