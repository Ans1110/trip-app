-- The QR / share-link invite flow was removed; the only invite path is now
-- search-by-name → send friend request, which uses friend.invitations.
DROP TABLE IF EXISTS friend.invitation_tokens;
