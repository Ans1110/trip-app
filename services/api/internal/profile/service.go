package profile

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidUsername  = errors.New("invalid username")
	ErrReservedUsername = errors.New("username is reserved")
)

type IService interface {
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error)
	GetProfileByUsername(ctx context.Context, viewerID uuid.UUID, username string) (*ProfileResponse, error)
	UpdateMyProfile(ctx context.Context, userID uuid.UUID, patch UpdateProfilePayload) (*ProfileResponse, error)

	Follow(ctx context.Context, followerID, followeeID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error

	ListFollowers(ctx context.Context, viewerID, targetID uuid.UUID, cursor *FollowCursor, limit int) (ListUsersResponse, error)
	ListFollowing(ctx context.Context, viewerID, targetID uuid.UUID, cursor *FollowCursor, limit int) (ListUsersResponse, error)

	ListFeed(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) (FeedResponse, error)

	Recommendations(ctx context.Context, userID uuid.UUID, limit int) (RecommendationResponse, error)
}

type service struct {
	repo   IRepository
	logger *zap.Logger
}

type ServiceConfig struct {
	Repo   IRepository
	Logger *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		logger: logger.With(zap.String("layer", "profile.service")),
	}
}

// ---- Profile ----

// GetMyProfile returns the caller's profile, auto-provisioning a placeholder
// row on first access so the FE always has something to render.
func (s *service) GetMyProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error) {
	p, err := s.repo.FindProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p, err = s.provisionPlaceholder(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	users, err := s.repo.FindUsersByIDs(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}
	u, ok := users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	counts, err := s.repo.GetCounts(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}
	return buildProfileResponse(p, u, counts[userID], false, true), nil
}

func (s *service) GetProfileByUsername(ctx context.Context, viewerID uuid.UUID, username string) (*ProfileResponse, error) {
	p, err := s.repo.FindProfileByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProfileNotFound
	}
	users, err := s.repo.FindUsersByIDs(ctx, []uuid.UUID{p.UserID})
	if err != nil {
		return nil, err
	}
	u, ok := users[p.UserID]
	if !ok {
		return nil, ErrUserNotFound
	}
	counts, err := s.repo.GetCounts(ctx, []uuid.UUID{p.UserID})
	if err != nil {
		return nil, err
	}
	isSelf := viewerID != uuid.Nil && viewerID == p.UserID
	isFollowing := false
	if !isSelf && viewerID != uuid.Nil {
		following, err := s.repo.AreFollowing(ctx, viewerID, []uuid.UUID{p.UserID})
		if err != nil {
			return nil, err
		}
		isFollowing = following[p.UserID]
	}
	return buildProfileResponse(p, u, counts[p.UserID], isFollowing, isSelf), nil
}

func (s *service) UpdateMyProfile(ctx context.Context, userID uuid.UUID, patch UpdateProfilePayload) (*ProfileResponse, error) {
	existing, err := s.repo.FindProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		existing, err = s.provisionPlaceholder(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	if patch.Username != nil {
		normalized := strings.TrimSpace(*patch.Username)
		if err := validateUsername(normalized); err != nil {
			return nil, err
		}
		if !strings.EqualFold(normalized, existing.Username) {
			taken, err := s.repo.UsernameExists(ctx, normalized, userID)
			if err != nil {
				return nil, err
			}
			if taken {
				return nil, ErrUsernameTaken
			}
		}
		existing.Username = normalized
	}
	if patch.Bio != nil {
		existing.Bio = *patch.Bio
	}
	if patch.AvatarURL != nil {
		existing.AvatarURL = *patch.AvatarURL
	}
	if patch.CoverImage != nil {
		existing.CoverImage = *patch.CoverImage
	}
	if patch.TravelTags != nil {
		existing.TravelTags = normalizeTags(*patch.TravelTags)
	}
	if patch.SocialInstagram != nil {
		existing.SocialInstagram = *patch.SocialInstagram
	}
	if patch.SocialX != nil {
		existing.SocialX = *patch.SocialX
	}
	if patch.SocialYoutube != nil {
		existing.SocialYoutube = *patch.SocialYoutube
	}
	if patch.SocialTiktok != nil {
		existing.SocialTiktok = *patch.SocialTiktok
	}
	if err := s.repo.UpsertProfile(ctx, existing); err != nil {
		return nil, err
	}
	return s.GetMyProfile(ctx, userID)
}

// provisionPlaceholder writes a minimal profile row with a generated username
// so downstream calls that assume a profile always exists don't have to guard.
func (s *service) provisionPlaceholder(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	p := &Profile{
		UserID:   userID,
		Username: "user_" + strings.ReplaceAll(userID.String(), "-", "")[:12],
	}
	if err := s.repo.UpsertProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ---- Follows ----

func (s *service) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return ErrCannotFollowSelf
	}
	users, err := s.repo.FindUsersByIDs(ctx, []uuid.UUID{followeeID})
	if err != nil {
		return err
	}
	if _, ok := users[followeeID]; !ok {
		return ErrUserNotFound
	}
	return s.repo.Follow(ctx, followerID, followeeID)
}

func (s *service) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return ErrCannotFollowSelf
	}
	return s.repo.Unfollow(ctx, followerID, followeeID)
}

func (s *service) ListFollowers(ctx context.Context, viewerID, targetID uuid.UUID, cursor *FollowCursor, limit int) (ListUsersResponse, error) {
	rows, err := s.repo.ListFollowers(ctx, targetID, cursor, limit)
	if err != nil {
		return ListUsersResponse{}, err
	}
	return s.hydrateFollowList(ctx, viewerID, rows, func(f Follow) uuid.UUID { return f.FollowerID }, func(f Follow) time.Time { return f.CreatedAt })
}

func (s *service) ListFollowing(ctx context.Context, viewerID, targetID uuid.UUID, cursor *FollowCursor, limit int) (ListUsersResponse, error) {
	rows, err := s.repo.ListFollowing(ctx, targetID, cursor, limit)
	if err != nil {
		return ListUsersResponse{}, err
	}
	return s.hydrateFollowList(ctx, viewerID, rows, func(f Follow) uuid.UUID { return f.FolloweeID }, func(f Follow) time.Time { return f.CreatedAt })
}

func (s *service) hydrateFollowList(ctx context.Context, viewerID uuid.UUID, rows []Follow, pickID func(Follow) uuid.UUID, pickTime func(Follow) time.Time) (ListUsersResponse, error) {
	if len(rows) == 0 {
		return ListUsersResponse{Users: []UserSummary{}}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, pickID(r))
	}
	profiles, err := s.findProfilesByUserIDs(ctx, ids)
	if err != nil {
		return ListUsersResponse{}, err
	}
	users, err := s.repo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return ListUsersResponse{}, err
	}
	out := make([]UserSummary, 0, len(rows))
	for _, r := range rows {
		id := pickID(r)
		u, ok := users[id]
		if !ok {
			continue
		}
		out = append(out, toUserSummary(u, profiles[id]))
	}
	// Suppress viewer-based following flag here (list already uses profile scope);
	// caller can query per-user via GetProfileByUsername when needed.
	_ = viewerID

	last := rows[len(rows)-1]
	cursor := encodeFollowCursor(pickTime(last), pickID(last))
	return ListUsersResponse{Users: out, NextCursor: cursor}, nil
}

// ---- Feed ----

func (s *service) ListFeed(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) (FeedResponse, error) {
	rows, err := s.repo.ListFeed(ctx, userID, cursor, limit)
	if err != nil {
		return FeedResponse{}, err
	}
	if len(rows) == 0 {
		return FeedResponse{Items: []FeedItemResponse{}}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{}, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.ActorID]; !ok {
			seen[r.ActorID] = struct{}{}
			ids = append(ids, r.ActorID)
		}
	}
	users, err := s.repo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return FeedResponse{}, err
	}
	profiles, err := s.findProfilesByUserIDs(ctx, ids)
	if err != nil {
		return FeedResponse{}, err
	}
	items := make([]FeedItemResponse, 0, len(rows))
	for _, r := range rows {
		u, ok := users[r.ActorID]
		if !ok {
			continue
		}
		item := FeedItemResponse{
			ID:          r.ID.String(),
			EventType:   r.EventType,
			Actor:       toUserSummary(u, profiles[r.ActorID]),
			PublishedAt: r.PublishedAt,
		}
		if r.TripID != nil {
			item.TripID = r.TripID.String()
		}
		items = append(items, item)
	}
	last := rows[len(rows)-1]
	return FeedResponse{
		Items:      items,
		NextCursor: encodeFeedCursor(last.PublishedAt, last.ID),
	}, nil
}

// ---- Recommendations ----

// Recommendations merges friends-of-followees candidates with 7-day trending
// as fallback. Both sources exclude users the caller already follows.
func (s *service) Recommendations(ctx context.Context, userID uuid.UUID, limit int) (RecommendationResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	fof, err := s.repo.FriendsOfFollowees(ctx, userID, limit)
	if err != nil {
		return RecommendationResponse{}, err
	}

	need := limit - len(fof)
	var trending []uuid.UUID
	if need > 0 {
		trending, err = s.repo.TrendingUsers(ctx, time.Now().Add(-7*24*time.Hour), userID, need+len(fof))
		if err != nil {
			return RecommendationResponse{}, err
		}
	}

	ids := mergeUnique(fof, trending, limit)
	if len(ids) == 0 {
		return RecommendationResponse{Users: []UserSummary{}}, nil
	}
	users, err := s.repo.FindUsersByIDs(ctx, ids)
	if err != nil {
		return RecommendationResponse{}, err
	}
	profiles, err := s.findProfilesByUserIDs(ctx, ids)
	if err != nil {
		return RecommendationResponse{}, err
	}
	out := make([]UserSummary, 0, len(ids))
	for _, id := range ids {
		u, ok := users[id]
		if !ok {
			continue
		}
		out = append(out, toUserSummary(u, profiles[id]))
	}
	return RecommendationResponse{Users: out}, nil
}

// ---- helpers ----

func (s *service) findProfilesByUserIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Profile, error) {
	out := make(map[uuid.UUID]*Profile, len(ids))
	for _, id := range ids {
		p, err := s.repo.FindProfileByUserID(ctx, id)
		if err != nil {
			return nil, err
		}
		if p != nil {
			out[id] = p
		}
	}
	return out, nil
}

func mergeUnique(primary, secondary []uuid.UUID, limit int) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, limit)
	out := make([]uuid.UUID, 0, limit)
	for _, id := range primary {
		if len(out) >= limit {
			break
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range secondary {
		if len(out) >= limit {
			break
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

var reservedUsernames = map[string]struct{}{
	"admin": {}, "root": {}, "system": {}, "support": {}, "help": {},
	"api": {}, "www": {}, "app": {}, "me": {}, "settings": {},
	"login": {}, "logout": {}, "signup": {}, "signin": {}, "register": {},
	"profile": {}, "profiles": {}, "user": {}, "users": {},
	"feed": {}, "explore": {}, "notifications": {},
	"trip": {}, "trips": {}, "album": {}, "albums": {}, "calendar": {},
	"public": {}, "private": {}, "static": {}, "assets": {},
}

func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}
	if _, reserved := reservedUsernames[strings.ToLower(username)]; reserved {
		return ErrReservedUsername
	}
	return nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

func toUserSummary(u auth.User, p *Profile) UserSummary {
	s := UserSummary{
		UserID:    u.ID.String(),
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
	}
	if p != nil {
		s.Username = p.Username
		if p.AvatarURL != "" {
			s.AvatarURL = p.AvatarURL
		}
	}
	return s
}

func buildProfileResponse(p *Profile, u auth.User, counts FollowCounts, isFollowing, isSelf bool) *ProfileResponse {
	avatar := u.AvatarURL
	if p.AvatarURL != "" {
		avatar = p.AvatarURL
	}
	tags := []string(p.TravelTags)
	if tags == nil {
		tags = []string{}
	}
	return &ProfileResponse{
		UserID:          p.UserID.String(),
		Username:        p.Username,
		Name:            u.Name,
		Bio:             p.Bio,
		AvatarURL:       avatar,
		CoverImage:      p.CoverImage,
		TravelTags:      tags,
		SocialInstagram: p.SocialInstagram,
		SocialX:         p.SocialX,
		SocialYoutube:   p.SocialYoutube,
		SocialTiktok:    p.SocialTiktok,
		FollowersCount:  counts.FollowersCount,
		FollowingCount:  counts.FollowingCount,
		IsFollowing:     isFollowing,
		IsSelf:          isSelf,
		CreatedAt:       p.CreatedAt,
	}
}

// ---- cursor encoding ----

func encodeFollowCursor(t time.Time, id uuid.UUID) string {
	return encodeCursor(t, id)
}

func encodeFeedCursor(t time.Time, id uuid.UUID) string {
	return encodeCursor(t, id)
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	if t.IsZero() {
		return ""
	}
	return timeToMicros(t) + ":" + id.String()
}

func timeToMicros(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func DecodeFollowCursor(token string) (*FollowCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &FollowCursor{CreatedAt: *t, OtherID: id}, nil
}

func DecodeFeedCursor(token string) (*FeedCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &FeedCursor{PublishedAt: *t, ID: id}, nil
}

func decodeCursor(token string) (*time.Time, uuid.UUID, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, uuid.Nil, nil
	}
	sep := strings.LastIndex(token, ":")
	if sep <= 0 || sep == len(token)-1 {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, token[:sep])
	if err != nil {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(token[sep+1:])
	if err != nil {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	return &t, id, nil
}
