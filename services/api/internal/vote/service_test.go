package vote_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/vote"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	withTx func(context.Context, func(vote.IRepository) error) error

	createPoll        func(context.Context, *vote.Poll) error
	findPoll          func(context.Context, uuid.UUID) (*vote.Poll, error)
	listPollsByTrip   func(context.Context, uuid.UUID) ([]vote.Poll, error)
	markClosed        func(context.Context, uuid.UUID, time.Time) error
	claimExpiredPolls func(context.Context, time.Time, int) ([]vote.Poll, error)
	deletePoll        func(context.Context, uuid.UUID) error

	createOption func(context.Context, *vote.Option) error
	findOption   func(context.Context, uuid.UUID) (*vote.Option, error)
	listOptions  func(context.Context, uuid.UUID) ([]vote.Option, error)
	deleteOption func(context.Context, uuid.UUID) error

	replaceBallots       func(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error
	listBallots          func(context.Context, uuid.UUID) ([]vote.Ballot, error)
	listMyChoices        func(context.Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error)
	countVotersByOption  func(context.Context, uuid.UUID) (map[uuid.UUID]int, error)
	countDistinctVoters  func(context.Context, uuid.UUID) (int, error)
}

func (r *repoMock) WithTx(c context.Context, fn func(vote.IRepository) error) error {
	if r.withTx != nil {
		return r.withTx(c, fn)
	}
	return fn(r)
}

func (r *repoMock) CreatePoll(c context.Context, p *vote.Poll) error {
	if r.createPoll != nil {
		return r.createPoll(c, p)
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindPoll(c context.Context, id uuid.UUID) (*vote.Poll, error) {
	if r.findPoll != nil {
		return r.findPoll(c, id)
	}
	return nil, nil
}

func (r *repoMock) ListPollsByTrip(c context.Context, tid uuid.UUID) ([]vote.Poll, error) {
	if r.listPollsByTrip != nil {
		return r.listPollsByTrip(c, tid)
	}
	return nil, nil
}

func (r *repoMock) MarkClosed(c context.Context, id uuid.UUID, t time.Time) error {
	if r.markClosed != nil {
		return r.markClosed(c, id, t)
	}
	return nil
}

func (r *repoMock) ClaimExpiredPolls(c context.Context, now time.Time, limit int) ([]vote.Poll, error) {
	if r.claimExpiredPolls != nil {
		return r.claimExpiredPolls(c, now, limit)
	}
	return nil, nil
}

func (r *repoMock) DeletePoll(c context.Context, id uuid.UUID) error {
	if r.deletePoll != nil {
		return r.deletePoll(c, id)
	}
	return nil
}

func (r *repoMock) CreateOption(c context.Context, o *vote.Option) error {
	if r.createOption != nil {
		return r.createOption(c, o)
	}
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindOption(c context.Context, id uuid.UUID) (*vote.Option, error) {
	if r.findOption != nil {
		return r.findOption(c, id)
	}
	return nil, nil
}

func (r *repoMock) ListOptions(c context.Context, pid uuid.UUID) ([]vote.Option, error) {
	if r.listOptions != nil {
		return r.listOptions(c, pid)
	}
	return nil, nil
}

func (r *repoMock) DeleteOption(c context.Context, id uuid.UUID) error {
	if r.deleteOption != nil {
		return r.deleteOption(c, id)
	}
	return nil
}

func (r *repoMock) ReplaceBallots(c context.Context, pid, uid uuid.UUID, oids []uuid.UUID) error {
	if r.replaceBallots != nil {
		return r.replaceBallots(c, pid, uid, oids)
	}
	return nil
}

func (r *repoMock) ListBallots(c context.Context, pid uuid.UUID) ([]vote.Ballot, error) {
	if r.listBallots != nil {
		return r.listBallots(c, pid)
	}
	return nil, nil
}

func (r *repoMock) ListMyChoices(c context.Context, pid, uid uuid.UUID) ([]uuid.UUID, error) {
	if r.listMyChoices != nil {
		return r.listMyChoices(c, pid, uid)
	}
	return nil, nil
}

func (r *repoMock) CountVotersByOption(c context.Context, pid uuid.UUID) (map[uuid.UUID]int, error) {
	if r.countVotersByOption != nil {
		return r.countVotersByOption(c, pid)
	}
	return map[uuid.UUID]int{}, nil
}

func (r *repoMock) CountDistinctVoters(c context.Context, pid uuid.UUID) (int, error) {
	if r.countDistinctVoters != nil {
		return r.countDistinctVoters(c, pid)
	}
	return 0, nil
}

// ---- authMock ----

type authMock struct {
	isRoomMember func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

func (a *authMock) IsRoomMember(c context.Context, tid, uid uuid.UUID) (bool, error) {
	if a.isRoomMember != nil {
		return a.isRoomMember(c, tid, uid)
	}
	return true, nil
}

// ---- bcastMock ----

type bcastCall struct {
	tripID  uuid.UUID
	payload []byte
}

type bcastMock struct {
	calls []bcastCall
}

func (b *bcastMock) PublishTripEvent(_ context.Context, tid uuid.UUID, payload []byte) {
	b.calls = append(b.calls, bcastCall{tripID: tid, payload: payload})
}

// ---- Helpers ----

func newSvc(repo vote.IRepository, auth vote.TripAuthorizer, bc vote.Broadcaster) vote.IService {
	return vote.NewService(vote.ServiceConfig{
		Repo:        repo,
		TripAuth:    auth,
		Broadcaster: bc,
	})
}

func memberAuth() *authMock {
	return &authMock{isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil }}
}

func nonMemberAuth() *authMock {
	return &authMock{isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }}
}

// ---- CreatePoll ----

func TestCreatePoll_HappyPath_NormalizesAndBroadcasts(t *testing.T) {
	trip, user := uuid.New(), uuid.New()
	var savedPoll *vote.Poll
	var savedOptions []vote.Option

	repo := &repoMock{
		createPoll: func(_ context.Context, p *vote.Poll) error {
			p.ID = uuid.New()
			p.CreatedAt = time.Now()
			p.UpdatedAt = time.Now()
			savedPoll = p
			return nil
		},
		createOption: func(_ context.Context, o *vote.Option) error {
			o.ID = uuid.New()
			savedOptions = append(savedOptions, *o)
			return nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)

	future := time.Now().Add(2 * time.Hour)
	dto, err := svc.CreatePoll(ctx, user, trip, vote.CreatePollPayload{
		Type:             "  LOCATION ",
		Title:            "  Where to stay?  ",
		Description:      "  extra ",
		MaxChoices:       0,
		ResultVisibility: "AFTER_VOTE",
		DeadlineAt:       &future,
		Options: []vote.OptionInput{
			{Text: "  Hotel A"},
			{Text: ""}, // skipped
			{Text: "Hotel B"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, savedPoll)
	assert.Equal(t, vote.PollLocation, savedPoll.Type)
	assert.Equal(t, "Where to stay?", savedPoll.Title)
	assert.Equal(t, "extra", savedPoll.Description)
	assert.Equal(t, vote.VisibilityAfterVote, savedPoll.ResultVisibility)
	assert.Equal(t, vote.StatusOpen, savedPoll.Status)
	assert.Equal(t, 1, savedPoll.MaxChoices, "MaxChoices=0 must be lifted to 1")
	assert.Equal(t, user, savedPoll.CreatedBy)
	assert.Equal(t, trip, savedPoll.TripID)

	require.Len(t, savedOptions, 2, "empty option text must be skipped")
	assert.Equal(t, "Hotel A", savedOptions[0].Text)
	assert.Equal(t, "Hotel B", savedOptions[1].Text)
	assert.Equal(t, 0, savedOptions[0].SortOrder)
	assert.Equal(t, 2, savedOptions[1].SortOrder, "sort order is the input index, not the compacted index")

	require.Len(t, bc.calls, 1)
	assert.Equal(t, trip, bc.calls[0].tripID)
	var frame struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(bc.calls[0].payload, &frame))
	assert.Equal(t, string(vote.BroadcastPollCreated), frame.Type)

	assert.Equal(t, vote.StatusOpen, dto.Status)
	assert.False(t, dto.ResultsVisible, "after_vote + no ballot must hide results")
}

func TestCreatePoll_UnknownTypeDefaultsToCustom(t *testing.T) {
	trip, user := uuid.New(), uuid.New()
	var saved *vote.Poll
	repo := &repoMock{
		createPoll: func(_ context.Context, p *vote.Poll) error {
			p.ID = uuid.New()
			saved = p
			return nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CreatePoll(ctx, user, trip, vote.CreatePollPayload{
		Type: "bogus", Title: "Q", MaxChoices: 3,
		Options: []vote.OptionInput{{Text: "a"}, {Text: "b"}},
	})
	require.NoError(t, err)
	assert.Equal(t, vote.PollCustom, saved.Type)
	assert.Equal(t, 3, saved.MaxChoices)
}

func TestCreatePoll_EmptyTitle_Rejected(t *testing.T) {
	svc := newSvc(&repoMock{}, memberAuth(), nil)
	_, err := svc.CreatePoll(ctx, uuid.New(), uuid.New(), vote.CreatePollPayload{Title: "   "})
	assert.ErrorIs(t, err, vote.ErrInvalidPayload)
}

func TestCreatePoll_PastDeadline_Rejected(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	svc := newSvc(&repoMock{}, memberAuth(), nil)
	_, err := svc.CreatePoll(ctx, uuid.New(), uuid.New(), vote.CreatePollPayload{
		Title: "Q", DeadlineAt: &past,
	})
	assert.ErrorIs(t, err, vote.ErrInvalidPayload)
}

func TestCreatePoll_TooManyOptions_Rejected(t *testing.T) {
	svc := newSvc(&repoMock{}, memberAuth(), nil)
	opts := make([]vote.OptionInput, 51)
	for i := range opts {
		opts[i] = vote.OptionInput{Text: "x"}
	}
	_, err := svc.CreatePoll(ctx, uuid.New(), uuid.New(), vote.CreatePollPayload{
		Title: "Q", Options: opts,
	})
	assert.ErrorIs(t, err, vote.ErrInvalidPayload)
}

func TestCreatePoll_NonMember_Forbidden(t *testing.T) {
	svc := newSvc(&repoMock{}, nonMemberAuth(), nil)
	_, err := svc.CreatePoll(ctx, uuid.New(), uuid.New(), vote.CreatePollPayload{Title: "Q"})
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

func TestCreatePoll_NoAuthConfigured_Forbidden(t *testing.T) {
	svc := newSvc(&repoMock{}, nil, nil)
	_, err := svc.CreatePoll(ctx, uuid.New(), uuid.New(), vote.CreatePollPayload{Title: "Q"})
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

func TestCreatePoll_TxError_Propagates(t *testing.T) {
	boom := errors.New("boom")
	repo := &repoMock{
		createPoll: func(_ context.Context, _ *vote.Poll) error { return boom },
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CreatePoll(ctx, uuid.New(), uuid.New(), vote.CreatePollPayload{
		Title:   "Q",
		Options: []vote.OptionInput{{Text: "a"}},
	})
	assert.ErrorIs(t, err, boom)
}

// ---- ListPolls / GetPoll ----

func TestListPolls_NonMember_Forbidden(t *testing.T) {
	svc := newSvc(&repoMock{}, nonMemberAuth(), nil)
	_, err := svc.ListPolls(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

func TestListPolls_BuildsDTOsForEach(t *testing.T) {
	trip, user := uuid.New(), uuid.New()
	p1, p2 := uuid.New(), uuid.New()
	repo := &repoMock{
		listPollsByTrip: func(_ context.Context, _ uuid.UUID) ([]vote.Poll, error) {
			return []vote.Poll{
				{ID: p1, TripID: trip, CreatedBy: user, Status: vote.StatusOpen, ResultVisibility: vote.VisibilityAlways},
				{ID: p2, TripID: trip, CreatedBy: user, Status: vote.StatusClosed, ResultVisibility: vote.VisibilityAlways},
			}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	out, err := svc.ListPolls(ctx, user, trip)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, p1, out[0].ID)
	assert.Equal(t, p2, out[1].ID)
}

func TestGetPoll_NotFound(t *testing.T) {
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) { return nil, nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.GetPoll(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, vote.ErrPollNotFound)
}

func TestGetPoll_NonMember_Forbidden(t *testing.T) {
	trip := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, id uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: id, TripID: trip, Status: vote.StatusOpen}, nil
		},
	}
	svc := newSvc(repo, nonMemberAuth(), nil)
	_, err := svc.GetPoll(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

// ---- AddOption ----

func TestAddOption_NonCreatorWithoutAllowFlag_Forbidden(t *testing.T) {
	creator, other := uuid.New(), uuid.New()
	pollID, trip := uuid.New(), uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pollID, TripID: trip, CreatedBy: creator,
				Status: vote.StatusOpen, AllowOptionAdd: false}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.AddOption(ctx, other, pollID, vote.AddOptionPayload{Text: "x"})
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

func TestAddOption_NonCreatorWithAllowFlag_Succeeds(t *testing.T) {
	creator, other := uuid.New(), uuid.New()
	pollID, trip := uuid.New(), uuid.New()
	var savedOpt *vote.Option
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pollID, TripID: trip, CreatedBy: creator,
				Status: vote.StatusOpen, AllowOptionAdd: true,
				ResultVisibility: vote.VisibilityAlways}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) {
			return []vote.Option{{ID: uuid.New(), Text: "existing"}}, nil
		},
		createOption: func(_ context.Context, o *vote.Option) error {
			o.ID = uuid.New()
			savedOpt = o
			return nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	_, err := svc.AddOption(ctx, other, pollID, vote.AddOptionPayload{Text: "  New "})
	require.NoError(t, err)
	require.NotNil(t, savedOpt)
	assert.Equal(t, "New", savedOpt.Text)
	assert.Equal(t, other, savedOpt.AddedBy)
	assert.Equal(t, 1, savedOpt.SortOrder, "SortOrder is next slot after existing options")
	require.Len(t, bc.calls, 1)
}

func TestAddOption_ClosedPoll_Rejected(t *testing.T) {
	pollID, trip := uuid.New(), uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pollID, TripID: trip, Status: vote.StatusClosed}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.AddOption(ctx, uuid.New(), pollID, vote.AddOptionPayload{Text: "x"})
	assert.ErrorIs(t, err, vote.ErrPollClosed)
}

func TestAddOption_EmptyText_Rejected(t *testing.T) {
	creator := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: uuid.New(), TripID: uuid.New(), CreatedBy: creator, Status: vote.StatusOpen}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.AddOption(ctx, creator, uuid.New(), vote.AddOptionPayload{Text: "   "})
	assert.ErrorIs(t, err, vote.ErrInvalidPayload)
}

func TestAddOption_HitsPollCap_Rejected(t *testing.T) {
	creator := uuid.New()
	full := make([]vote.Option, 50)
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: uuid.New(), TripID: uuid.New(), CreatedBy: creator, Status: vote.StatusOpen}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) { return full, nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.AddOption(ctx, creator, uuid.New(), vote.AddOptionPayload{Text: "x"})
	assert.ErrorIs(t, err, vote.ErrInvalidPayload)
}

// ---- DeleteOption ----

func TestDeleteOption_NotFound(t *testing.T) {
	repo := &repoMock{
		findOption: func(_ context.Context, _ uuid.UUID) (*vote.Option, error) { return nil, nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.DeleteOption(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, vote.ErrOptionNotFound)
}

func TestDeleteOption_OtherMember_CannotDeleteSomeoneElsesOption(t *testing.T) {
	creator, adder, other := uuid.New(), uuid.New(), uuid.New()
	pollID, optID, trip := uuid.New(), uuid.New(), uuid.New()
	repo := &repoMock{
		findOption: func(_ context.Context, _ uuid.UUID) (*vote.Option, error) {
			return &vote.Option{ID: optID, PollID: pollID, AddedBy: adder}, nil
		},
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pollID, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.DeleteOption(ctx, other, optID)
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

func TestDeleteOption_OwnOption_Succeeds(t *testing.T) {
	creator, adder := uuid.New(), uuid.New()
	pollID, optID, trip := uuid.New(), uuid.New(), uuid.New()
	deleted := false
	repo := &repoMock{
		findOption: func(_ context.Context, _ uuid.UUID) (*vote.Option, error) {
			return &vote.Option{ID: optID, PollID: pollID, AddedBy: adder}, nil
		},
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pollID, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen,
				ResultVisibility: vote.VisibilityAlways}, nil
		},
		deleteOption: func(_ context.Context, _ uuid.UUID) error { deleted = true; return nil },
	}
	svc := newSvc(repo, memberAuth(), &bcastMock{})
	_, err := svc.DeleteOption(ctx, adder, optID)
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestDeleteOption_CreatorCanPruneAny(t *testing.T) {
	creator, adder := uuid.New(), uuid.New()
	pollID, optID, trip := uuid.New(), uuid.New(), uuid.New()
	deleted := false
	repo := &repoMock{
		findOption: func(_ context.Context, _ uuid.UUID) (*vote.Option, error) {
			return &vote.Option{ID: optID, PollID: pollID, AddedBy: adder}, nil
		},
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pollID, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen}, nil
		},
		deleteOption: func(_ context.Context, _ uuid.UUID) error { deleted = true; return nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.DeleteOption(ctx, creator, optID)
	require.NoError(t, err)
	assert.True(t, deleted)
}

// ---- CastVote ----

func TestCastVote_DeadlinePassed_Rejected(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: uuid.New(), TripID: uuid.New(), Status: vote.StatusOpen,
				MaxChoices: 1, DeadlineAt: &past}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CastVote(ctx, uuid.New(), uuid.New(), vote.CastVotePayload{OptionIDs: []uuid.UUID{uuid.New()}})
	assert.ErrorIs(t, err, vote.ErrPollClosed)
}

func TestCastVote_EmptySelection_Rejected(t *testing.T) {
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: uuid.New(), TripID: uuid.New(), Status: vote.StatusOpen, MaxChoices: 3}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CastVote(ctx, uuid.New(), uuid.New(), vote.CastVotePayload{OptionIDs: nil})
	assert.ErrorIs(t, err, vote.ErrInvalidPayload)
}

func TestCastVote_ExceedsMaxChoices_Rejected(t *testing.T) {
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: uuid.New(), TripID: uuid.New(), Status: vote.StatusOpen, MaxChoices: 1}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CastVote(ctx, uuid.New(), uuid.New(),
		vote.CastVotePayload{OptionIDs: []uuid.UUID{uuid.New(), uuid.New()}})
	assert.ErrorIs(t, err, vote.ErrTooManyChoices)
}

func TestCastVote_UnknownOptionID_Rejected(t *testing.T) {
	real := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: uuid.New(), TripID: uuid.New(), Status: vote.StatusOpen, MaxChoices: 3}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) {
			return []vote.Option{{ID: real}}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CastVote(ctx, uuid.New(), uuid.New(),
		vote.CastVotePayload{OptionIDs: []uuid.UUID{uuid.New()}})
	assert.ErrorIs(t, err, vote.ErrOptionNotFound)
}

func TestCastVote_DuplicateOptionIDs_Rejected(t *testing.T) {
	oid := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: uuid.New(), TripID: uuid.New(), Status: vote.StatusOpen, MaxChoices: 3}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) {
			return []vote.Option{{ID: oid}}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CastVote(ctx, uuid.New(), uuid.New(),
		vote.CastVotePayload{OptionIDs: []uuid.UUID{oid, oid}})
	assert.ErrorIs(t, err, vote.ErrInvalidPayload)
}

func TestCastVote_Succeeds_ReplacesBallotsAndBroadcasts(t *testing.T) {
	user, oid, trip := uuid.New(), uuid.New(), uuid.New()
	var replaced struct {
		pollID, userID uuid.UUID
		oids           []uuid.UUID
	}
	repo := &repoMock{
		findPoll: func(_ context.Context, id uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: id, TripID: trip, Status: vote.StatusOpen,
				MaxChoices: 3, ResultVisibility: vote.VisibilityAlways}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) {
			return []vote.Option{{ID: oid}}, nil
		},
		replaceBallots: func(_ context.Context, pid, uid uuid.UUID, oids []uuid.UUID) error {
			replaced.pollID, replaced.userID, replaced.oids = pid, uid, oids
			return nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	pid := uuid.New()
	_, err := svc.CastVote(ctx, user, pid, vote.CastVotePayload{OptionIDs: []uuid.UUID{oid}})
	require.NoError(t, err)
	assert.Equal(t, pid, replaced.pollID)
	assert.Equal(t, user, replaced.userID)
	assert.Equal(t, []uuid.UUID{oid}, replaced.oids)
	require.Len(t, bc.calls, 1)
	var frame struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(bc.calls[0].payload, &frame))
	assert.Equal(t, string(vote.BroadcastVoteCast), frame.Type)
}

// ---- ClosePoll ----

func TestClosePoll_NotFound(t *testing.T) {
	repo := &repoMock{findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) { return nil, nil }}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.ClosePoll(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, vote.ErrPollNotFound)
}

func TestClosePoll_NotCreator_Forbidden(t *testing.T) {
	creator := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, id uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: id, TripID: uuid.New(), CreatedBy: creator, Status: vote.StatusOpen}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.ClosePoll(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

func TestClosePoll_AlreadyClosed_IsIdempotent(t *testing.T) {
	creator := uuid.New()
	marked := false
	repo := &repoMock{
		findPoll: func(_ context.Context, id uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: id, TripID: uuid.New(), CreatedBy: creator, Status: vote.StatusClosed}, nil
		},
		markClosed: func(_ context.Context, _ uuid.UUID, _ time.Time) error { marked = true; return nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	dto, err := svc.ClosePoll(ctx, creator, uuid.New())
	require.NoError(t, err)
	assert.False(t, marked, "already-closed poll must not be re-marked")
	assert.Equal(t, vote.StatusClosed, dto.Status)
}

func TestClosePoll_ByCreator_Succeeds(t *testing.T) {
	creator, trip := uuid.New(), uuid.New()
	marked := false
	repo := &repoMock{
		findPoll: func(_ context.Context, id uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: id, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen,
				ResultVisibility: vote.VisibilityAlways}, nil
		},
		markClosed: func(_ context.Context, _ uuid.UUID, _ time.Time) error { marked = true; return nil },
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	dto, err := svc.ClosePoll(ctx, creator, uuid.New())
	require.NoError(t, err)
	assert.True(t, marked)
	assert.Equal(t, vote.StatusClosed, dto.Status)
	require.Len(t, bc.calls, 1)
}

// ---- DeletePoll ----

func TestDeletePoll_NotCreator_Forbidden(t *testing.T) {
	creator := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, id uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: id, TripID: uuid.New(), CreatedBy: creator}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	err := svc.DeletePoll(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, vote.ErrForbidden)
}

func TestDeletePoll_ByCreator_Succeeds_BroadcastsDeletePayload(t *testing.T) {
	creator, pid, trip := uuid.New(), uuid.New(), uuid.New()
	deleted := false
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pid, TripID: trip, CreatedBy: creator}, nil
		},
		deletePoll: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, pid, id)
			deleted = true
			return nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	require.NoError(t, svc.DeletePoll(ctx, creator, pid))
	assert.True(t, deleted)
	require.Len(t, bc.calls, 1)
	var frame struct {
		Type   string    `json:"type"`
		TripID uuid.UUID `json:"trip_id"`
		PollID uuid.UUID `json:"poll_id"`
	}
	require.NoError(t, json.Unmarshal(bc.calls[0].payload, &frame))
	assert.Equal(t, string(vote.BroadcastPollDeleted), frame.Type)
	assert.Equal(t, trip, frame.TripID)
	assert.Equal(t, pid, frame.PollID)
}

// ---- SweepExpiredPolls ----

func TestSweepExpiredPolls_ClosesEachAndBroadcasts(t *testing.T) {
	trip := uuid.New()
	creator := uuid.New()
	p1, p2 := uuid.New(), uuid.New()
	claimed := []vote.Poll{
		{ID: p1, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen,
			ResultVisibility: vote.VisibilityAlways},
		{ID: p2, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen,
			ResultVisibility: vote.VisibilityAlways},
	}
	marks := []uuid.UUID{}
	repo := &repoMock{
		claimExpiredPolls: func(_ context.Context, _ time.Time, _ int) ([]vote.Poll, error) {
			return claimed, nil
		},
		markClosed: func(_ context.Context, id uuid.UUID, _ time.Time) error {
			marks = append(marks, id)
			return nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	n, err := svc.SweepExpiredPolls(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.ElementsMatch(t, []uuid.UUID{p1, p2}, marks)
	assert.Len(t, bc.calls, 2)
}

func TestSweepExpiredPolls_ContinuesOnMarkFailure(t *testing.T) {
	trip, creator := uuid.New(), uuid.New()
	p1, p2 := uuid.New(), uuid.New()
	claimed := []vote.Poll{
		{ID: p1, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen, ResultVisibility: vote.VisibilityAlways},
		{ID: p2, TripID: trip, CreatedBy: creator, Status: vote.StatusOpen, ResultVisibility: vote.VisibilityAlways},
	}
	repo := &repoMock{
		claimExpiredPolls: func(_ context.Context, _ time.Time, _ int) ([]vote.Poll, error) {
			return claimed, nil
		},
		markClosed: func(_ context.Context, id uuid.UUID, _ time.Time) error {
			if id == p1 {
				return errors.New("boom")
			}
			return nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	n, err := svc.SweepExpiredPolls(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "one closed successfully, one skipped due to mark failure")
	require.Len(t, bc.calls, 1)
}

func TestSweepExpiredPolls_ClaimError_Propagates(t *testing.T) {
	boom := errors.New("boom")
	repo := &repoMock{
		claimExpiredPolls: func(_ context.Context, _ time.Time, _ int) ([]vote.Poll, error) {
			return nil, boom
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	n, err := svc.SweepExpiredPolls(ctx)
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, 0, n)
}

// ---- computeResultsVisible / DTO behavior through public API ----

func TestGetPoll_AfterVoteVisibility_HiddenWhenNoBallot(t *testing.T) {
	me := uuid.New()
	pid, trip := uuid.New(), uuid.New()
	oid := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pid, TripID: trip, Status: vote.StatusOpen,
				ResultVisibility: vote.VisibilityAfterVote}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) {
			return []vote.Option{{ID: oid, Text: "A"}}, nil
		},
		listBallots: func(_ context.Context, _ uuid.UUID) ([]vote.Ballot, error) {
			return []vote.Ballot{{PollID: pid, UserID: uuid.New(), OptionID: oid}}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	dto, err := svc.GetPoll(ctx, me, pid)
	require.NoError(t, err)
	assert.False(t, dto.ResultsVisible, "viewer without a ballot must not see counts")
	require.Len(t, dto.Options, 1)
	assert.Equal(t, 0, dto.Options[0].Count, "hidden count must be zero")
	assert.Nil(t, dto.Options[0].Voters)
}

func TestGetPoll_AfterVoteVisibility_ShownAfterVoterCasts(t *testing.T) {
	me := uuid.New()
	pid, trip := uuid.New(), uuid.New()
	oid := uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pid, TripID: trip, Status: vote.StatusOpen,
				ResultVisibility: vote.VisibilityAfterVote}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) {
			return []vote.Option{{ID: oid, Text: "A"}}, nil
		},
		listBallots: func(_ context.Context, _ uuid.UUID) ([]vote.Ballot, error) {
			return []vote.Ballot{{PollID: pid, UserID: me, OptionID: oid}}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	dto, err := svc.GetPoll(ctx, me, pid)
	require.NoError(t, err)
	assert.True(t, dto.ResultsVisible)
	assert.Equal(t, 1, dto.Options[0].Count)
	assert.Contains(t, dto.MyChoices, oid)
	assert.Equal(t, 1, dto.TotalVoters)
}

func TestGetPoll_AnonymousPoll_HidesVoters(t *testing.T) {
	me := uuid.New()
	pid, trip, oid := uuid.New(), uuid.New(), uuid.New()
	repo := &repoMock{
		findPoll: func(_ context.Context, _ uuid.UUID) (*vote.Poll, error) {
			return &vote.Poll{ID: pid, TripID: trip, Status: vote.StatusOpen,
				ResultVisibility: vote.VisibilityAlways, IsAnonymous: true}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]vote.Option, error) {
			return []vote.Option{{ID: oid, Text: "A"}}, nil
		},
		listBallots: func(_ context.Context, _ uuid.UUID) ([]vote.Ballot, error) {
			return []vote.Ballot{{PollID: pid, UserID: uuid.New(), OptionID: oid}}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	dto, err := svc.GetPoll(ctx, me, pid)
	require.NoError(t, err)
	assert.True(t, dto.ResultsVisible)
	assert.Equal(t, 1, dto.Options[0].Count, "anon poll still shows count")
	assert.Nil(t, dto.Options[0].Voters, "anon poll must scrub voter list")
}
