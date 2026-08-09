"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  mediaApi,
  type Asset,
  type CompleteUploadInput,
  type InitUpload,
  type InitUploadInput,
  type MediaPurpose,
  type PresignedGet,
} from "@/lib/apis/media-api";
import { ApiError } from "@/lib/utils";

export const mediaKeys = {
  asset: (id: string) => ["media", "asset", id] as const,
  url: (id: string) => ["media", "url", id] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

export function useAsset(id: string | null, options?: QueryOpts<Asset>) {
  return useQuery<Asset, ApiError>({
    queryKey: mediaKeys.asset(id ?? ""),
    queryFn: ({ signal }) => mediaApi.getAsset(id as string, signal),
    enabled: !!id,
    ...options,
  });
}

// Presigned GET URLs expire (backend default ~15 min). Refetch once staleTime
// passes so components hand out a fresh URL before it dies. Callers can pass
// a tighter staleTime if their storage config uses a shorter TTL.
export function useAssetUrl(
  id: string | null,
  options?: QueryOpts<PresignedGet>,
) {
  return useQuery<PresignedGet, ApiError>({
    queryKey: mediaKeys.url(id ?? ""),
    queryFn: ({ signal }) => mediaApi.presignRead(id as string, signal),
    enabled: !!id,
    staleTime: 5 * 60_000,
    ...options,
  });
}

export function useInitUpload(options?: MutOpts<InitUpload, InitUploadInput>) {
  return useMutation<InitUpload, ApiError, InitUploadInput>({
    mutationFn: (input) => mediaApi.initUpload(input),
    ...options,
  });
}

export function useCompleteUpload(
  options?: MutOpts<Asset, CompleteUploadInput>,
) {
  const qc = useQueryClient();
  return useMutation<Asset, ApiError, CompleteUploadInput>({
    mutationFn: (input) => mediaApi.completeUpload(input),
    ...options,
    onSuccess: (asset, ...rest) => {
      qc.setQueryData(mediaKeys.asset(asset.id), asset);
      return options?.onSuccess?.(asset, ...rest);
    },
  });
}

export type UploadFileInput = { purpose: MediaPurpose; file: File | Blob };

// useUploadFile runs the full init → browser PUT → complete handshake in one
// mutation, so components can just do `await upload({ purpose, file })` and
// receive the materialized Asset.
export function useUploadFile(options?: MutOpts<Asset, UploadFileInput>) {
  const qc = useQueryClient();
  return useMutation<Asset, ApiError, UploadFileInput>({
    mutationFn: ({ purpose, file }) => mediaApi.uploadFile(purpose, file),
    ...options,
    onSuccess: (asset, ...rest) => {
      qc.setQueryData(mediaKeys.asset(asset.id), asset);
      return options?.onSuccess?.(asset, ...rest);
    },
  });
}

export function useDeleteAsset(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (id) => mediaApi.softDelete(id),
    ...options,
    onSuccess: (data, id, ...rest) => {
      qc.removeQueries({ queryKey: mediaKeys.asset(id) });
      qc.removeQueries({ queryKey: mediaKeys.url(id) });
      return options?.onSuccess?.(data, id, ...rest);
    },
  });
}

export { errorMessage } from "@/lib/utils";
