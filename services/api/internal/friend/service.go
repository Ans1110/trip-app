package friend

import (
	"context"
	"errors"
	"strings"

	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/pkg/event"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrCannotTargetSelf = errors.New("cannot target self")
	ErrAlreadyFriends   = errors.New("already friends")
	ErrNotFriends       = errors.New("not friends")
	ErrRequestExists    = errors.New("friend request already pending")
	ErrRequestNotFound  = errors.New("friend request not found")
	ErrForbidden        = errors.New("forbidden")
	ErrBlocked          = errors.New("interaction blocked")
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
}

type service struct {
	repo   IRepository
	logger *zap.Logger
	audit  audit.Writer
	bus    *event.Bus
}

type ServiceConfig struct {
	Repo   IRepository
	Logger *zap.Logger
	Audit  audit.Writer
	Bus    *event.Bus
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		logger: logger.With(zap.String("layer", "friend.service")),
		audit:  cfg.Audit,
		bus:    cfg.Bus,
	}
}

func (s *service) publish(ctx context.Context, evtType string, actorID, targetID uuid.UUID, payload map[string]any) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(ctx, event.Event{
		Type:      evtType,
		Payload:   payload,
		UserID:    actorID,
		TargetID:  targetID,
		Timestamp: time.Now(),
	})
}

func (s *service) recordAudit(
	ctx context.Context,
	action audit.Action,
	status audit.Status,
	actorID uuid.UUID,
	targetID *uuid.UUID,
	resourceType string,
	resourceID string,
	detail map[string]any,
) {
	if s.audit == nil {
		return
	}
	log := &audit.Log{
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
	if err := s.audit.Create(ctx, log); err != nil {
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
	s.recordAudit(ctx, AuditFriendRemoved, audit.Success,
		userID, &friendID, AuditResourceFriendship, friendID.String(), nil)
	s.publish(ctx, event.EventFriendRemoved, userID, friendID, map[string]any{
		"actor_id":  userID.String(),
		"friend_id": friendID.String(),
	})
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
	// A pending request from the receiver to the sender means the user should
	// accept that incoming request rather than send their own — don't overwrite it.
	incoming, err := s.repo.FindPendingRequest(ctx, receiverID, senderID)
	if err != nil {
		return nil, err
	}
	if incoming != nil {
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
	s.recordAudit(ctx, AuditFriendRequestSent, audit.Success,
		senderID, &receiverID, AuditResourceFriendRequest, req.ID.String(), nil)
	s.publish(ctx, event.EventFriendRequestSent, senderID, receiverID, map[string]any{
		"request_id":  req.ID.String(),
		"sender_id":   senderID.String(),
		"receiver_id": receiverID.String(),
	})
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
	s.recordAudit(ctx, AuditFriendRequestAccepted, audit.Success,
		userID, &sender, AuditResourceFriendRequest, req.ID.String(), nil)
	s.publish(ctx, event.EventFriendRequestAccepted, userID, sender, map[string]any{
		"request_id":  req.ID.String(),
		"sender_id":   sender.String(),
		"receiver_id": userID.String(),
	})
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
	s.recordAudit(ctx, AuditFriendRequestDeclined, audit.Success,
		userID, &sender, AuditResourceFriendRequest, req.ID.String(), nil)
	s.publish(ctx, event.EventFriendRequestDeclined, userID, sender, map[string]any{
		"request_id": req.ID.String(),
		"sender_id":  sender.String(),
	})
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
	s.recordAudit(ctx, AuditFriendRequestCancelled, audit.Success,
		userID, &receiver, AuditResourceFriendRequest, req.ID.String(), nil)
	s.publish(ctx, event.EventFriendRequestCancelled, userID, receiver, map[string]any{
		"request_id":  req.ID.String(),
		"receiver_id": receiver.String(),
	})
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
	s.recordAudit(ctx, AuditFriendBlocked, audit.Success,
		blockerID, &targetID, AuditResourceFriendBlock, targetID.String(), nil)
	s.publish(ctx, event.EventFriendBlocked, blockerID, targetID, map[string]any{
		"blocker_id": blockerID.String(),
		"target_id":  targetID.String(),
	})
	return nil
}

func (s *service) UnblockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	if err := s.repo.UnblockUser(ctx, blockerID, targetID); err != nil {
		return err
	}
	s.recordAudit(ctx, AuditFriendUnblocked, audit.Success,
		blockerID, &targetID, AuditResourceFriendBlock, targetID.String(), nil)
	s.publish(ctx, event.EventFriendUnblocked, blockerID, targetID, map[string]any{
		"blocker_id": blockerID.String(),
		"target_id":  targetID.String(),
	})
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
