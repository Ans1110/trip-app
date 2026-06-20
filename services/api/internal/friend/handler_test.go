package friend_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/friend"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- mock service implementing friend.IService ----

type svcMock struct {
	listFriends          func(context.Context, uuid.UUID) ([]friend.FriendResponse, error)
	removeFriend         func(context.Context, uuid.UUID, uuid.UUID) error
	searchUsers          func(context.Context, uuid.UUID, string, int) ([]friend.UserSummary, error)
	sendRequest          func(context.Context, uuid.UUID, uuid.UUID, string) (*friend.RequestResponse, error)
	acceptRequest        func(context.Context, uuid.UUID, uuid.UUID) error
	declineRequest       func(context.Context, uuid.UUID, uuid.UUID) error
	cancelRequest        func(context.Context, uuid.UUID, uuid.UUID) error
	listIncomingRequests func(context.Context, uuid.UUID) ([]friend.RequestResponse, error)
	listOutgoingRequests func(context.Context, uuid.UUID) ([]friend.RequestResponse, error)
	blockUser            func(context.Context, uuid.UUID, uuid.UUID) error
	unblockUser          func(context.Context, uuid.UUID, uuid.UUID) error
	listBlocks           func(context.Context, uuid.UUID) ([]friend.BlockResponse, error)
}

func (m *svcMock) ListFriends(c context.Context, uid uuid.UUID) ([]friend.FriendResponse, error) {
	if m.listFriends != nil {
		return m.listFriends(c, uid)
	}
	return []friend.FriendResponse{}, nil
}

func (m *svcMock) RemoveFriend(c context.Context, uid, fid uuid.UUID) error {
	if m.removeFriend != nil {
		return m.removeFriend(c, uid, fid)
	}
	return nil
}

func (m *svcMock) SearchUsers(c context.Context, uid uuid.UUID, q string, lim int) ([]friend.UserSummary, error) {
	if m.searchUsers != nil {
		return m.searchUsers(c, uid, q, lim)
	}
	return []friend.UserSummary{}, nil
}

func (m *svcMock) SendRequest(c context.Context, s, r uuid.UUID, msg string) (*friend.RequestResponse, error) {
	if m.sendRequest != nil {
		return m.sendRequest(c, s, r, msg)
	}
	return &friend.RequestResponse{ID: uuid.NewString(), Status: "pending"}, nil
}

func (m *svcMock) AcceptRequest(c context.Context, uid, rid uuid.UUID) error {
	if m.acceptRequest != nil {
		return m.acceptRequest(c, uid, rid)
	}
	return nil
}

func (m *svcMock) DeclineRequest(c context.Context, uid, rid uuid.UUID) error {
	if m.declineRequest != nil {
		return m.declineRequest(c, uid, rid)
	}
	return nil
}

func (m *svcMock) CancelRequest(c context.Context, uid, rid uuid.UUID) error {
	if m.cancelRequest != nil {
		return m.cancelRequest(c, uid, rid)
	}
	return nil
}

func (m *svcMock) ListIncomingRequests(c context.Context, uid uuid.UUID) ([]friend.RequestResponse, error) {
	if m.listIncomingRequests != nil {
		return m.listIncomingRequests(c, uid)
	}
	return []friend.RequestResponse{}, nil
}

func (m *svcMock) ListOutgoingRequests(c context.Context, uid uuid.UUID) ([]friend.RequestResponse, error) {
	if m.listOutgoingRequests != nil {
		return m.listOutgoingRequests(c, uid)
	}
	return []friend.RequestResponse{}, nil
}

func (m *svcMock) BlockUser(c context.Context, b, t uuid.UUID) error {
	if m.blockUser != nil {
		return m.blockUser(c, b, t)
	}
	return nil
}

func (m *svcMock) UnblockUser(c context.Context, b, t uuid.UUID) error {
	if m.unblockUser != nil {
		return m.unblockUser(c, b, t)
	}
	return nil
}

func (m *svcMock) ListBlocks(c context.Context, uid uuid.UUID) ([]friend.BlockResponse, error) {
	if m.listBlocks != nil {
		return m.listBlocks(c, uid)
	}
	return []friend.BlockResponse{}, nil
}

// ---- helpers ----

// newRouter mounts the friend handler at /api/v1, optionally injecting a userID
// onto the gin context to simulate JWTAuth.
func newRouter(svc friend.IService, userID *uuid.UUID) *gin.Engine {
	r := gin.New()
	pri := r.Group("/api/v1")
	if userID != nil {
		uid := *userID
		pri.Use(func(c *gin.Context) {
			c.Set(middleware.ContextUserID, uid)
			c.Next()
		})
	}
	friend.NewHandler(svc, zap.NewNop()).RegisterRoutes(pri)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		buf = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func parseEnvelope(t *testing.T, w *httptest.ResponseRecorder) apiResp {
	t.Helper()
	var resp apiResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func ptr(u uuid.UUID) *uuid.UUID { return &u }

// ---- auth gating ----

func TestRoutesRequireAuth(t *testing.T) {
	// No userID middleware → every protected route should 401.
	r := newRouter(&svcMock{}, nil)
	cases := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/v1/friends", nil},
		{http.MethodGet, "/api/v1/friends/search?q=x", nil},
		{http.MethodDelete, "/api/v1/friends/" + uuid.NewString(), nil},
		{http.MethodGet, "/api/v1/friends/requests", nil},
		{http.MethodGet, "/api/v1/friends/requests/outgoing", nil},
		{http.MethodPost, "/api/v1/friends/requests", friend.SendRequestPayload{ReceiverID: uuid.NewString()}},
		{http.MethodPost, "/api/v1/friends/requests/" + uuid.NewString() + "/accept", nil},
		{http.MethodGet, "/api/v1/friends/blocks", nil},
		{http.MethodPost, "/api/v1/friends/blocks", friend.BlockPayload{TargetID: uuid.NewString()}},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := doJSON(t, r, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// ---- friends list / remove / search ----

func TestHandlerListFriends(t *testing.T) {
	uid := uuid.New()
	t.Run("success", func(t *testing.T) {
		svc := &svcMock{listFriends: func(_ context.Context, got uuid.UUID) ([]friend.FriendResponse, error) {
			assert.Equal(t, uid, got)
			return []friend.FriendResponse{{User: friend.UserSummary{ID: uuid.NewString(), Name: "A"}, CreatedAt: time.Now()}}, nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodGet, "/api/v1/friends", nil)
		require.Equal(t, http.StatusOK, w.Code)
		env := parseEnvelope(t, w)
		var fs []friend.FriendResponse
		require.NoError(t, json.Unmarshal(env.Data, &fs))
		require.Len(t, fs, 1)
		assert.Equal(t, "A", fs[0].User.Name)
	})

	t.Run("internal_error", func(t *testing.T) {
		svc := &svcMock{listFriends: func(context.Context, uuid.UUID) ([]friend.FriendResponse, error) {
			return nil, assert.AnError
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodGet, "/api/v1/friends", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandlerRemoveFriend(t *testing.T) {
	uid := uuid.New()
	friendID := uuid.New()

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &svcMock{removeFriend: func(_ context.Context, a, b uuid.UUID) error {
			called = true
			assert.Equal(t, uid, a)
			assert.Equal(t, friendID, b)
			return nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodDelete, "/api/v1/friends/"+friendID.String(), nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("invalid_uuid_returns_400", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodDelete, "/api/v1/friends/not-a-uuid", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("self_target_returns_400", func(t *testing.T) {
		svc := &svcMock{removeFriend: func(context.Context, uuid.UUID, uuid.UUID) error {
			return friend.ErrCannotTargetSelf
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodDelete, "/api/v1/friends/"+friendID.String(), nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not_friends_returns_404", func(t *testing.T) {
		svc := &svcMock{removeFriend: func(context.Context, uuid.UUID, uuid.UUID) error {
			return friend.ErrNotFriends
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodDelete, "/api/v1/friends/"+friendID.String(), nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandlerSearch(t *testing.T) {
	uid := uuid.New()

	t.Run("success_passes_query_and_limit", func(t *testing.T) {
		svc := &svcMock{searchUsers: func(_ context.Context, _ uuid.UUID, q string, lim int) ([]friend.UserSummary, error) {
			assert.Equal(t, "ali", q)
			assert.Equal(t, 5, lim)
			return []friend.UserSummary{{ID: uuid.NewString(), Name: "Alice"}}, nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodGet, "/api/v1/friends/search?q=ali&limit=5", nil)
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing_q_returns_400", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodGet, "/api/v1/friends/search", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ---- requests ----

func TestHandlerSendRequest(t *testing.T) {
	uid := uuid.New()
	receiverID := uuid.New()

	t.Run("success_returns_201", func(t *testing.T) {
		called := false
		svc := &svcMock{sendRequest: func(_ context.Context, s, r uuid.UUID, msg string) (*friend.RequestResponse, error) {
			called = true
			assert.Equal(t, uid, s)
			assert.Equal(t, receiverID, r)
			assert.Equal(t, "hi", msg)
			return &friend.RequestResponse{ID: uuid.NewString(), Status: "pending"}, nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/friends/requests",
			friend.SendRequestPayload{ReceiverID: receiverID.String(), Message: "hi"})
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.True(t, called)
	})

	t.Run("invalid_payload_returns_400", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodPost, "/api/v1/friends/requests",
			friend.SendRequestPayload{ReceiverID: "not-a-uuid"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error_mapping", func(t *testing.T) {
		cases := map[string]struct {
			err  error
			code int
		}{
			"already_friends":   {friend.ErrAlreadyFriends, http.StatusConflict},
			"pending":           {friend.ErrRequestExists, http.StatusConflict},
			"receiver_missing":  {friend.ErrUserNotFound, http.StatusNotFound},
			"self":              {friend.ErrCannotTargetSelf, http.StatusBadRequest},
			"blocked":           {friend.ErrBlocked, http.StatusForbidden},
		}
		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				svc := &svcMock{sendRequest: func(context.Context, uuid.UUID, uuid.UUID, string) (*friend.RequestResponse, error) {
					return nil, c.err
				}}
				w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/friends/requests",
					friend.SendRequestPayload{ReceiverID: receiverID.String()})
				assert.Equal(t, c.code, w.Code)
			})
		}
	})
}

func TestHandlerAcceptDeclineCancel(t *testing.T) {
	uid := uuid.New()
	reqID := uuid.New()

	t.Run("accept_success", func(t *testing.T) {
		called := false
		svc := &svcMock{acceptRequest: func(_ context.Context, u, r uuid.UUID) error {
			called = true
			assert.Equal(t, uid, u)
			assert.Equal(t, reqID, r)
			return nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/friends/requests/"+reqID.String()+"/accept", nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("decline_forbidden_returns_403", func(t *testing.T) {
		svc := &svcMock{declineRequest: func(context.Context, uuid.UUID, uuid.UUID) error { return friend.ErrForbidden }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/friends/requests/"+reqID.String()+"/decline", nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("cancel_not_found_returns_404", func(t *testing.T) {
		svc := &svcMock{cancelRequest: func(context.Context, uuid.UUID, uuid.UUID) error { return friend.ErrRequestNotFound }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodDelete, "/api/v1/friends/requests/"+reqID.String(), nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid_request_id_returns_400", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodPost, "/api/v1/friends/requests/not-a-uuid/accept", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlerListRequests(t *testing.T) {
	uid := uuid.New()
	svc := &svcMock{
		listIncomingRequests: func(context.Context, uuid.UUID) ([]friend.RequestResponse, error) {
			return []friend.RequestResponse{{ID: uuid.NewString(), Status: "pending"}}, nil
		},
		listOutgoingRequests: func(context.Context, uuid.UUID) ([]friend.RequestResponse, error) {
			return []friend.RequestResponse{}, nil
		},
	}
	r := newRouter(svc, &uid)

	w := doJSON(t, r, http.MethodGet, "/api/v1/friends/requests", nil)
	require.Equal(t, http.StatusOK, w.Code)
	env := parseEnvelope(t, w)
	var rs []friend.RequestResponse
	require.NoError(t, json.Unmarshal(env.Data, &rs))
	assert.Len(t, rs, 1)

	w = doJSON(t, r, http.MethodGet, "/api/v1/friends/requests/outgoing", nil)
	require.Equal(t, http.StatusOK, w.Code)
}

// ---- block list ----

func TestHandlerBlock(t *testing.T) {
	uid := uuid.New()
	targetID := uuid.New()

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &svcMock{blockUser: func(_ context.Context, b, target uuid.UUID) error {
			called = true
			assert.Equal(t, uid, b)
			assert.Equal(t, targetID, target)
			return nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/friends/blocks",
			friend.BlockPayload{TargetID: targetID.String()})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("invalid_payload_returns_400", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodPost, "/api/v1/friends/blocks",
			friend.BlockPayload{TargetID: "nope"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("target_missing_returns_404", func(t *testing.T) {
		svc := &svcMock{blockUser: func(context.Context, uuid.UUID, uuid.UUID) error { return friend.ErrUserNotFound }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/friends/blocks",
			friend.BlockPayload{TargetID: targetID.String()})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandlerUnblockAndList(t *testing.T) {
	uid := uuid.New()
	targetID := uuid.New()

	t.Run("unblock_success", func(t *testing.T) {
		called := false
		svc := &svcMock{unblockUser: func(_ context.Context, _, _ uuid.UUID) error { called = true; return nil }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodDelete, "/api/v1/friends/blocks/"+targetID.String(), nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("list_returns_data", func(t *testing.T) {
		svc := &svcMock{listBlocks: func(context.Context, uuid.UUID) ([]friend.BlockResponse, error) {
			return []friend.BlockResponse{{User: friend.UserSummary{ID: uuid.NewString(), Name: "X"}, CreatedAt: time.Now()}}, nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodGet, "/api/v1/friends/blocks", nil)
		require.Equal(t, http.StatusOK, w.Code)
		env := parseEnvelope(t, w)
		var bs []friend.BlockResponse
		require.NoError(t, json.Unmarshal(env.Data, &bs))
		require.Len(t, bs, 1)
		assert.Equal(t, "X", bs[0].User.Name)
	})
}

// silence unused linter on ptr helper if not used in every case
var _ = ptr
