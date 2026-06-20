package friend_test

import (
	"context"
	"testing"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/friend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditWriterMock struct {
	logs []*audit.Log
}

func (a *auditWriterMock) Create(_ context.Context, log *audit.Log) error {
	a.logs = append(a.logs, log)
	return nil
}

func withAudit(w audit.Writer) func(*friend.ServiceConfig) {
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
		assert.Equal(t, friend.AuditFriendRequestSent, log.Action)
		assert.Equal(t, audit.Success, log.Status)
		assert.Equal(t, friend.AuditResourceFriendRequest, log.ResourceType)
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
		assert.Equal(t, friend.AuditFriendRemoved, aw.logs[0].Action)
		assert.Equal(t, friend.AuditResourceFriendship, aw.logs[0].ResourceType)
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
		assert.Equal(t, friend.AuditFriendRequestAccepted, aw.logs[0].Action)
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
		assert.Equal(t, friend.AuditFriendBlocked, aw.logs[0].Action)
		assert.Equal(t, friend.AuditResourceFriendBlock, aw.logs[0].ResourceType)
	})

	t.Run("unblock_writes_audit", func(t *testing.T) {
		repo := &repoMock{unblockUser: func(_ context.Context, _, _ uuid.UUID) error { return nil }}
		aw := &auditWriterMock{}
		svc := newSvc(repo, withAudit(aw))

		require.NoError(t, svc.UnblockUser(ctx, uuid.New(), uuid.New()))
		require.Len(t, aw.logs, 1)
		assert.Equal(t, friend.AuditFriendUnblocked, aw.logs[0].Action)
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
