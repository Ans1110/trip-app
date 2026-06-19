package friend

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrCannotTargetSelf  = errors.New("cannot target self")
	ErrAlreadyFriends    = errors.New("already friends")
	ErrNotFriends        = errors.New("not friends")
	ErrRequestExists     = errors.New("friend request already pending")
	ErrRequestNotFound   = errors.New("friend request not found")
	ErrForbidden         = errors.New("forbidden")
	ErrBlocked           = errors.New("interaction blocked")
	ErrInviteNotFound    = errors.New("invite not found")
	ErrInviteInvalid     = errors.New("invite invalid or expired")
	ErrInviteExhausted   = errors.New("invite exhausted")
	ErrInviteSelf        = errors.New("cannot accept your own invite")
)

const (
	defaultInviteTTL     = 7 * 24 * time.Hour
	maxInviteTTL         = 30 * 24 * time.Hour
	defaultInviteMaxUses = 1
	inviteTokenBytes     = 24 // 192-bit token, base64url ~32 chars
)

type IService interface {
	ListFriends(ctx context.Context, userID uuid.UUID) ([]FriendResponse, error)
	RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error
	SearchUsers(ctx context.Context, requesterID uuid.UUID, query string, limit int) ([]UserSummary, error)

	SendRequest(ctx context.Context, senderID, receiverID uuid.UUID, message string) (*RequestResponse, error)
	AcceptRequest(ctx context.Context, userID, requestID uuid.UUID) error
	DeclineRequest(ctx context.Context, userID, requestID uuid.UUID) error
	CancelRequest(ctx context.Context, userID, requestID uuid.UUID) error
	ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]RequestResponse, error)
	ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]RequestResponse, error)

	BlockUser(ctx context.Context, blockerID, targetID uuid.UUID) error
	UnblockUser(ctx context.Context, blockerID, targetID uuid.UUID) error
	ListBlocks(ctx context.Context, userID uuid.UUID) ([]BlockResponse, error)

	CreateInvite(ctx context.Context, userID uuid.UUID, ttl time.Duration, maxUses int) (*InviteResponse, error)
	ListInvites(ctx context.Context, userID uuid.UUID) ([]InviteResponse, error)
	RevokeInvite(ctx context.Context, userID, inviteID uuid.UUID) error
	AcceptInvite(ctx context.Context, userID uuid.UUID, token string) (*UserSummary, error)
}

type service struct {
	repo      IRepository
	logger    *zap.Logger
	webURL    string
	inviteTTL time.Duration
	audit     AuditWriter
}

type ServiceConfig struct {
	Repo      IRepository
	Logger    *zap.Logger
	WebURL    string
	InviteTTL time.Duration
	Audit     AuditWriter
}

func NewService(cfg ServiceConfig) IService {
	ttl := cfg.InviteTTL
	if ttl <= 0 {
		ttl = defaultInviteTTL
	}
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:      cfg.Repo,
		logger:    logger.With(zap.String("layer", "friend.service")),
		webURL:    strings.TrimRight(cfg.WebURL, "/"),
		inviteTTL: ttl,
		audit:     cfg.Audit,
	}
}

// recordAudit writes a friend-domain audit row. No-op when no writer is wired
// (e.g. in unit tests). Failures are logged but never propagated — audit is
// best-effort and must not break the user-facing action.
func (s *service) recordAudit(
	ctx context.Context,
	action auth.AuditAction,
	status auth.AuditStatus,
	actorID uuid.UUID,
	targetID *uuid.UUID,
	resourceType string,
	resourceID string,
	detail map[string]any,
) {
	if s.audit == nil {
		return
	}
	log := &auth.AuditLog{
		ID:           uuid.New(),
		Action:       action,
		Status:       status,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		TraceID:      middleware.GetTraceID(ctx),
		Detail:       detail,
	}
	if actorID != uuid.Nil {
		actor := actorID
		log.ActorUserID = &actor
	}
	if targetID != nil && *targetID != uuid.Nil {
		t := *targetID
		log.TargetUserID = &t
	}
	if rid := middleware.RequestIDFromContext(ctx); rid != uuid.Nil {
		log.RequestID = &rid
	}
	meta := auditMetaFromCtx(ctx)
	if meta.IPAddress != "" {
		ip := meta.IPAddress
		log.IPAddress = &ip
	}
	if meta.UserAgent != "" {
		log.UserAgent = meta.UserAgent
	}
	if err := s.audit.CreateAuditLog(ctx, log); err != nil {
		s.logger.Warn("audit log write failed",
			zap.Error(err),
			zap.String("action", string(action)),
		)
	}
}

func (s *service) ListFriends(ctx context.Context, userID uuid.UUID) ([]FriendResponse, error) {
	rows, err := s.repo.ListFriends(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []FriendResponse{}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.FriendID)
	}
	users, err := s.repo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]FriendResponse, 0, len(rows))
	for _, r := range rows {
		u, ok := users[r.FriendID]
		if !ok {
			continue
		}
		out = append(out, FriendResponse{
			User:      toUserSummary(u),
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *service) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if userID == friendID {
		return ErrCannotTargetSelf
	}
	areFriends, err := s.repo.AreFriends(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !areFriends {
		return ErrNotFriends
	}
	if err := s.repo.DeleteFriendship(ctx, userID, friendID); err != nil {
		return err
	}
	s.recordAudit(ctx, auth.AuditFriendRemoved, auth.AuditSuccess,
		userID, &friendID, auth.AuditResourceFriendship, friendID.String(), nil)
	return nil
}

func (s *service) SearchUsers(ctx context.Context, requesterID uuid.UUID, query string, limit int) ([]UserSummary, error) {
	users, err := s.repo.SearchUsers(ctx, requesterID, query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]UserSummary, 0, len(users))
	for _, u := range users {
		out = append(out, toUserSummary(u))
	}
	return out, nil
}

func (s *service) SendRequest(ctx context.Context, senderID, receiverID uuid.UUID, message string) (*RequestResponse, error) {
	if senderID == receiverID {
		return nil, ErrCannotTargetSelf
	}
	receiver, err := s.repo.FindUserByID(ctx, receiverID)
	if err != nil {
		return nil, err
	}
	if receiver == nil || receiver.Status != auth.UserStatusActive {
		return nil, ErrUserNotFound
	}
	if err := s.checkNotBlocked(ctx, senderID, receiverID); err != nil {
		return nil, err
	}
	already, err := s.repo.AreFriends(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if already {
		return nil, ErrAlreadyFriends
	}
	existing, err := s.repo.FindPendingRequest(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRequestExists
	}

	req := &Request{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Message:    strings.TrimSpace(message),
	}
	if err := s.repo.CreateOrReviveRequest(ctx, req); err != nil {
		return nil, err
	}

	sender, err := s.repo.FindUserByID(ctx, senderID)
	if err != nil {
		return nil, err
	}
	if sender == nil {
		return nil, ErrUserNotFound
	}
	s.recordAudit(ctx, auth.AuditFriendRequestSent, auth.AuditSuccess,
		senderID, &receiverID, auth.AuditResourceFriendRequest, req.ID.String(), nil)
	return buildRequestResponse(req, *sender, *receiver), nil
}

func (s *service) AcceptRequest(ctx context.Context, userID, requestID uuid.UUID) error {
	req, err := s.repo.FindRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil || req.Status != RequestPending {
		return ErrRequestNotFound
	}
	if req.ReceiverID != userID {
		return ErrForbidden
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.UpdateRequestStatus(ctx, req.ID, RequestAccepted); err != nil {
			return err
		}
		return tx.CreateFriendship(ctx, req.SenderID, req.ReceiverID)
	}); err != nil {
		return err
	}
	sender := req.SenderID
	s.recordAudit(ctx, auth.AuditFriendRequestAccepted, auth.AuditSuccess,
		userID, &sender, auth.AuditResourceFriendRequest, req.ID.String(), nil)
	return nil
}

func (s *service) DeclineRequest(ctx context.Context, userID, requestID uuid.UUID) error {
	req, err := s.repo.FindRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil || req.Status != RequestPending {
		return ErrRequestNotFound
	}
	if req.ReceiverID != userID {
		return ErrForbidden
	}
	if err := s.repo.UpdateRequestStatus(ctx, req.ID, RequestDeclined); err != nil {
		return err
	}
	sender := req.SenderID
	s.recordAudit(ctx, auth.AuditFriendRequestDeclined, auth.AuditSuccess,
		userID, &sender, auth.AuditResourceFriendRequest, req.ID.String(), nil)
	return nil
}

func (s *service) CancelRequest(ctx context.Context, userID, requestID uuid.UUID) error {
	req, err := s.repo.FindRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil || req.Status != RequestPending {
		return ErrRequestNotFound
	}
	if req.SenderID != userID {
		return ErrForbidden
	}
	if err := s.repo.UpdateRequestStatus(ctx, req.ID, RequestCancelled); err != nil {
		return err
	}
	receiver := req.ReceiverID
	s.recordAudit(ctx, auth.AuditFriendRequestCancelled, auth.AuditSuccess,
		userID, &receiver, auth.AuditResourceFriendRequest, req.ID.String(), nil)
	return nil
}

func (s *service) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]RequestResponse, error) {
	rows, err := s.repo.ListIncomingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.hydrateRequests(ctx, rows)
}

func (s *service) ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]RequestResponse, error) {
	rows, err := s.repo.ListOutgoingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.hydrateRequests(ctx, rows)
}

func (s *service) BlockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	if blockerID == targetID {
		return ErrCannotTargetSelf
	}
	target, err := s.repo.FindUserByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrUserNotFound
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.BlockUser(ctx, blockerID, targetID); err != nil {
			return err
		}
		if err := tx.DeleteFriendship(ctx, blockerID, targetID); err != nil {
			return err
		}
		if pending, err := tx.FindPendingRequest(ctx, blockerID, targetID); err != nil {
			return err
		} else if pending != nil {
			if err := tx.UpdateRequestStatus(ctx, pending.ID, RequestCancelled); err != nil {
				return err
			}
		}
		if pending, err := tx.FindPendingRequest(ctx, targetID, blockerID); err != nil {
			return err
		} else if pending != nil {
			if err := tx.UpdateRequestStatus(ctx, pending.ID, RequestDeclined); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	s.recordAudit(ctx, auth.AuditFriendBlocked, auth.AuditSuccess,
		blockerID, &targetID, auth.AuditResourceFriendBlock, targetID.String(), nil)
	return nil
}

func (s *service) UnblockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	if err := s.repo.UnblockUser(ctx, blockerID, targetID); err != nil {
		return err
	}
	s.recordAudit(ctx, auth.AuditFriendUnblocked, auth.AuditSuccess,
		blockerID, &targetID, auth.AuditResourceFriendBlock, targetID.String(), nil)
	return nil
}

func (s *service) ListBlocks(ctx context.Context, userID uuid.UUID) ([]BlockResponse, error) {
	rows, err := s.repo.ListBlocks(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []BlockResponse{}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.BlockedID)
	}
	users, err := s.repo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]BlockResponse, 0, len(rows))
	for _, r := range rows {
		u, ok := users[r.BlockedID]
		if !ok {
			continue
		}
		out = append(out, BlockResponse{
			User:      toUserSummary(u),
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *service) CreateInvite(ctx context.Context, userID uuid.UUID, ttl time.Duration, maxUses int) (*InviteResponse, error) {
	if ttl <= 0 {
		ttl = s.inviteTTL
	}
	if ttl > maxInviteTTL {
		ttl = maxInviteTTL
	}
	if maxUses <= 0 {
		maxUses = defaultInviteMaxUses
	}

	token, err := generateToken(inviteTokenBytes)
	if err != nil {
		return nil, err
	}
	hash := hashToken(token)
	now := time.Now()
	rec := &InviteToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hash,
		MaxUses:   maxUses,
		Uses:      0,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
	if err := s.repo.CreateInviteToken(ctx, rec); err != nil {
		return nil, err
	}
	s.recordAudit(ctx, auth.AuditFriendInviteCreated, auth.AuditSuccess,
		userID, nil, auth.AuditResourceFriendInvite, rec.ID.String(),
		map[string]any{"max_uses": rec.MaxUses, "expires_at": rec.ExpiresAt})
	return &InviteResponse{
		ID:        rec.ID.String(),
		Token:     token,
		URL:       s.buildInviteURL(token),
		MaxUses:   rec.MaxUses,
		Uses:      rec.Uses,
		ExpiresAt: rec.ExpiresAt,
		CreatedAt: rec.CreatedAt,
	}, nil
}

func (s *service) ListInvites(ctx context.Context, userID uuid.UUID) ([]InviteResponse, error) {
	rows, err := s.repo.ListActiveInvites(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]InviteResponse, 0, len(rows))
	for _, t := range rows {
		out = append(out, InviteResponse{
			ID:        t.ID.String(),
			MaxUses:   t.MaxUses,
			Uses:      t.Uses,
			ExpiresAt: t.ExpiresAt,
			CreatedAt: t.CreatedAt,
		})
	}
	return out, nil
}

func (s *service) RevokeInvite(ctx context.Context, userID, inviteID uuid.UUID) error {
	if err := s.repo.RevokeInviteToken(ctx, userID, inviteID); err != nil {
		return err
	}
	s.recordAudit(ctx, auth.AuditFriendInviteRevoked, auth.AuditSuccess,
		userID, nil, auth.AuditResourceFriendInvite, inviteID.String(), nil)
	return nil
}

func (s *service) AcceptInvite(ctx context.Context, userID uuid.UUID, token string) (*UserSummary, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInviteInvalid
	}
	hash := hashToken(token)
	rec, err := s.repo.FindInviteTokenByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrInviteInvalid
	}
	if rec.UserID == userID {
		return nil, ErrInviteSelf
	}
	if err := s.checkNotBlocked(ctx, userID, rec.UserID); err != nil {
		return nil, err
	}
	inviter, err := s.repo.FindUserByID(ctx, rec.UserID)
	if err != nil {
		return nil, err
	}
	if inviter == nil || inviter.Status != auth.UserStatusActive {
		return nil, ErrInviteInvalid
	}

	var summary UserSummary
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		already, err := tx.AreFriends(ctx, userID, rec.UserID)
		if err != nil {
			return err
		}
		if already {
			return ErrAlreadyFriends
		}
		if err := tx.IncrementInviteUses(ctx, rec.ID); err != nil {
			return err
		}
		if err := tx.CreateFriendship(ctx, userID, rec.UserID); err != nil {
			return err
		}
		summary = toUserSummary(*inviter)
		return nil
	})
	if err != nil {
		return nil, err
	}
	inviterID := rec.UserID
	s.recordAudit(ctx, auth.AuditFriendInviteAccepted, auth.AuditSuccess,
		userID, &inviterID, auth.AuditResourceFriendInvite, rec.ID.String(), nil)
	return &summary, nil
}

func (s *service) checkNotBlocked(ctx context.Context, a, b uuid.UUID) error {
	if blocked, err := s.repo.IsBlocked(ctx, a, b); err != nil {
		return err
	} else if blocked {
		return ErrBlocked
	}
	if blocked, err := s.repo.IsBlocked(ctx, b, a); err != nil {
		return err
	} else if blocked {
		return ErrBlocked
	}
	return nil
}

func (s *service) hydrateRequests(ctx context.Context, rows []Request) ([]RequestResponse, error) {
	if len(rows) == 0 {
		return []RequestResponse{}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows)*2)
	seen := make(map[uuid.UUID]struct{}, len(rows)*2)
	for _, r := range rows {
		if _, ok := seen[r.SenderID]; !ok {
			seen[r.SenderID] = struct{}{}
			ids = append(ids, r.SenderID)
		}
		if _, ok := seen[r.ReceiverID]; !ok {
			seen[r.ReceiverID] = struct{}{}
			ids = append(ids, r.ReceiverID)
		}
	}
	users, err := s.repo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]RequestResponse, 0, len(rows))
	for _, r := range rows {
		sender, sok := users[r.SenderID]
		receiver, rok := users[r.ReceiverID]
		if !sok || !rok {
			continue
		}
		out = append(out, *buildRequestResponse(&r, sender, receiver))
	}
	return out, nil
}

func (s *service) buildInviteURL(token string) string {
	if s.webURL == "" {
		return ""
	}
	return s.webURL + "/friends/invite/" + token
}

func toUserSummary(u auth.User) UserSummary {
	return UserSummary{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
	}
}

func buildRequestResponse(r *Request, sender, receiver auth.User) *RequestResponse {
	return &RequestResponse{
		ID:        r.ID.String(),
		Status:    string(r.Status),
		Message:   r.Message,
		Sender:    toUserSummary(sender),
		Receiver:  toUserSummary(receiver),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
