"use client";

import { create } from "zustand";

// Shared state for the floating chat window. External triggers (e.g. the
// friend context menu's "Chat" item) call `openWithPeer(peerId)`; the window
// then resolves that to a DM room and selects it.
type ChatWindowState = {
  open: boolean;
  selectedRoomId: string | null;
  // Set by callers that want to open a DM with a specific friend before the
  // DM room id is known. The window consumes it, resolves via ensureDM, and
  // clears it via `clearPendingPeer`.
  pendingPeerId: string | null;

  openWindow: () => void;
  close: () => void;
  toggle: () => void;

  selectRoom: (roomId: string | null) => void;
  openWithPeer: (peerId: string) => void;
  clearPendingPeer: () => void;
};

export const useChatWindowStore = create<ChatWindowState>((set) => ({
  open: false,
  selectedRoomId: null,
  pendingPeerId: null,

  openWindow: () => set({ open: true }),
  close: () => set({ open: false }),
  toggle: () => set((s) => ({ open: !s.open })),

  selectRoom: (roomId) => set({ selectedRoomId: roomId }),
  openWithPeer: (peerId) =>
    set({ open: true, pendingPeerId: peerId, selectedRoomId: null }),
  clearPendingPeer: () => set({ pendingPeerId: null }),
}));
