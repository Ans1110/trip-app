"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/react-query";

import {
  chatApi,
  type ChatMessage,
  type ChatReadReceipt,
  type ChatRoom,
  type EnsureDMInput,
  type ListMessagesQuery,
  type MarkReadInput,
} from "@/lib/apis/chat-api";
import { RealtimeClient, type RealtimeState } from "@/lib/realtime/client";
import {
  isChatServerMsg,
  type ChatAckEvent,
  type ChatClientMsg,
  type ChatErrorEvent,
  type ChatMessageEvent,
  type ChatReadUpdatedEvent,
  type ChatServerMsg,
  type ChatTypingEvent,
  type ChatWelcomeEvent,
} from "@/lib/realtime/chat-protocol";
import { ApiError } from "@/lib/utils";

const DEFAULT_WS_URL = "ws://localhost:8080";

export const chatKeys = {
  rooms: ["chat", "rooms"] as const,
  messages: (roomId: string) => ["chat", "messages", roomId] as const,
  receipts: (roomId: string) => ["chat", "receipts", roomId] as const,
};

type MutOpts<TData, TInput> = Omit<
  UseMutationOptions<TData, ApiError, TInput>,
  "mutationFn"
>;

type QueryOpts<TData> = Omit<
  UseQueryOptions<TData, ApiError>,
  "queryKey" | "queryFn"
>;

// -------- HTTP hooks --------

export function useChatRooms(options?: QueryOpts<ChatRoom[]>) {
  return useQuery<ChatRoom[], ApiError>({
    queryKey: chatKeys.rooms,
    queryFn: ({ signal }) => chatApi.listRooms(signal),
    ...options,
  });
}

export function useChatMessages(
  roomId: string | null | undefined,
  query: ListMessagesQuery = {},
  options?: QueryOpts<ChatMessage[]>,
) {
  return useQuery<ChatMessage[], ApiError>({
    queryKey: chatKeys.messages(roomId ?? ""),
    queryFn: ({ signal }) =>
      chatApi.listMessages(roomId as string, query, signal),
    enabled: !!roomId,
    ...options,
  });
}

export function useChatReceipts(
  roomId: string | null | undefined,
  options?: QueryOpts<ChatReadReceipt[]>,
) {
  return useQuery<ChatReadReceipt[], ApiError>({
    queryKey: chatKeys.receipts(roomId ?? ""),
    queryFn: ({ signal }) => chatApi.listReadReceipts(roomId as string, signal),
    enabled: !!roomId,
    ...options,
  });
}

export function useEnsureDM(options?: MutOpts<ChatRoom, EnsureDMInput>) {
  const qc = useQueryClient();
  return useMutation<ChatRoom, ApiError, EnsureDMInput>({
    mutationFn: (input) => chatApi.ensureDM(input),
    ...options,
    onSuccess: (room, ...rest) => {
      qc.invalidateQueries({ queryKey: chatKeys.rooms });
      return options?.onSuccess?.(room, ...rest);
    },
  });
}

export function useEnsureTripRoom(options?: MutOpts<ChatRoom, string>) {
  const qc = useQueryClient();
  return useMutation<ChatRoom, ApiError, string>({
    mutationFn: (tripId) => chatApi.ensureTripRoom(tripId),
    ...options,
    onSuccess: (room, ...rest) => {
      qc.invalidateQueries({ queryKey: chatKeys.rooms });
      return options?.onSuccess?.(room, ...rest);
    },
  });
}

export type MarkReadArgs = { roomId: string; input: MarkReadInput };

export function useMarkRead(options?: MutOpts<ChatReadReceipt, MarkReadArgs>) {
  const qc = useQueryClient();
  return useMutation<ChatReadReceipt, ApiError, MarkReadArgs>({
    mutationFn: ({ roomId, input }) => chatApi.markRead(roomId, input),
    ...options,
    onSuccess: (receipt, args, ...rest) => {
      applyReadReceipt(qc, args.roomId, receipt);
      return options?.onSuccess?.(receipt, args, ...rest);
    },
  });
}

export function useHideRoom(options?: MutOpts<null, string>) {
  const qc = useQueryClient();
  return useMutation<null, ApiError, string>({
    mutationFn: (roomId) => chatApi.hideRoom(roomId),
    ...options,
    onSuccess: (data, roomId, ...rest) => {
      // Optimistically drop the room from the cache; server re-list will
      // confirm on invalidation.
      qc.setQueryData<ChatRoom[]>(chatKeys.rooms, (prev) =>
        (prev ?? []).filter((r) => r.id !== roomId),
      );
      qc.invalidateQueries({ queryKey: chatKeys.rooms });
      return options?.onSuccess?.(data, roomId, ...rest);
    },
  });
}

// -------- Realtime hook --------

export type UseChatRealtimeOptions = {
  enabled?: boolean;
  onWelcome?: (msg: ChatWelcomeEvent) => void;
  onMessageEvent?: (msg: ChatMessageEvent) => void;
  onAck?: (msg: ChatAckEvent) => void;
  onServerError?: (msg: ChatErrorEvent) => void;
  onReadUpdated?: (msg: ChatReadUpdatedEvent) => void;
  onTyping?: (msg: ChatTypingEvent) => void;
};

export type UseChatRealtimeResult = {
  state: RealtimeState;
  send: (msg: ChatClientMsg) => boolean;
};

export function useChatRealtime(
  roomId: string | null | undefined,
  options?: UseChatRealtimeOptions,
): UseChatRealtimeResult {
  const qc = useQueryClient();
  const [state, setState] = useState<RealtimeState>("idle");
  const clientRef = useRef<RealtimeClient<ChatClientMsg, ChatServerMsg> | null>(
    null,
  );

  const optsRef = useRef(options);
  useEffect(() => {
    optsRef.current = options;
  }, [options]);

  const enabled = (options?.enabled ?? true) && !!roomId;

  useEffect(() => {
    if (!enabled || !roomId) return;

    const client = new RealtimeClient<ChatClientMsg, ChatServerMsg>({
      wsUrl: buildRoomWsUrl(roomId),
      fetchTicket: async (signal) => {
        const res = await chatApi.issueRoomTicket(roomId, signal);
        return res.ticket;
      },
      parseMessage: (raw) => (isChatServerMsg(raw) ? raw : null),
      onMessage: (msg) => dispatchServerMsg(qc, roomId, msg, optsRef.current),
      onStateChange: setState,
      onError: (err) => {
        if (process.env.NODE_ENV !== "production") {
          console.warn("[chat]", err);
        }
      },
    });

    clientRef.current = client;
    client.connect();

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [enabled, roomId, qc]);

  const send = useCallback((msg: ChatClientMsg): boolean => {
    return clientRef.current?.send(msg) ?? false;
  }, []);

  return useMemo(() => ({ state, send }), [state, send]);
}

// dispatchServerMsg keeps the cache in sync with WS traffic.
function dispatchServerMsg(
  qc: QueryClient,
  roomId: string,
  msg: ChatServerMsg,
  opts: UseChatRealtimeOptions | undefined,
): void {
  switch (msg.type) {
    case "WELCOME":
      opts?.onWelcome?.(msg);
      return;
    case "MESSAGE":
      appendMessage(qc, roomId, msg.data);
      updateRoomLastMessage(qc, roomId, msg.data);
      opts?.onMessageEvent?.(msg);
      return;
    case "ACK":
      if (isChatMessagePayload(msg.data)) {
        appendMessage(qc, roomId, msg.data);
        updateRoomLastMessage(qc, roomId, msg.data);
      } else if (isReadReceiptPayload(msg.data)) {
        applyReadReceipt(qc, roomId, msg.data);
      }
      opts?.onAck?.(msg);
      return;
    case "ERROR":
      opts?.onServerError?.(msg);
      return;
    case "READ_UPDATED":
      applyReadReceipt(qc, roomId, msg.data);
      opts?.onReadUpdated?.(msg);
      return;
    case "TYPING":
      // Ephemeral — nothing to cache.
      opts?.onTyping?.(msg);
      return;
  }
}

function appendMessage(qc: QueryClient, roomId: string, m: ChatMessage): void {
  qc.setQueryData<ChatMessage[]>(chatKeys.messages(roomId), (prev) => {
    const list = prev ?? [];
    if (list.some((x) => x.id === m.id)) return list;
    // Server returns messages DESC (newest first). Insert at head to match.
    return [m, ...list];
  });
}

function updateRoomLastMessage(
  qc: QueryClient,
  roomId: string,
  m: ChatMessage,
): void {
  qc.setQueryData<ChatRoom[]>(chatKeys.rooms, (prev) => {
    if (!prev) return prev;
    return prev.map((r) => (r.id === roomId ? { ...r, last_message: m } : r));
  });
}

function applyReadReceipt(
  qc: QueryClient,
  roomId: string,
  receipt: ChatReadReceipt,
): void {
  qc.setQueryData<ChatReadReceipt[]>(chatKeys.receipts(roomId), (prev) => {
    const list = prev ?? [];
    const others = list.filter((r) => r.user_id !== receipt.user_id);
    return [...others, receipt];
  });
  // The unread counter on rooms depends on receipts; simplest correct thing is
  // to refetch it. Cheap and only fires when someone moves the pointer.
  qc.invalidateQueries({ queryKey: chatKeys.rooms });
}

function isChatMessagePayload(v: unknown): v is ChatMessage {
  return (
    !!v &&
    typeof v === "object" &&
    typeof (v as ChatMessage).id === "string" &&
    typeof (v as ChatMessage).sender_id === "string" &&
    typeof (v as ChatMessage).content === "string"
  );
}

function isReadReceiptPayload(v: unknown): v is ChatReadReceipt {
  return (
    !!v &&
    typeof v === "object" &&
    typeof (v as ChatReadReceipt).last_read_id === "string" &&
    typeof (v as ChatReadReceipt).user_id === "string"
  );
}

function buildRoomWsUrl(roomId: string): string {
  const base = (process.env.NEXT_PUBLIC_WS_URL ?? DEFAULT_WS_URL).replace(
    /\/+$/,
    "",
  );
  return `${base}/ws/chat/rooms/${encodeURIComponent(roomId)}`;
}

export { errorMessage } from "@/lib/utils";
