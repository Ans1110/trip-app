-- Invite tokens removed: joining is now solely via room code (shared by QR or
-- raw code entry). The table and its index are no longer referenced from the
-- service layer.

DROP INDEX IF EXISTS trip.idx_room_invite_tokens_room_id;
DROP TABLE IF EXISTS trip.room_invite_tokens;
