package friend_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/friend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var ctx = context.Background()

// ---- mock repository ----

type repoMock struct {
	withTx                func(context.Context, func(friend.IRepository) error) error
	findUserByID          func(context.Context, uuid.UUID) (*auth.User, error)
	findUsersByIDs        func(context.Context, []uuid.UUID) (map[uuid.UUID]auth.User, error)
	searchUsers           func(context.Context, uuid.UUID, string, int) ([]auth.User, error)
	areFriends            func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	listFriends           func(context.Context, uuid.UUID) ([]friend.Friendship, error)
	createFriendship      func(context.Context, uuid.UUID, uuid.UUID) error
	deleteFriendship      func(context.Context, uuid.UUID, uuid.UUID) error
	findRequestByID       func(context.Context, uuid.UUID) (*friend.Request, error)
	findPendingRequest    func(context.Context, uuid.UUID, uuid.UUID) (*friend.Request, error)
	createOrReviveRequest func(context.Context, *friend.Request) error
	updateRequestStatus   func(context.Context, uuid.UUID, friend.RequestStatus) error
	listIncomingRequests  func(context.Context, uuid.UUID) ([]friend.Request, error)
	listOutgoingRequests  func(context.Context, uuid.UUID) ([]friend.Request, error)
	isBlocked             func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	blockUser             func(context.Context, uuid.UUID, uuid.UUID) error
	unblockUser           func(context.Context, uuid.UUID, uuid.UUID) error
	listBlocks            func(context.Context, uuid.UUID) ([]friend.Block, error)
}

func (m *repoMock) WithTx(c context.Context, fn func(friend.IRepository) error) error {
	if m.withTx != nil {
		return m.withTx(c, fn)
	}
	return fn(m)
}

func (m *repoMock) FindUserByID(c context.Context, id uuid.UUID) (*auth.User, error) {
	if m.findUserByID != nil {
		return m.findUserByID(c, id)
	}
	return nil, nil
}

func (m *repoMock) FindUsersByIDs(c context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	if m.findUsersByIDs != nil {
		return m.findUsersByIDs(c, ids)
	}
	return map[uuid.UUID]auth.User{}, nil
}

func (m *repoMock) SearchUsers(c context.Context, requesterID uuid.UUID, q string, lim int) ([]auth.User, error) {
	if m.searchUsers != nil {
		return m.searchUsers(c, requesterID, q, lim)
	}
	return nil, nil
}

func (m *repoMock) AreFriends(c context.Context, a, b uuid.UUID) (bool, error) {
	if m.areFriends != nil {
		return m.areFriends(c, a, b)
	}
	return false, nil
}

func (m *repoMock) ListFriends(c context.Context, id uuid.UUID) ([]friend.Friendship, error) {
	if m.listFriends != nil {
		return m.listFriends(c, id)
	}
	return nil, nil
}

func (m *repoMock) CreateFriendship(c context.Context, a, b uuid.UUID) error {
	if m.createFriendship != nil {
		return m.createFriendship(c, a, b)
	}
	return nil
}

func (m *repoMock) DeleteFriendship(c context.Context, a, b uuid.UUID) error {
	if m.deleteFriendship != nil {
		return m.deleteFriendship(c, a, b)
	}
	return nil
}

func (m *repoMock) FindRequestByID(c context.Context, id uuid.UUID) (*friend.Request, error) {
	if m.findRequestByID != nil {
		return m.findRequestByID(c, id)
	}
	return nil, nil
}

func (m *repoMock) FindPendingRequest(c context.Context, sender, receiver uuid.UUID) (*friend.Request, error) {
	if m.findPendingRequest != nil {
		return m.findPendingRequest(c, sender, receiver)
	}
	return nil, nil
}

func (m *repoMock) CreateOrReviveRequest(c context.Context, r *friend.Request) error {
	if m.createOrReviveRequest != nil {
		return m.createOrReviveRequest(c, r)
	}
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	r.Status = friend.RequestPending
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (m *repoMock) UpdateRequestStatus(c context.Context, id uuid.UUID, s friend.RequestStatus) error {
	if m.updateRequestStatus != nil {
		return m.updateRequestStatus(c, id, s)
	}
	return nil
}

func (m *repoMock) ListIncomingRequests(c context.Context, id uuid.UUID) ([]friend.Request, error) {
	if m.listIncomingRequests != nil {
		return m.listIncomingRequests(c, id)
	}
	return nil, nil
}

func (m *repoMock) ListOutgoingRequests(c context.Context, id uuid.UUID) ([]friend.Request, error) {
	if m.listOutgoingRequests != nil {
		return m.listOutgoingRequests(c, id)
	}
	return nil, nil
}

func (m *repoMock) IsBlocked(c context.Context, a, b uuid.UUID) (bool, error) {
	if m.isBlocked != nil {
		return m.isBlocked(c, a, b)
	}
	return false, nil
}

func (m *repoMock) BlockUser(c context.Context, a, b uuid.UUID) error {
	if m.blockUser != nil {
		return m.blockUser(c, a, b)
	}
	return nil
}

func (m *repoMock) UnblockUser(c context.Context, a, b uuid.UUID) error {
	if m.unblockUser != nil {
		return m.unblockUser(c, a, b)
	}
	return nil
}

func (m *repoMock) ListBlocks(c context.Context, id uuid.UUID) ([]friend.Block, error) {
	if m.listBlocks != nil {
		return m.listBlocks(c, id)
	}
	return nil, nil
}

// ---- helpers ----

func newSvc(repo friend.IRepository, opts ...func(*friend.ServiceConfig)) friend.IService {
	cfg := friend.ServiceConfig{
		Repo:   repo,
		Logger: zap.NewNop(),
	}
	for _, o := range opts {
		o(&cfg)
	}
	return friend.NewService(cfg)
}

func newUser(name string) *auth.User {
	return &auth.User{
		ID:        uuid.New(),
		Email:     strings.ToLower(name) + "@example.com",
		Name:      name,
		Status:    auth.UserStatusActive,
		CreatedAt: time.Now(),
	}
}

// ---- ListFriends ----

func TestListFriends(t *testing.T) {
	t.Run("hydrates_friend_users", func(t *testing.T) {
		me := uuid.New()
		alice := newUser("Alice")
		bob := newUser("Bob")
		repo := &repoMock{
			listFriends: func(_ context.Context, uid uuid.UUID) ([]friend.Friendship, error) {
				assert.Equal(t, me, uid)
				return []friend.Friendship{
					{UserID: me, FriendID: alice.ID, CreatedAt: time.Now()},
					{UserID: me, FriendID: bob.ID, CreatedAt: time.Now()},
				}, nil
			},
			findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
				assert.ElementsMatch(t, []uuid.UUID{alice.ID, bob.ID}, ids)
				return map[uuid.UUID]auth.User{alice.ID: *alice, bob.ID: *bob}, nil
			},
		}
		out, err := newSvc(repo).ListFriends(ctx, me)
		require.NoError(t, err)
		assert.Len(t, out, 2)
		// order preserved from listFriends
		assert.Equal(t, alice.ID.String(), out[0].User.ID)
		assert.Equal(t, bob.ID.String(), out[1].User.ID)
	})

	t.Run("empty", func(t *testing.T) {
		out, err := newSvc(&repoMock{}).ListFriends(ctx, uuid.New())
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("repo_error_propagates", func(t *testing.T) {
		boom := errors.New("db down")
		repo := &repoMock{
			listFriends: func(context.Context, uuid.UUID) ([]friend.Friendship, error) { return nil, boom },
		}
		_, err := newSvc(repo).ListFriends(ctx, uuid.New())
		require.ErrorIs(t, err, boom)
	})
}

// ---- RemoveFriend ----

func TestRemoveFriend(t *testing.T) {
	t.Run("self_rejected", func(t *testing.T) {
		uid := uuid.New()
		err := newSvc(&repoMock{}).RemoveFriend(ctx, uid, uid)
		require.ErrorIs(t, err, friend.ErrCannotTargetSelf)
	})

	t.Run("delegates_to_repo", func(t *testing.T) {
		called := false
		me, them := uuid.New(), uuid.New()
		repo := &repoMock{
			areFriends: func(_ context.Context, a, b uuid.UUID) (bool, error) {
				assert.Equal(t, me, a)
				assert.Equal(t, them, b)
				return true, nil
			},
			deleteFriendship: func(_ context.Context, a, b uuid.UUID) error {
				called = true
				assert.Equal(t, me, a)
				assert.Equal(t, them, b)
				return nil
			},
		}
		require.NoError(t, newSvc(repo).RemoveFriend(ctx, me, them))
		assert.True(t, called)
	})

	t.Run("not_friends_rejected", func(t *testing.T) {
		deleted := false
		repo := &repoMock{
			areFriends:       func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil },
			deleteFriendship: func(context.Context, uuid.UUID, uuid.UUID) error { deleted = true; return nil },
		}
		err := newSvc(repo).RemoveFriend(ctx, uuid.New(), uuid.New())
		require.ErrorIs(t, err, friend.ErrNotFriends)
		assert.False(t, deleted)
	})
}

// ---- SearchUsers ----

func TestSearchUsers(t *testing.T) {
	t.Run("maps_to_summary", func(t *testing.T) {
		me := uuid.New()
		alice := newUser("Alice")
		repo := &repoMock{
			searchUsers: func(_ context.Context, requesterID uuid.UUID, q string, lim int) ([]auth.User, error) {
				assert.Equal(t, me, requesterID)
				assert.Equal(t, "ali", q)
				assert.Equal(t, 5, lim)
				return []auth.User{*alice}, nil
			},
		}
		out, err := newSvc(repo).SearchUsers(ctx, me, "ali", 5)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, alice.ID.String(), out[0].ID)
		assert.Equal(t, alice.Email, out[0].Email)
	})
}

// ---- SendRequest ----

func TestSendRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		sender := newUser("Sender")
		receiver := newUser("Receiver")
		created := false
		repo := &repoMock{
			findUserByID: func(_ context.Context, id uuid.UUID) (*auth.User, error) {
				switch id {
				case sender.ID:
					return sender, nil
				case receiver.ID:
					return receiver, nil
				}
				return nil, nil
			},
			createOrReviveRequest: func(_ context.Context, r *friend.Request) error {
				created = true
				assert.Equal(t, sender.ID, r.SenderID)
				assert.Equal(t, receiver.ID, r.ReceiverID)
				assert.Equal(t, "hello", r.Message)
				r.ID = uuid.New()
				r.Status = friend.RequestPending
				return nil
			},
		}
		out, err := newSvc(repo).SendRequest(ctx, sender.ID, receiver.ID, "hello")
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.True(t, created)
		assert.Equal(t, string(friend.RequestPending), out.Status)
		assert.Equal(t, sender.ID.String(), out.Sender.ID)
		assert.Equal(t, receiver.ID.String(), out.Receiver.ID)
	})

	t.Run("self_rejected", func(t *testing.T) {
		me := uuid.New()
		_, err := newSvc(&repoMock{}).SendRequest(ctx, me, me, "hi")
		require.ErrorIs(t, err, friend.ErrCannotTargetSelf)
	})

	t.Run("receiver_not_found", func(t *testing.T) {
		repo := &repoMock{
			findUserByID: func(context.Context, uuid.UUID) (*auth.User, error) { return nil, nil },
		}
		_, err := newSvc(repo).SendRequest(ctx, uuid.New(), uuid.New(), "")
		require.ErrorIs(t, err, friend.ErrUserNotFound)
	})

	t.Run("receiver_inactive", func(t *testing.T) {
		receiver := newUser("R")
		receiver.Status = auth.UserStatusDeactivated
		repo := &repoMock{
			findUserByID: func(context.Context, uuid.UUID) (*auth.User, error) { return receiver, nil },
		}
		_, err := newSvc(repo).SendRequest(ctx, uuid.New(), receiver.ID, "")
		require.ErrorIs(t, err, friend.ErrUserNotFound)
	})

	t.Run("either_side_blocked", func(t *testing.T) {
		receiver := newUser("R")
		repo := &repoMock{
			findUserByID: func(context.Context, uuid.UUID) (*auth.User, error) { return receiver, nil },
			isBlocked: func(_ context.Context, blocker, blocked uuid.UUID) (bool, error) {
				// receiver has blocked sender
				return blocker == receiver.ID, nil
			},
		}
		_, err := newSvc(repo).SendRequest(ctx, uuid.New(), receiver.ID, "")
		require.ErrorIs(t, err, friend.ErrBlocked)
	})

	t.Run("already_friends", func(t *testing.T) {
		receiver := newUser("R")
		repo := &repoMock{
			findUserByID: func(context.Context, uuid.UUID) (*auth.User, error) { return receiver, nil },
			areFriends:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		}
		_, err := newSvc(repo).SendRequest(ctx, uuid.New(), receiver.ID, "")
		require.ErrorIs(t, err, friend.ErrAlreadyFriends)
	})

	t.Run("pending_already_exists", func(t *testing.T) {
		receiver := newUser("R")
		repo := &repoMock{
			findUserByID: func(context.Context, uuid.UUID) (*auth.User, error) { return receiver, nil },
			findPendingRequest: func(context.Context, uuid.UUID, uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: uuid.New(), Status: friend.RequestPending}, nil
			},
		}
		_, err := newSvc(repo).SendRequest(ctx, uuid.New(), receiver.ID, "")
		require.ErrorIs(t, err, friend.ErrRequestExists)
	})
}

// ---- AcceptRequest / DeclineRequest / CancelRequest ----

func TestAcceptRequest(t *testing.T) {
	receiverID := uuid.New()
	senderID := uuid.New()
	reqID := uuid.New()

	t.Run("success_runs_in_transaction", func(t *testing.T) {
		updated := false
		linked := false
		repo := &repoMock{
			findRequestByID: func(_ context.Context, id uuid.UUID) (*friend.Request, error) {
				assert.Equal(t, reqID, id)
				return &friend.Request{ID: reqID, SenderID: senderID, ReceiverID: receiverID, Status: friend.RequestPending}, nil
			},
			updateRequestStatus: func(_ context.Context, id uuid.UUID, s friend.RequestStatus) error {
				updated = true
				assert.Equal(t, reqID, id)
				assert.Equal(t, friend.RequestAccepted, s)
				return nil
			},
			createFriendship: func(_ context.Context, a, b uuid.UUID) error {
				linked = true
				assert.Equal(t, senderID, a)
				assert.Equal(t, receiverID, b)
				return nil
			},
		}
		require.NoError(t, newSvc(repo).AcceptRequest(ctx, receiverID, reqID))
		assert.True(t, updated)
		assert.True(t, linked)
	})

	t.Run("not_receiver_is_forbidden", func(t *testing.T) {
		repo := &repoMock{
			findRequestByID: func(context.Context, uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: reqID, SenderID: senderID, ReceiverID: uuid.New(), Status: friend.RequestPending}, nil
			},
		}
		err := newSvc(repo).AcceptRequest(ctx, receiverID, reqID)
		require.ErrorIs(t, err, friend.ErrForbidden)
	})

	t.Run("missing_request", func(t *testing.T) {
		repo := &repoMock{
			findRequestByID: func(context.Context, uuid.UUID) (*friend.Request, error) { return nil, nil },
		}
		err := newSvc(repo).AcceptRequest(ctx, receiverID, reqID)
		require.ErrorIs(t, err, friend.ErrRequestNotFound)
	})

	t.Run("non_pending_request", func(t *testing.T) {
		repo := &repoMock{
			findRequestByID: func(context.Context, uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: reqID, ReceiverID: receiverID, Status: friend.RequestAccepted}, nil
			},
		}
		err := newSvc(repo).AcceptRequest(ctx, receiverID, reqID)
		require.ErrorIs(t, err, friend.ErrRequestNotFound)
	})
}

func TestDeclineAndCancel(t *testing.T) {
	receiverID, senderID, reqID := uuid.New(), uuid.New(), uuid.New()

	t.Run("decline_by_receiver_marks_declined", func(t *testing.T) {
		var got friend.RequestStatus
		repo := &repoMock{
			findRequestByID: func(context.Context, uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: reqID, SenderID: senderID, ReceiverID: receiverID, Status: friend.RequestPending}, nil
			},
			updateRequestStatus: func(_ context.Context, _ uuid.UUID, s friend.RequestStatus) error {
				got = s
				return nil
			},
		}
		require.NoError(t, newSvc(repo).DeclineRequest(ctx, receiverID, reqID))
		assert.Equal(t, friend.RequestDeclined, got)
	})

	t.Run("cancel_by_sender_marks_cancelled", func(t *testing.T) {
		var got friend.RequestStatus
		repo := &repoMock{
			findRequestByID: func(context.Context, uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: reqID, SenderID: senderID, ReceiverID: receiverID, Status: friend.RequestPending}, nil
			},
			updateRequestStatus: func(_ context.Context, _ uuid.UUID, s friend.RequestStatus) error {
				got = s
				return nil
			},
		}
		require.NoError(t, newSvc(repo).CancelRequest(ctx, senderID, reqID))
		assert.Equal(t, friend.RequestCancelled, got)
	})

	t.Run("decline_by_non_receiver_forbidden", func(t *testing.T) {
		repo := &repoMock{
			findRequestByID: func(context.Context, uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: reqID, SenderID: senderID, ReceiverID: receiverID, Status: friend.RequestPending}, nil
			},
		}
		require.ErrorIs(t, newSvc(repo).DeclineRequest(ctx, senderID, reqID), friend.ErrForbidden)
	})

	t.Run("cancel_by_non_sender_forbidden", func(t *testing.T) {
		repo := &repoMock{
			findRequestByID: func(context.Context, uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: reqID, SenderID: senderID, ReceiverID: receiverID, Status: friend.RequestPending}, nil
			},
		}
		require.ErrorIs(t, newSvc(repo).CancelRequest(ctx, receiverID, reqID), friend.ErrForbidden)
	})
}

// ---- ListIncoming/Outgoing ----

func TestListRequests(t *testing.T) {
	me := uuid.New()
	other := newUser("Other")
	myself := newUser("Me")
	myself.ID = me

	t.Run("incoming_hydrates_users", func(t *testing.T) {
		reqID := uuid.New()
		repo := &repoMock{
			listIncomingRequests: func(_ context.Context, uid uuid.UUID) ([]friend.Request, error) {
				assert.Equal(t, me, uid)
				return []friend.Request{{ID: reqID, SenderID: other.ID, ReceiverID: me, Status: friend.RequestPending}}, nil
			},
			findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
				assert.ElementsMatch(t, []uuid.UUID{other.ID, me}, ids)
				return map[uuid.UUID]auth.User{other.ID: *other, me: *myself}, nil
			},
		}
		out, err := newSvc(repo).ListIncomingRequests(ctx, me)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, other.ID.String(), out[0].Sender.ID)
		assert.Equal(t, me.String(), out[0].Receiver.ID)
	})

	t.Run("empty_returns_non_nil_slice", func(t *testing.T) {
		out, err := newSvc(&repoMock{}).ListOutgoingRequests(ctx, me)
		require.NoError(t, err)
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})
}

// ---- BlockUser / UnblockUser / ListBlocks ----

func TestBlockUser(t *testing.T) {
	blocker := uuid.New()
	target := newUser("T")

	t.Run("self_rejected", func(t *testing.T) {
		require.ErrorIs(t, newSvc(&repoMock{}).BlockUser(ctx, blocker, blocker), friend.ErrCannotTargetSelf)
	})

	t.Run("target_missing", func(t *testing.T) {
		repo := &repoMock{
			findUserByID: func(context.Context, uuid.UUID) (*auth.User, error) { return nil, nil },
		}
		require.ErrorIs(t, newSvc(repo).BlockUser(ctx, blocker, uuid.New()), friend.ErrUserNotFound)
	})

	t.Run("transactional_cleanup_of_friendship_and_pending", func(t *testing.T) {
		blocked := false
		fsDeleted := false
		var statusUpdates []friend.RequestStatus
		repo := &repoMock{
			findUserByID: func(context.Context, uuid.UUID) (*auth.User, error) { return target, nil },
			blockUser: func(_ context.Context, b, bd uuid.UUID) error {
				blocked = true
				assert.Equal(t, blocker, b)
				assert.Equal(t, target.ID, bd)
				return nil
			},
			deleteFriendship: func(_ context.Context, a, b uuid.UUID) error {
				fsDeleted = true
				return nil
			},
			findPendingRequest: func(_ context.Context, sender, receiver uuid.UUID) (*friend.Request, error) {
				return &friend.Request{ID: uuid.New(), SenderID: sender, ReceiverID: receiver, Status: friend.RequestPending}, nil
			},
			updateRequestStatus: func(_ context.Context, _ uuid.UUID, s friend.RequestStatus) error {
				statusUpdates = append(statusUpdates, s)
				return nil
			},
		}
		require.NoError(t, newSvc(repo).BlockUser(ctx, blocker, target.ID))
		assert.True(t, blocked)
		assert.True(t, fsDeleted)
		assert.ElementsMatch(t, []friend.RequestStatus{friend.RequestCancelled, friend.RequestDeclined}, statusUpdates)
	})
}

func TestListBlocks(t *testing.T) {
	me := uuid.New()
	blocked := newUser("Blocked")
	repo := &repoMock{
		listBlocks: func(_ context.Context, uid uuid.UUID) ([]friend.Block, error) {
			assert.Equal(t, me, uid)
			return []friend.Block{{BlockerID: me, BlockedID: blocked.ID, CreatedAt: time.Now()}}, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{blocked.ID: *blocked}, nil
		},
	}
	out, err := newSvc(repo).ListBlocks(ctx, me)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, blocked.ID.String(), out[0].User.ID)
}

func TestUnblockUserDelegates(t *testing.T) {
	called := false
	repo := &repoMock{
		unblockUser: func(context.Context, uuid.UUID, uuid.UUID) error {
			called = true
			return nil
		},
	}
	require.NoError(t, newSvc(repo).UnblockUser(ctx, uuid.New(), uuid.New()))
	assert.True(t, called)
}

