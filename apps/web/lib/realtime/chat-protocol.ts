import type {
  ChatMessage,
  ChatMessageType,
  ChatReadReceipt,
} from "../apis/chat-api";

export type ChatClientOp = "SEND_MESSAGE" | "MARK_READ" | "TYPING";

export type ChatErrorCode =
  | "BAD_OP"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "INTERNAL"
  | "UNSUPPORTED_OP"
  | "OVERLOADED";

// ---- Client -> Server ----

export type SendMessageData = {
  content: string;
  type?: ChatMessageType;
  reply_to?: string;
  client_msg_id?: string;
  media_id?: string;
  media_filename?: string;
};

export type MarkReadData = { last_read_id: string };
export type TypingData = { is_typing: boolean };

type BaseClientChatMsg = {
  msg_id?: string;
  room_id: string;
};

export type SendMessageMsg = BaseClientChatMsg & {
  type: "SEND_MESSAGE";
  data: SendMessageData;
};

export type MarkReadMsg = BaseClientChatMsg & {
  type: "MARK_READ";
  data: MarkReadData;
};

export type TypingMsg = BaseClientChatMsg & {
  type: "TYPING";
  data: TypingData;
};

export type ChatClientMsg = SendMessageMsg | MarkReadMsg | TypingMsg;

// ---- Server -> Client ----

export type ChatWelcomeEvent = {
  type: "WELCOME";
  room_id: string;
};

export type ChatMessageEvent = {
  type: "MESSAGE";
  room_id: string;
  data: ChatMessage;
};

// ACK payloads vary: SEND_MESSAGE echoes the persisted ChatMessage; MARK_READ
// echoes the ChatReadReceipt. Consumers should key off `ref` (msg_id) to route.
export type ChatAckEvent = {
  type: "ACK";
  ref?: string;
  room_id?: string;
  data?: ChatMessage | ChatReadReceipt | { deduped?: boolean };
};

export type ChatErrorEvent = {
  type: "ERROR";
  ref?: string;
  code: ChatErrorCode;
  message: string;
};

export type ChatReadUpdatedEvent = {
  type: "READ_UPDATED";
  room_id: string;
  data: ChatReadReceipt;
};

export type ChatTypingEvent = {
  type: "TYPING";
  room_id: string;
  data: { user_id: string; is_typing: boolean };
};

export type ChatServerMsg =
  | ChatWelcomeEvent
  | ChatMessageEvent
  | ChatAckEvent
  | ChatErrorEvent
  | ChatReadUpdatedEvent
  | ChatTypingEvent;

export function isChatServerMsg(v: unknown): v is ChatServerMsg {
  return (
    typeof v === "object" &&
    v !== null &&
    typeof (v as { type?: unknown }).type === "string"
  );
}
