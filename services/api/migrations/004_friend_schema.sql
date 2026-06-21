-- +goose Up
CREATE SCHEMA IF NOT EXISTS friend;

CREATE TABLE IF NOT EXISTS friend.friends (
  user_id UUID NOT NULL,
  friend_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  PRIMARY KEY (user_id, friend_id),

  CHECK (user_id <> friend_id)
);

CREATE TABLE IF NOT EXISTS friend.invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  sender_id UUID NOT NULL,
  receiver_id UUID NOT NULL,

  status TEXT NOT NULL DEFAULT 'pending',

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CHECK (sender_id <> receiver_id),

  UNIQUE(sender_id, receiver_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_friend_pair
ON friend.invitations (
  LEAST(sender_id, receiver_id),
  GREATEST(sender_id, receiver_id)
);
