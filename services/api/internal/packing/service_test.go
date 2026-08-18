package packing_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Ans1110/trip-app/internal/packing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	withTx func(context.Context, func(packing.IRepository) error) error

	createItem       func(context.Context, *packing.Item) error
	findItem         func(context.Context, uuid.UUID) (*packing.Item, error)
	listItemsByTrip  func(context.Context, uuid.UUID) ([]packing.Item, error)
	updateItem       func(context.Context, uuid.UUID, map[string]any) (*packing.Item, error)
	deleteItem       func(context.Context, uuid.UUID) error
	nextSortOrder    func(context.Context, uuid.UUID) (int, error)
	setPacked        func(context.Context, uuid.UUID, uuid.UUID, bool) error
	packInfoForItems func(context.Context, []uuid.UUID, uuid.UUID) (map[uuid.UUID]packing.ItemPackInfo, error)
}

func (r *repoMock) WithTx(c context.Context, fn func(packing.IRepository) error) error {
	if r.withTx != nil {
		return r.withTx(c, fn)
	}
	return fn(r)
}

func (r *repoMock) CreateItem(c context.Context, i *packing.Item) error {
	if r.createItem != nil {
		return r.createItem(c, i)
	}
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindItem(c context.Context, id uuid.UUID) (*packing.Item, error) {
	if r.findItem != nil {
		return r.findItem(c, id)
	}
	return nil, nil
}

func (r *repoMock) ListItemsByTrip(c context.Context, tid uuid.UUID) ([]packing.Item, error) {
	if r.listItemsByTrip != nil {
		return r.listItemsByTrip(c, tid)
	}
	return nil, nil
}

func (r *repoMock) UpdateItem(c context.Context, id uuid.UUID, patch map[string]any) (*packing.Item, error) {
	if r.updateItem != nil {
		return r.updateItem(c, id, patch)
	}
	return nil, nil
}

func (r *repoMock) DeleteItem(c context.Context, id uuid.UUID) error {
	if r.deleteItem != nil {
		return r.deleteItem(c, id)
	}
	return nil
}

func (r *repoMock) NextSortOrder(c context.Context, tid uuid.UUID) (int, error) {
	if r.nextSortOrder != nil {
		return r.nextSortOrder(c, tid)
	}
	return 0, nil
}

func (r *repoMock) SetPacked(c context.Context, itemID, userID uuid.UUID, packed bool) error {
	if r.setPacked != nil {
		return r.setPacked(c, itemID, userID, packed)
	}
	return nil
}

func (r *repoMock) PackInfoForItems(c context.Context, ids []uuid.UUID, uid uuid.UUID) (map[uuid.UUID]packing.ItemPackInfo, error) {
	if r.packInfoForItems != nil {
		return r.packInfoForItems(c, ids, uid)
	}
	return map[uuid.UUID]packing.ItemPackInfo{}, nil
}

// ---- authMock ----

type authMock struct {
	isRoomMember func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	tripOwnerID  func(context.Context, uuid.UUID) (uuid.UUID, error)
}

func (a *authMock) IsRoomMember(c context.Context, tid, uid uuid.UUID) (bool, error) {
	if a.isRoomMember != nil {
		return a.isRoomMember(c, tid, uid)
	}
	return true, nil
}

func (a *authMock) TripOwnerID(c context.Context, tid uuid.UUID) (uuid.UUID, error) {
	if a.tripOwnerID != nil {
		return a.tripOwnerID(c, tid)
	}
	return uuid.Nil, nil
}

func memberAuth() *authMock                          { return &authMock{} }
func nonMemberAuth() *authMock {
	return &authMock{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
	}
}

// authWithOwner returns an auth stub that accepts membership for everyone and
// reports ownerID as the trip owner. Used to test the "trip owner can moderate"
// branch in loadItemForEditor.
func authWithOwner(ownerID uuid.UUID) *authMock {
	return &authMock{
		tripOwnerID: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) { return ownerID, nil },
	}
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

// ---- helpers ----

func newSvc(repo packing.IRepository, auth packing.TripAuthorizer, bc packing.Broadcaster) packing.IService {
	return packing.NewService(packing.ServiceConfig{
		Repo:        repo,
		TripAuth:    auth,
		Broadcaster: bc,
	})
}

func decodeItemFrame(t *testing.T, payload []byte) (kind string, item *packing.ItemDTO) {
	t.Helper()
	var frame struct {
		Type string            `json:"type"`
		Item *packing.ItemDTO  `json:"item"`
	}
	require.NoError(t, json.Unmarshal(payload, &frame))
	return frame.Type, frame.Item
}

// ---- CreateItem ----

func TestCreateItem_HappyPath_NormalizesAndBroadcasts(t *testing.T) {
	tripID, userID := uuid.New(), uuid.New()
	var saved *packing.Item
	repo := &repoMock{
		createItem: func(_ context.Context, i *packing.Item) error {
			i.ID = uuid.New()
			saved = i
			return nil
		},
		nextSortOrder: func(_ context.Context, _ uuid.UUID) (int, error) { return 7, nil },
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)

	dto, err := svc.CreateItem(ctx, userID, tripID, packing.CreateItemPayload{
		Name:     "  Sunscreen  ",
		Quantity: 0, // must be lifted to 1
		Category: "  Toiletries  ",
		Note:     "SPF 50",
	})
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "Sunscreen", saved.Name)
	assert.Equal(t, 1, saved.Quantity, "Quantity<1 must default to 1")
	assert.Equal(t, "Toiletries", saved.Category)
	assert.Equal(t, "SPF 50", saved.Note)
	assert.Equal(t, 7, saved.SortOrder, "SortOrder=0 must fetch NextSortOrder")
	assert.Equal(t, userID, saved.CreatedBy)
	assert.Equal(t, tripID, saved.TripID)

	require.Len(t, bc.calls, 1)
	assert.Equal(t, tripID, bc.calls[0].tripID)
	kind, item := decodeItemFrame(t, bc.calls[0].payload)
	assert.Equal(t, string(packing.BroadcastItemCreated), kind)
	require.NotNil(t, item)
	assert.Equal(t, saved.ID, item.ID)
	assert.False(t, dto.PackedByMe, "newly created item is not packed by the creator")
	assert.Equal(t, 0, dto.PackedCount)
}

func TestCreateItem_PreservesExplicitSortOrder(t *testing.T) {
	var saved *packing.Item
	repo := &repoMock{
		createItem: func(_ context.Context, i *packing.Item) error {
			i.ID = uuid.New()
			saved = i
			return nil
		},
		nextSortOrder: func(_ context.Context, _ uuid.UUID) (int, error) {
			t.Fatal("NextSortOrder must not be called when SortOrder is set")
			return 0, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CreateItem(ctx, uuid.New(), uuid.New(), packing.CreateItemPayload{
		Name: "Toothbrush", SortOrder: 42,
	})
	require.NoError(t, err)
	assert.Equal(t, 42, saved.SortOrder)
}

func TestCreateItem_EmptyName_Rejected(t *testing.T) {
	repo := &repoMock{
		createItem: func(_ context.Context, _ *packing.Item) error {
			t.Fatal("CreateItem must not be called for empty name")
			return nil
		},
	}
	svc := newSvc(repo, memberAuth(), &bcastMock{})
	_, err := svc.CreateItem(ctx, uuid.New(), uuid.New(), packing.CreateItemPayload{Name: "   "})
	assert.ErrorIs(t, err, packing.ErrInvalidPayload)
}

func TestCreateItem_NonMember_Forbidden(t *testing.T) {
	svc := newSvc(&repoMock{}, nonMemberAuth(), &bcastMock{})
	_, err := svc.CreateItem(ctx, uuid.New(), uuid.New(), packing.CreateItemPayload{Name: "x"})
	assert.ErrorIs(t, err, packing.ErrForbidden)
}

func TestCreateItem_LongName_Truncated(t *testing.T) {
	longName := strings.Repeat("a", 500)
	var saved *packing.Item
	repo := &repoMock{
		createItem: func(_ context.Context, i *packing.Item) error {
			i.ID = uuid.New()
			saved = i
			return nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CreateItem(ctx, uuid.New(), uuid.New(), packing.CreateItemPayload{Name: longName})
	require.NoError(t, err)
	assert.Len(t, saved.Name, 200, "name must be truncated to maxNameLen")
}

func TestCreateItem_LongNote_Truncated(t *testing.T) {
	longNote := strings.Repeat("b", 3000)
	var saved *packing.Item
	repo := &repoMock{
		createItem: func(_ context.Context, i *packing.Item) error {
			i.ID = uuid.New()
			saved = i
			return nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.CreateItem(ctx, uuid.New(), uuid.New(), packing.CreateItemPayload{Name: "n", Note: longNote})
	require.NoError(t, err)
	assert.Len(t, saved.Note, 2000, "note must be truncated to maxNoteLen")
}

// ---- ListItems ----

func TestListItems_NonMember_Forbidden(t *testing.T) {
	svc := newSvc(&repoMock{}, nonMemberAuth(), nil)
	_, err := svc.ListItems(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, packing.ErrForbidden)
}

func TestListItems_ReturnsRowsWithPackInfo(t *testing.T) {
	trip, user := uuid.New(), uuid.New()
	i1 := packing.Item{ID: uuid.New(), TripID: trip, Name: "a"}
	i2 := packing.Item{ID: uuid.New(), TripID: trip, Name: "b"}
	repo := &repoMock{
		listItemsByTrip: func(_ context.Context, tid uuid.UUID) ([]packing.Item, error) {
			assert.Equal(t, trip, tid)
			return []packing.Item{i1, i2}, nil
		},
		packInfoForItems: func(_ context.Context, ids []uuid.UUID, uid uuid.UUID) (map[uuid.UUID]packing.ItemPackInfo, error) {
			assert.Equal(t, user, uid)
			assert.ElementsMatch(t, []uuid.UUID{i1.ID, i2.ID}, ids)
			return map[uuid.UUID]packing.ItemPackInfo{
				i1.ID: {PackedByMe: true, PackedCount: 2},
			}, nil
		},
	}
	svc := newSvc(repo, memberAuth(), nil)
	out, err := svc.ListItems(ctx, user, trip)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.True(t, out[0].PackedByMe)
	assert.Equal(t, 2, out[0].PackedCount)
	assert.False(t, out[1].PackedByMe, "items without pack info default to unpacked")
	assert.Equal(t, 0, out[1].PackedCount)
}

// ---- UpdateItem: authorization ----

func TestUpdateItem_Creator_Allowed(t *testing.T) {
	itemID, tripID, user := uuid.New(), uuid.New(), uuid.New()
	original := &packing.Item{ID: itemID, TripID: tripID, CreatedBy: user, Name: "old", Quantity: 1}
	updated := &packing.Item{ID: itemID, TripID: tripID, CreatedBy: user, Name: "new", Quantity: 1}
	repo := &repoMock{
		findItem:   func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return original, nil },
		updateItem: func(_ context.Context, _ uuid.UUID, _ map[string]any) (*packing.Item, error) { return updated, nil },
	}
	name := "new"
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	dto, err := svc.UpdateItem(ctx, user, itemID, packing.UpdateItemPayload{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "new", dto.Name)
	require.Len(t, bc.calls, 1)
	kind, _ := decodeItemFrame(t, bc.calls[0].payload)
	assert.Equal(t, string(packing.BroadcastItemUpdated), kind)
}

func TestUpdateItem_TripOwner_Allowed(t *testing.T) {
	itemID, tripID, creator, owner := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	original := &packing.Item{ID: itemID, TripID: tripID, CreatedBy: creator, Name: "old", Quantity: 1}
	updated := &packing.Item{ID: itemID, TripID: tripID, CreatedBy: creator, Name: "renamed", Quantity: 1}
	repo := &repoMock{
		findItem:   func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return original, nil },
		updateItem: func(_ context.Context, _ uuid.UUID, _ map[string]any) (*packing.Item, error) { return updated, nil },
	}
	name := "renamed"
	svc := newSvc(repo, authWithOwner(owner), nil)
	dto, err := svc.UpdateItem(ctx, owner, itemID, packing.UpdateItemPayload{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "renamed", dto.Name)
}

func TestUpdateItem_OtherMember_Forbidden(t *testing.T) {
	itemID, creator, owner, stranger := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	original := &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: creator}
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return original, nil },
		updateItem: func(_ context.Context, _ uuid.UUID, _ map[string]any) (*packing.Item, error) {
			t.Fatal("UpdateItem must not run for non-creator, non-owner")
			return nil, nil
		},
	}
	name := "x"
	svc := newSvc(repo, authWithOwner(owner), nil)
	_, err := svc.UpdateItem(ctx, stranger, itemID, packing.UpdateItemPayload{Name: &name})
	assert.ErrorIs(t, err, packing.ErrForbidden)
}

func TestUpdateItem_NonMember_Forbidden(t *testing.T) {
	itemID, user := uuid.New(), uuid.New()
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: user}, nil
		},
	}
	svc := newSvc(repo, nonMemberAuth(), nil)
	_, err := svc.UpdateItem(ctx, user, itemID, packing.UpdateItemPayload{})
	assert.ErrorIs(t, err, packing.ErrForbidden)
}

func TestUpdateItem_NotFound(t *testing.T) {
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return nil, nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.UpdateItem(ctx, uuid.New(), uuid.New(), packing.UpdateItemPayload{})
	assert.ErrorIs(t, err, packing.ErrItemNotFound)
}

// ---- UpdateItem: patch shape ----

func TestUpdateItem_EmptyPatch_NoBroadcast(t *testing.T) {
	itemID, user := uuid.New(), uuid.New()
	original := &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: user, Name: "same", Quantity: 1}
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return original, nil },
		updateItem: func(_ context.Context, _ uuid.UUID, _ map[string]any) (*packing.Item, error) {
			t.Fatal("UpdateItem must not be called when patch is empty")
			return nil, nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	dto, err := svc.UpdateItem(ctx, user, itemID, packing.UpdateItemPayload{})
	require.NoError(t, err)
	assert.Equal(t, "same", dto.Name)
	assert.Empty(t, bc.calls, "no fields changed => no broadcast")
}

func TestUpdateItem_AllFields(t *testing.T) {
	itemID, user := uuid.New(), uuid.New()
	original := &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: user, Name: "old"}
	var gotPatch map[string]any
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return original, nil },
		updateItem: func(_ context.Context, _ uuid.UUID, patch map[string]any) (*packing.Item, error) {
			gotPatch = patch
			return &packing.Item{ID: itemID, TripID: original.TripID, CreatedBy: user, Name: "new"}, nil
		},
	}
	name := "  new  "
	qty := 5
	cat := "  Gear  "
	note := "keep dry"
	sort := 3
	svc := newSvc(repo, memberAuth(), &bcastMock{})
	_, err := svc.UpdateItem(ctx, user, itemID, packing.UpdateItemPayload{
		Name: &name, Quantity: &qty, Category: &cat, Note: &note, SortOrder: &sort,
	})
	require.NoError(t, err)
	assert.Equal(t, "new", gotPatch["name"], "name must be trimmed")
	assert.Equal(t, 5, gotPatch["quantity"])
	assert.Equal(t, "Gear", gotPatch["category"], "category must be trimmed")
	assert.Equal(t, "keep dry", gotPatch["note"])
	assert.Equal(t, 3, gotPatch["sort_order"])
}

func TestUpdateItem_NameToEmpty_Rejected(t *testing.T) {
	itemID, user := uuid.New(), uuid.New()
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: user}, nil
		},
	}
	empty := "   "
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.UpdateItem(ctx, user, itemID, packing.UpdateItemPayload{Name: &empty})
	assert.ErrorIs(t, err, packing.ErrInvalidPayload)
}

func TestUpdateItem_QuantityZero_Rejected(t *testing.T) {
	itemID, user := uuid.New(), uuid.New()
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: user}, nil
		},
	}
	zero := 0
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.UpdateItem(ctx, user, itemID, packing.UpdateItemPayload{Quantity: &zero})
	assert.ErrorIs(t, err, packing.ErrInvalidPayload)
}

// ---- DeleteItem ----

func TestDeleteItem_Creator_Allowed_Broadcasts(t *testing.T) {
	itemID, tripID, user := uuid.New(), uuid.New(), uuid.New()
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: tripID, CreatedBy: user}, nil
		},
		deleteItem: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, itemID, id)
			return nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	require.NoError(t, svc.DeleteItem(ctx, user, itemID))
	require.Len(t, bc.calls, 1)
	assert.Equal(t, tripID, bc.calls[0].tripID)
	var frame struct {
		Type   string    `json:"type"`
		ItemID uuid.UUID `json:"item_id"`
	}
	require.NoError(t, json.Unmarshal(bc.calls[0].payload, &frame))
	assert.Equal(t, string(packing.BroadcastItemDeleted), frame.Type)
	assert.Equal(t, itemID, frame.ItemID)
}

func TestDeleteItem_TripOwner_Allowed(t *testing.T) {
	itemID, creator, owner := uuid.New(), uuid.New(), uuid.New()
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: creator}, nil
		},
	}
	svc := newSvc(repo, authWithOwner(owner), &bcastMock{})
	assert.NoError(t, svc.DeleteItem(ctx, owner, itemID))
}

func TestDeleteItem_OtherMember_Forbidden(t *testing.T) {
	itemID, creator, owner, stranger := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: creator}, nil
		},
		deleteItem: func(_ context.Context, _ uuid.UUID) error {
			t.Fatal("DeleteItem must not run for non-creator, non-owner")
			return nil
		},
	}
	svc := newSvc(repo, authWithOwner(owner), nil)
	assert.ErrorIs(t, svc.DeleteItem(ctx, stranger, itemID), packing.ErrForbidden)
}

func TestDeleteItem_NotFound(t *testing.T) {
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return nil, nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	assert.ErrorIs(t, svc.DeleteItem(ctx, uuid.New(), uuid.New()), packing.ErrItemNotFound)
}

// ---- SetPacked ----

func TestSetPacked_NotFound(t *testing.T) {
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) { return nil, nil },
	}
	svc := newSvc(repo, memberAuth(), nil)
	_, err := svc.SetPacked(ctx, uuid.New(), uuid.New(), true)
	assert.ErrorIs(t, err, packing.ErrItemNotFound)
}

func TestSetPacked_NonMember_Forbidden(t *testing.T) {
	itemID := uuid.New()
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: uuid.New()}, nil
		},
	}
	svc := newSvc(repo, nonMemberAuth(), nil)
	_, err := svc.SetPacked(ctx, uuid.New(), itemID, true)
	assert.ErrorIs(t, err, packing.ErrForbidden)
}

// Any trip member (not just the creator) can flip their own packed state —
// this is the key contract distinguishing SetPacked from Update/Delete.
func TestSetPacked_AnyMemberCanPack(t *testing.T) {
	itemID, tripID, creator, someMember := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	var gotItem, gotUser uuid.UUID
	var gotPacked bool
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: tripID, CreatedBy: creator}, nil
		},
		setPacked: func(_ context.Context, iid, uid uuid.UUID, packed bool) error {
			gotItem, gotUser, gotPacked = iid, uid, packed
			return nil
		},
		packInfoForItems: func(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (map[uuid.UUID]packing.ItemPackInfo, error) {
			return map[uuid.UUID]packing.ItemPackInfo{itemID: {PackedByMe: true, PackedCount: 1}}, nil
		},
	}
	bc := &bcastMock{}
	svc := newSvc(repo, memberAuth(), bc)
	dto, err := svc.SetPacked(ctx, someMember, itemID, true)
	require.NoError(t, err)
	assert.Equal(t, itemID, gotItem)
	assert.Equal(t, someMember, gotUser)
	assert.True(t, gotPacked)
	assert.True(t, dto.PackedByMe)
	assert.Equal(t, 1, dto.PackedCount)
	require.Len(t, bc.calls, 1)
	kind, _ := decodeItemFrame(t, bc.calls[0].payload)
	assert.Equal(t, string(packing.BroadcastItemUpdated), kind)
}

func TestSetPacked_UnpackPassesFalse(t *testing.T) {
	itemID := uuid.New()
	var gotPacked bool
	repo := &repoMock{
		findItem: func(_ context.Context, _ uuid.UUID) (*packing.Item, error) {
			return &packing.Item{ID: itemID, TripID: uuid.New(), CreatedBy: uuid.New()}, nil
		},
		setPacked: func(_ context.Context, _, _ uuid.UUID, packed bool) error {
			gotPacked = packed
			return nil
		},
	}
	svc := newSvc(repo, memberAuth(), &bcastMock{})
	_, err := svc.SetPacked(ctx, uuid.New(), itemID, false)
	require.NoError(t, err)
	assert.False(t, gotPacked, "packed=false must forward to repo")
}

// ---- error passthrough ----

func TestCreateItem_RepoError_Bubbles(t *testing.T) {
	sentinel := errors.New("boom")
	repo := &repoMock{
		createItem: func(_ context.Context, _ *packing.Item) error { return sentinel },
	}
	svc := newSvc(repo, memberAuth(), &bcastMock{})
	_, err := svc.CreateItem(ctx, uuid.New(), uuid.New(), packing.CreateItemPayload{Name: "x"})
	assert.ErrorIs(t, err, sentinel)
}
