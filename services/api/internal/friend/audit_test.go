package friend_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/friend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditWriterMock struct {
	logs []*auth.AuditLog
}

func (a *auditWriterMock) CreateAuditLog(_ context.Context, log *auth.AuditLog) error {
	a.logs = append(a.logs, log)
	return nil
}

func withAudit(w friend.AuditWriter) func(*friend.ServiceConfig) {
	return func(cfg *friend.ServiceConfig) {
		cfg.Audit = w
	}
}

func TestAuditRecordsFriendActions(t *testing.T) {
	t.Run("send_request_writes_audit", func(t *testing.T) {
		sender, receiver := newUser("Alice"), newUser("Bob")
		repo := &repoMock{
			findUserByID: func(_ context.Context, id uuid.UUID) (*auth.User, error) {
				if id == sender.ID {
					return sender, nil
				}
				return receiver, nil
			},
			createOrReviveRequest: func(_ context.Context, r *friend.Request) error {
				if r.ID == uuid.Nil {
					r.ID = uuid.New()
				}
				return nil
			},
		}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		_, err := svc.SendRequest(ctx, sender.ID, receiver.ID, "hi")
		require.NoError(t, err)
		require.Len(t, aw.logs, 1)
		log := aw.logs[0]
		assert.Equal(t, auth.AuditFriendRequestSent, log.Action)
		assert.Equal(t, auth.AuditSuccess, log.Status)
		assert.Equal(t, auth.AuditResourceFriendRequest, log.ResourceType)
		require.NotNil(t, log.ActorUserID)
		assert.Equal(t, sender.ID, *log.ActorUserID)
		require.NotNil(t, log.TargetUserID)
		assert.Equal(t, receiver.ID, *log.TargetUserID)
	})

	t.Run("remove_friend_writes_audit", func(t *testing.T) {
		actor, target := uuid.New(), uuid.New()
		repo := &repoMock{
			areFriends:       func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil },
			deleteFriendship: func(_ context.Context, _, _ uuid.UUID) error { return nil },
		}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		require.NoError(t, svc.RemoveFriend(ctx, actor, target))
		require.Len(t, aw.logs, 1)
		assert.Equal(t, auth.AuditFriendRemoved, aw.logs[0].Action)
		assert.Equal(t, auth.AuditResourceFriendship, aw.logs[0].ResourceType)
		assert.Equal(t, target.String(), aw.logs[0].ResourceID)
	})

	t.Run("accept_request_writes_audit", func(t *testing.T) {
		req := &friend.Request{
			ID:         uuid.New(),
			SenderID:   uuid.New(),
			ReceiverID: uuid.New(),
			Status:     friend.RequestPending,
		}
		repo := &repoMock{
			findRequestByID:     func(_ context.Context, id uuid.UUID) (*friend.Request, error) { return req, nil },
			updateRequestStatus: func(_ context.Context, _ uuid.UUID, _ friend.RequestStatus) error { return nil },
			createFriendship:    func(_ context.Context, _, _ uuid.UUID) error { return nil },
		}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		require.NoError(t, svc.AcceptRequest(ctx, req.ReceiverID, req.ID))
		require.Len(t, aw.logs, 1)
		assert.Equal(t, auth.AuditFriendRequestAccepted, aw.logs[0].Action)
		require.NotNil(t, aw.logs[0].TargetUserID)
		assert.Equal(t, req.SenderID, *aw.logs[0].TargetUserID)
	})

	t.Run("block_user_writes_audit", func(t *testing.T) {
		blocker, target := uuid.New(), newUser("Target")
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return target, nil },
			blockUser:    func(_ context.Context, _, _ uuid.UUID) error { return nil },
		}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		require.NoError(t, svc.BlockUser(ctx, blocker, target.ID))
		require.Len(t, aw.logs, 1)
		assert.Equal(t, auth.AuditFriendBlocked, aw.logs[0].Action)
		assert.Equal(t, auth.AuditResourceFriendBlock, aw.logs[0].ResourceType)
	})

	t.Run("unblock_writes_audit", func(t *testing.T) {
		repo := &repoMock{unblockUser: func(_ context.Context, _, _ uuid.UUID) error { return nil }}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		require.NoError(t, svc.UnblockUser(ctx, uuid.New(), uuid.New()))
		require.Len(t, aw.logs, 1)
		assert.Equal(t, auth.AuditFriendUnblocked, aw.logs[0].Action)
	})

	t.Run("create_invite_writes_audit", func(t *testing.T) {
		repo := &repoMock{createInviteToken: func(_ context.Context, _ *friend.InviteToken) error { return nil }}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		_, err := svc.CreateInvite(ctx, uuid.New(), time.Hour, 3)
		require.NoError(t, err)
		require.Len(t, aw.logs, 1)
		assert.Equal(t, auth.AuditFriendInviteCreated, aw.logs[0].Action)
		require.NotNil(t, aw.logs[0].Detail)
		assert.Equal(t, 3, aw.logs[0].Detail["max_uses"])
	})

	t.Run("revoke_invite_writes_audit", func(t *testing.T) {
		repo := &repoMock{revokeInviteToken: func(_ context.Context, _, _ uuid.UUID) error { return nil }}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		require.NoError(t, svc.RevokeInvite(ctx, uuid.New(), uuid.New()))
		require.Len(t, aw.logs, 1)
		assert.Equal(t, auth.AuditFriendInviteRevoked, aw.logs[0].Action)
	})

	t.Run("accept_invite_writes_audit", func(t *testing.T) {
		inviter := newUser("Inviter")
		tok := &friend.InviteToken{
			ID:        uuid.New(),
			UserID:    inviter.ID,
			MaxUses:   1,
			Uses:      0,
			ExpiresAt: time.Now().Add(time.Hour),
		}
		repo := &repoMock{
			findInviteTokenByHash: func(_ context.Context, _ string) (*friend.InviteToken, error) { return tok, nil },
			findUserByID:          func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return inviter, nil },
			incrementInviteUses:   func(_ context.Context, _ uuid.UUID) error { return nil },
			createFriendship:      func(_ context.Context, _, _ uuid.UUID) error { return nil },
		}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		_, err := svc.AcceptInvite(ctx, uuid.New(), "TOKEN-LONG-ENOUGH")
		require.NoError(t, err)
		require.Len(t, aw.logs, 1)
		assert.Equal(t, auth.AuditFriendInviteAccepted, aw.logs[0].Action)
		require.NotNil(t, aw.logs[0].TargetUserID)
		assert.Equal(t, inviter.ID, *aw.logs[0].TargetUserID)
	})

	t.Run("failed_send_request_does_not_write_audit", func(t *testing.T) {
		// User not found → SendRequest errors before audit point.
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return nil, nil },
		}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		_, err := svc.SendRequest(ctx, uuid.New(), uuid.New(), "hi")
		require.Error(t, err)
		assert.Empty(t, aw.logs)
	})

	t.Run("audit_carries_ip_and_user_agent_from_ctx", func(t *testing.T) {
		repo := &repoMock{
			areFriends:       func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil },
			deleteFriendship: func(_ context.Context, _, _ uuid.UUID) error { return nil },
		}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		c := friend.WithAuditMeta(ctx, "10.1.2.3", "test-agent/1.0")
		require.NoError(t, svc.RemoveFriend(c, uuid.New(), uuid.New()))
		require.Len(t, aw.logs, 1)
		require.NotNil(t, aw.logs[0].IPAddress)
		assert.Equal(t, "10.1.2.3", *aw.logs[0].IPAddress)
		assert.Equal(t, "test-agent/1.0", aw.logs[0].UserAgent)
	})
}
