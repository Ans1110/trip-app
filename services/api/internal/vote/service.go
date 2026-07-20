package vote

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TripAuthorizer is the narrow slice of trip.Repository
type TripAuthorizer interface {
	IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error)
}

// Broadcaster publishes an opaque JSON frame to every WS client currently
// subscribed to a trip's realtime channel. Vote uses the trip realtime hub
// (rather than a hub of its own) so a member's single WS carries both
// itinerary edits and poll updates.
type Broadcaster interface {
	PublishTripEvent(ctx context.Context, tripID uuid.UUID, payload []byte)
}

type IService interface {
	CreatePoll(ctx context.Context, userID, tripID uuid.UUID, p CreatePollPayload) (*PollDTO, error)
	ListPolls(ctx context.Context, userID, tripID uuid.UUID) ([]PollDTO, error)
	GetPoll(ctx context.Context, userID, pollID uuid.UUID) (*PollDTO, error)
	AddOption(ctx context.Context, userID, pollID uuid.UUID, p AddOptionPayload) (*PollDTO, error)
	DeleteOption(ctx context.Context, userID, optionID uuid.UUID) (*PollDTO, error)
	CastVote(ctx context.Context, userID, pollID uuid.UUID, p CastVotePayload) (*PollDTO, error)
	ClosePoll(ctx context.Context, userID, pollID uuid.UUID) (*PollDTO, error)
	DeletePoll(ctx context.Context, userID, pollID uuid.UUID) error

	// SweepExpiredPolls closes polls whose deadline_at has passed. Called by
	// the background scheduler.
	SweepExpiredPolls(ctx context.Context) (int, error)
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
		logger: logger.With(zap.String("layer", "vote.service")),
	}
}

const (
	maxTitleLen       = 200
	maxDescriptionLen = 2000
	maxOptionTextLen  = 200
	maxOptionsPerPoll = 50
)

// ---- Create ----

func (s *service) CreatePoll(ctx context.Context, userID, tripID uuid.UUID, p CreatePollPayload) (*PollDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}

	poll := &Poll{
		TripID:           tripID,
		CreatedBy:        userID,
		Type:             normalizeType(p.Type),
		Title:            strings.TrimSpace(p.Title),
		Description:      strings.TrimSpace(p.Description),
		IsAnonymous:      p.IsAnonymous,
		MaxChoices:       p.MaxChoices,
		AllowOptionAdd:   p.AllowOptionAdd,
		ResultVisibility: normalizeVisibility(p.ResultVisibility),
		DeadlineAt:       p.DeadlineAt,
		Status:           StatusOpen,
	}
	if poll.Title == "" {
		return nil, ErrInvalidPayload
	}
	if len(poll.Title) > maxTitleLen {
		poll.Title = poll.Title[:maxTitleLen]
	}
	if len(poll.Description) > maxDescriptionLen {
		poll.Description = poll.Description[:maxDescriptionLen]
	}
	if poll.MaxChoices < 1 {
		poll.MaxChoices = 1
	}
	if poll.DeadlineAt != nil && !poll.DeadlineAt.After(time.Now()) {
		return nil, ErrInvalidPayload
	}
	if len(p.Options) > maxOptionsPerPoll {
		return nil, ErrInvalidPayload
	}

	err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.CreatePoll(ctx, poll); err != nil {
			return err
		}
		for i, in := range p.Options {
			text := strings.TrimSpace(in.Text)
			if text == "" {
				continue
			}
			if len(text) > maxOptionTextLen {
				text = text[:maxOptionTextLen]
			}
			opt := &Option{
				PollID:    poll.ID,
				Text:      text,
				Meta:      normalizeMeta(in.Meta),
				AddedBy:   userID,
				SortOrder: i,
			}
			if err := tx.CreateOption(ctx, opt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	dto, err := s.buildPollDTO(ctx, poll, userID)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, tripID, BroadcastPollCreated, dto)
	return dto, nil
}

// ---- List / Get ----

func (s *service) ListPolls(ctx context.Context, userID, tripID uuid.UUID) ([]PollDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	polls, err := s.repo.ListPollsByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]PollDTO, 0, len(polls))
	for i := range polls {
		dto, err := s.buildPollDTO(ctx, &polls[i], userID)
		if err != nil {
			return nil, err
		}
		out = append(out, *dto)
	}
	return out, nil
}

func (s *service) GetPoll(ctx context.Context, userID, pollID uuid.UUID) (*PollDTO, error) {
	poll, err := s.repo.FindPoll(ctx, pollID)
	if err != nil {
		return nil, err
	}
	if poll == nil {
		return nil, ErrPollNotFound
	}
	if err := s.mustBeMember(ctx, poll.TripID, userID); err != nil {
		return nil, err
	}
	return s.buildPollDTO(ctx, poll, userID)
}

// ---- Options ----

func (s *service) AddOption(ctx context.Context, userID, pollID uuid.UUID, p AddOptionPayload) (*PollDTO, error) {
	poll, err := s.loadOpenPollForMember(ctx, userID, pollID)
	if err != nil {
		return nil, err
	}
	if poll.CreatedBy != userID && !poll.AllowOptionAdd {
		return nil, ErrForbidden
	}
	text := strings.TrimSpace(p.Text)
	if text == "" {
		return nil, ErrInvalidPayload
	}
	if len(text) > maxOptionTextLen {
		text = text[:maxOptionTextLen]
	}

	existing, err := s.repo.ListOptions(ctx, pollID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= maxOptionsPerPoll {
		return nil, ErrInvalidPayload
	}
	opt := &Option{
		PollID:    pollID,
		Text:      text,
		Meta:      normalizeMeta(p.Meta),
		AddedBy:   userID,
		SortOrder: len(existing),
	}
	if err := s.repo.CreateOption(ctx, opt); err != nil {
		return nil, err
	}
	dto, err := s.buildPollDTO(ctx, poll, userID)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, poll.TripID, BroadcastOptionAdded, dto)
	return dto, nil
}

func (s *service) DeleteOption(ctx context.Context, userID, optionID uuid.UUID) (*PollDTO, error) {
	opt, err := s.repo.FindOption(ctx, optionID)
	if err != nil {
		return nil, err
	}
	if opt == nil {
		return nil, ErrOptionNotFound
	}
	poll, err := s.loadOpenPollForMember(ctx, userID, opt.PollID)
	if err != nil {
		return nil, err
	}
	// The poll creator can prune any option; a regular member can only
	// remove one they added themselves.
	if poll.CreatedBy != userID && opt.AddedBy != userID {
		return nil, ErrForbidden
	}
	if err := s.repo.DeleteOption(ctx, optionID); err != nil {
		return nil, err
	}
	dto, err := s.buildPollDTO(ctx, poll, userID)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, poll.TripID, BroadcastPollUpdated, dto)
	return dto, nil
}

// ---- Cast ----

func (s *service) CastVote(ctx context.Context, userID, pollID uuid.UUID, p CastVotePayload) (*PollDTO, error) {
	poll, err := s.loadOpenPollForMember(ctx, userID, pollID)
	if err != nil {
		return nil, err
	}
	// Server-enforced deadline. The sweeper closes eventually, but a lazy
	// vote that lands in the gap between deadline and next sweep must still
	// be rejected.
	if poll.DeadlineAt != nil && !time.Now().Before(*poll.DeadlineAt) {
		return nil, ErrPollClosed
	}
	if len(p.OptionIDs) == 0 {
		return nil, ErrInvalidPayload
	}
	if len(p.OptionIDs) > poll.MaxChoices {
		return nil, ErrTooManyChoices
	}

	opts, err := s.repo.ListOptions(ctx, pollID)
	if err != nil {
		return nil, err
	}
	valid := make(map[uuid.UUID]struct{}, len(opts))
	for _, o := range opts {
		valid[o.ID] = struct{}{}
	}
	seen := make(map[uuid.UUID]struct{}, len(p.OptionIDs))
	for _, oid := range p.OptionIDs {
		if _, ok := valid[oid]; !ok {
			return nil, ErrOptionNotFound
		}
		if _, dup := seen[oid]; dup {
			return nil, ErrInvalidPayload
		}
		seen[oid] = struct{}{}
	}

	if err := s.repo.ReplaceBallots(ctx, pollID, userID, p.OptionIDs); err != nil {
		return nil, err
	}
	dto, err := s.buildPollDTO(ctx, poll, userID)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, poll.TripID, BroadcastVoteCast, dto)
	return dto, nil
}

// ---- Close ----

func (s *service) ClosePoll(ctx context.Context, userID, pollID uuid.UUID) (*PollDTO, error) {
	poll, err := s.repo.FindPoll(ctx, pollID)
	if err != nil {
		return nil, err
	}
	if poll == nil {
		return nil, ErrPollNotFound
	}
	if err := s.mustBeMember(ctx, poll.TripID, userID); err != nil {
		return nil, err
	}
	if poll.CreatedBy != userID {
		return nil, ErrForbidden
	}
	if poll.Status == StatusClosed {
		return s.buildPollDTO(ctx, poll, userID)
	}
	now := time.Now()
	if err := s.repo.MarkClosed(ctx, pollID, now); err != nil {
		return nil, err
	}
	poll.Status = StatusClosed
	poll.ClosedAt = &now
	poll.UpdatedAt = now
	dto, err := s.buildPollDTO(ctx, poll, userID)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, poll.TripID, BroadcastPollClosed, dto)
	return dto, nil
}

func (s *service) DeletePoll(ctx context.Context, userID, pollID uuid.UUID) error {
	poll, err := s.repo.FindPoll(ctx, pollID)
	if err != nil {
		return err
	}
	if poll == nil {
		return ErrPollNotFound
	}
	if err := s.mustBeMember(ctx, poll.TripID, userID); err != nil {
		return err
	}
	if poll.CreatedBy != userID {
		return ErrForbidden
	}
	if err := s.repo.DeletePoll(ctx, pollID); err != nil {
		return err
	}
	// Notify subscribers so their UI can drop the card.
	payload, _ := json.Marshal(map[string]any{
		"type":    string(BroadcastPollDeleted),
		"trip_id": poll.TripID,
		"poll_id": poll.ID,
	})
	if s.bcast != nil && payload != nil {
		s.bcast.PublishTripEvent(ctx, poll.TripID, payload)
	}
	return nil
}

// SweepExpiredPolls is invoked by the scheduler. It closes polls whose
// deadline has passed and broadcasts a POLL_CLOSED frame for each. Errors on
// individual polls are logged and the sweep continues.
func (s *service) SweepExpiredPolls(ctx context.Context) (int, error) {
	polls, err := s.repo.ClaimExpiredPolls(ctx, time.Now(), 128)
	if err != nil {
		return 0, err
	}
	closed := 0
	now := time.Now()
	for i := range polls {
		poll := &polls[i]
		if err := s.repo.MarkClosed(ctx, poll.ID, now); err != nil {
			s.logger.Warn("vote: mark closed",
				zap.String("poll_id", poll.ID.String()),
				zap.Error(err),
			)
			continue
		}
		poll.Status = StatusClosed
		poll.ClosedAt = &now
		poll.UpdatedAt = now
		// buildPollDTO uses the poll creator's identity for MyChoices/etc.
		// so the broadcast frame carries the creator's perspective. Each WS
		// client re-derives its own per-user view from the poll id if it
		// needs to.
		dto, err := s.buildPollDTO(ctx, poll, poll.CreatedBy)
		if err != nil {
			s.logger.Warn("vote: build dto",
				zap.String("poll_id", poll.ID.String()),
				zap.Error(err),
			)
			continue
		}
		s.broadcast(ctx, poll.TripID, BroadcastPollClosed, dto)
		closed++
	}
	return closed, nil
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

func (s *service) loadOpenPollForMember(ctx context.Context, userID, pollID uuid.UUID) (*Poll, error) {
	poll, err := s.repo.FindPoll(ctx, pollID)
	if err != nil {
		return nil, err
	}
	if poll == nil {
		return nil, ErrPollNotFound
	}
	if err := s.mustBeMember(ctx, poll.TripID, userID); err != nil {
		return nil, err
	}
	if poll.Status == StatusClosed {
		return nil, ErrPollClosed
	}
	return poll, nil
}

// buildPollDTO composes the response for a specific viewer. It applies:
//   - result_visibility (are aggregate counts visible to the caller?)
//   - is_anonymous       (should voter identities be scrubbed?)
func (s *service) buildPollDTO(ctx context.Context, poll *Poll, viewerID uuid.UUID) (*PollDTO, error) {
	options, err := s.repo.ListOptions(ctx, poll.ID)
	if err != nil {
		return nil, err
	}
	ballots, err := s.repo.ListBallots(ctx, poll.ID)
	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int, len(options))
	voters := make(map[uuid.UUID][]uuid.UUID, len(options))
	distinct := make(map[uuid.UUID]struct{}, len(ballots))
	myChoices := make([]uuid.UUID, 0)
	for _, b := range ballots {
		counts[b.OptionID]++
		voters[b.OptionID] = append(voters[b.OptionID], b.UserID)
		distinct[b.UserID] = struct{}{}
		if b.UserID == viewerID {
			myChoices = append(myChoices, b.OptionID)
		}
	}

	resultsVisible := computeResultsVisible(poll, len(myChoices) > 0)

	optDTOs := make([]OptionDTO, 0, len(options))
	for _, o := range options {
		od := OptionDTO{
			ID:        o.ID,
			Text:      o.Text,
			Meta:      o.Meta,
			AddedBy:   o.AddedBy,
			SortOrder: o.SortOrder,
			CreatedAt: o.CreatedAt,
		}
		if resultsVisible {
			od.Count = counts[o.ID]
			if !poll.IsAnonymous {
				od.Voters = voters[o.ID]
			}
		}
		optDTOs = append(optDTOs, od)
	}

	return &PollDTO{
		ID:               poll.ID,
		TripID:           poll.TripID,
		CreatedBy:        poll.CreatedBy,
		Type:             poll.Type,
		Title:            poll.Title,
		Description:      poll.Description,
		IsAnonymous:      poll.IsAnonymous,
		MaxChoices:       poll.MaxChoices,
		AllowOptionAdd:   poll.AllowOptionAdd,
		ResultVisibility: poll.ResultVisibility,
		DeadlineAt:       poll.DeadlineAt,
		Status:           poll.Status,
		ClosedAt:         poll.ClosedAt,
		CreatedAt:        poll.CreatedAt,
		UpdatedAt:        poll.UpdatedAt,
		Options:          optDTOs,
		MyChoices:        myChoices,
		ResultsVisible:   resultsVisible,
		TotalVoters:      len(distinct),
	}, nil
}

func computeResultsVisible(poll *Poll, viewerHasVoted bool) bool {
	switch poll.ResultVisibility {
	case VisibilityAfterVote:
		return viewerHasVoted || poll.Status == StatusClosed
	case VisibilityAfterDeadline:
		return poll.Status == StatusClosed
	default:
		return true
	}
}

func (s *service) broadcast(ctx context.Context, tripID uuid.UUID, kind BroadcastKind, dto *PollDTO) {
	if s.bcast == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Type   BroadcastKind `json:"type"`
		TripID uuid.UUID     `json:"trip_id"`
		Poll   *PollDTO      `json:"poll"`
	}{
		Type:   kind,
		TripID: tripID,
		Poll:   dto,
	})
	if err != nil {
		s.logger.Warn("vote: marshal broadcast", zap.Error(err))
		return
	}
	s.bcast.PublishTripEvent(ctx, tripID, payload)
}

func normalizeType(raw string) PollType {
	switch PollType(strings.ToLower(strings.TrimSpace(raw))) {
	case PollLocation:
		return PollLocation
	case PollTime:
		return PollTime
	default:
		return PollCustom
	}
}

func normalizeVisibility(raw string) ResultVisibility {
	switch ResultVisibility(strings.ToLower(strings.TrimSpace(raw))) {
	case VisibilityAfterVote:
		return VisibilityAfterVote
	case VisibilityAfterDeadline:
		return VisibilityAfterDeadline
	default:
		return VisibilityAlways
	}
}

func normalizeMeta(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}
