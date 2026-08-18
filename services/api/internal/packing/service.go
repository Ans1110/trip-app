package packing

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TripAuthorizer is the narrow slice of trip.Repository this package needs.
// TripOwnerID is used to authorize edits/deletes on items the caller didn't
// create — the trip owner can always moderate.
type TripAuthorizer interface {
	IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error)
	TripOwnerID(ctx context.Context, tripID uuid.UUID) (uuid.UUID, error)
}

type Broadcaster interface {
	PublishTripEvent(ctx context.Context, tripID uuid.UUID, payload []byte)
}

type IService interface {
	CreateItem(ctx context.Context, userID, tripID uuid.UUID, p CreateItemPayload) (*ItemDTO, error)
	ListItems(ctx context.Context, userID, tripID uuid.UUID) ([]ItemDTO, error)
	UpdateItem(ctx context.Context, userID, itemID uuid.UUID, p UpdateItemPayload) (*ItemDTO, error)
	DeleteItem(ctx context.Context, userID, itemID uuid.UUID) error
	SetPacked(ctx context.Context, userID, itemID uuid.UUID, packed bool) (*ItemDTO, error)
}

type ServiceConfig struct {
	Repo        IRepository
	TripAuth    TripAuthorizer
	Broadcaster Broadcaster
	Logger      *zap.Logger
}

type service struct {
	repo   IRepository
	auth   TripAuthorizer
	bcast  Broadcaster
	logger *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		auth:   cfg.TripAuth,
		bcast:  cfg.Broadcaster,
		logger: logger.With(zap.String("layer", "packing.service")),
	}
}

const (
	maxNameLen = 200
	maxNoteLen = 2000
)

func (s *service) CreateItem(ctx context.Context, userID, tripID uuid.UUID, p CreateItemPayload) (*ItemDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, ErrInvalidPayload
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	qty := p.Quantity
	if qty < 1 {
		qty = 1
	}
	note := p.Note
	if len(note) > maxNoteLen {
		note = note[:maxNoteLen]
	}

	sortOrder := p.SortOrder
	if sortOrder == 0 {
		next, err := s.repo.NextSortOrder(ctx, tripID)
		if err != nil {
			return nil, err
		}
		sortOrder = next
	}

	item := &Item{
		TripID:    tripID,
		CreatedBy: userID,
		Name:      name,
		Quantity:  qty,
		Category:  strings.TrimSpace(p.Category),
		Note:      note,
		SortOrder: sortOrder,
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}
	dto := toItemDTO(item, ItemPackInfo{})
	s.broadcast(ctx, tripID, BroadcastItemCreated, dto)
	return dto, nil
}

func (s *service) ListItems(ctx context.Context, userID, tripID uuid.UUID) ([]ItemDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListItemsByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	packInfo, err := s.repo.PackInfoForItems(ctx, ids, userID)
	if err != nil {
		return nil, err
	}
	out := make([]ItemDTO, 0, len(rows))
	for i := range rows {
		out = append(out, *toItemDTO(&rows[i], packInfo[rows[i].ID]))
	}
	return out, nil
}

func (s *service) UpdateItem(ctx context.Context, userID, itemID uuid.UUID, p UpdateItemPayload) (*ItemDTO, error) {
	item, err := s.loadItemForEditor(ctx, userID, itemID)
	if err != nil {
		return nil, err
	}

	patch := map[string]any{}
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" {
			return nil, ErrInvalidPayload
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen]
		}
		patch["name"] = name
	}
	if p.Quantity != nil {
		if *p.Quantity < 1 {
			return nil, ErrInvalidPayload
		}
		patch["quantity"] = *p.Quantity
	}
	if p.Category != nil {
		patch["category"] = strings.TrimSpace(*p.Category)
	}
	if p.Note != nil {
		note := *p.Note
		if len(note) > maxNoteLen {
			note = note[:maxNoteLen]
		}
		patch["note"] = note
	}
	if p.SortOrder != nil {
		patch["sort_order"] = *p.SortOrder
	}

	target := item
	if len(patch) > 0 {
		updated, err := s.repo.UpdateItem(ctx, itemID, patch)
		if err != nil {
			return nil, err
		}
		target = updated
	}
	dto, err := s.itemDTOWithPack(ctx, target, userID)
	if err != nil {
		return nil, err
	}
	if len(patch) > 0 {
		s.broadcast(ctx, item.TripID, BroadcastItemUpdated, dto)
	}
	return dto, nil
}

func (s *service) DeleteItem(ctx context.Context, userID, itemID uuid.UUID) error {
	item, err := s.loadItemForEditor(ctx, userID, itemID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteItem(ctx, itemID); err != nil {
		return err
	}
	s.broadcastDeleted(ctx, item.TripID, itemID)
	return nil
}

// SetPacked toggles the caller's own pack state on an item. Any trip member
// can flip their own checkbox — this is intentionally not gated by the
// creator/owner check that governs edit/delete.
func (s *service) SetPacked(ctx context.Context, userID, itemID uuid.UUID, packed bool) (*ItemDTO, error) {
	item, err := s.repo.FindItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrItemNotFound
	}
	if err := s.mustBeMember(ctx, item.TripID, userID); err != nil {
		return nil, err
	}
	if err := s.repo.SetPacked(ctx, itemID, userID, packed); err != nil {
		return nil, err
	}
	dto, err := s.itemDTOWithPack(ctx, item, userID)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, item.TripID, BroadcastItemUpdated, dto)
	return dto, nil
}

// ---- helpers ----

func (s *service) mustBeMember(ctx context.Context, tripID, userID uuid.UUID) error {
	if s.auth == nil {
		return ErrForbidden
	}
	ok, err := s.auth.IsRoomMember(ctx, tripID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// loadItemForEditor fetches the item, verifies the caller belongs to the trip,
// and confirms they are either the item's creator or the trip owner.
func (s *service) loadItemForEditor(ctx context.Context, userID, itemID uuid.UUID) (*Item, error) {
	item, err := s.repo.FindItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrItemNotFound
	}
	if err := s.mustBeMember(ctx, item.TripID, userID); err != nil {
		return nil, err
	}
	if item.CreatedBy == userID {
		return item, nil
	}
	ownerID, err := s.auth.TripOwnerID(ctx, item.TripID)
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrForbidden
	}
	return item, nil
}

func (s *service) itemDTOWithPack(ctx context.Context, item *Item, userID uuid.UUID) (*ItemDTO, error) {
	info, err := s.repo.PackInfoForItems(ctx, []uuid.UUID{item.ID}, userID)
	if err != nil {
		return nil, err
	}
	return toItemDTO(item, info[item.ID]), nil
}

func (s *service) broadcast(ctx context.Context, tripID uuid.UUID, kind BroadcastKind, dto *ItemDTO) {
	if s.bcast == nil || dto == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Type   BroadcastKind `json:"type"`
		TripID uuid.UUID     `json:"trip_id"`
		Item   *ItemDTO      `json:"item"`
	}{
		Type:   kind,
		TripID: tripID,
		Item:   dto,
	})
	if err != nil {
		s.logger.Warn("packing: marshal broadcast", zap.Error(err))
		return
	}
	s.bcast.PublishTripEvent(ctx, tripID, payload)
}

func (s *service) broadcastDeleted(ctx context.Context, tripID, itemID uuid.UUID) {
	if s.bcast == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Type   BroadcastKind `json:"type"`
		TripID uuid.UUID     `json:"trip_id"`
		ItemID uuid.UUID     `json:"item_id"`
	}{
		Type:   BroadcastItemDeleted,
		TripID: tripID,
		ItemID: itemID,
	})
	if err != nil {
		s.logger.Warn("packing: marshal delete broadcast", zap.Error(err))
		return
	}
	s.bcast.PublishTripEvent(ctx, tripID, payload)
}

func toItemDTO(i *Item, info ItemPackInfo) *ItemDTO {
	if i == nil {
		return nil
	}
	return &ItemDTO{
		ID:          i.ID,
		TripID:      i.TripID,
		CreatedBy:   i.CreatedBy,
		Name:        i.Name,
		Quantity:    i.Quantity,
		Category:    i.Category,
		Note:        i.Note,
		PackedByMe:  info.PackedByMe,
		PackedCount: info.PackedCount,
		SortOrder:   i.SortOrder,
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
	}
}
