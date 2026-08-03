package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

var (
	ErrProfileNotFound  = errors.New("profile not found")
	ErrUsernameTaken    = errors.New("username taken")
	ErrAlreadyFollowing = errors.New("already following")
	ErrNotFollowing     = errors.New("not following")
	ErrCannotFollowSelf = errors.New("cannot follow self")
)

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	// Profile
	FindProfileByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
	FindProfileByUsername(ctx context.Context, username string) (*Profile, error)
	UsernameExists(ctx context.Context, username string, exceptUserID uuid.UUID) (bool, error)
	UpsertProfile(ctx context.Context, p *Profile) error

	// Follows
	Follow(ctx context.Context, followerID, followeeID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error
	IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error)
	AreFollowing(ctx context.Context, followerID uuid.UUID, targetIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	ListFollowers(ctx context.Context, userID uuid.UUID, cursor *FollowCursor, limit int) ([]Follow, error)
	ListFollowing(ctx context.Context, userID uuid.UUID, cursor *FollowCursor, limit int) ([]Follow, error)
	GetCounts(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]FollowCounts, error)

	// Users (denorm join)
	FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error)

	// Feed
	InsertFeedEntries(ctx context.Context, entries []FeedEntry) error
	ListFeed(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) ([]FeedEntry, error)

	// Recommendations
	FriendsOfFollowees(ctx context.Context, userID uuid.UUID, limit int) ([]uuid.UUID, error)
	TrendingUsers(ctx context.Context, since time.Time, excludeUserID uuid.UUID, limit int) ([]uuid.UUID, error)
}

type FollowCursor struct {
	CreatedAt time.Time
	OtherID   uuid.UUID
}

type FeedCursor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

// ---- Profile ----

func (r *repository) FindProfileByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.db.WithContext(ctx).First(&p, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *repository) FindProfileByUsername(ctx context.Context, username string) (*Profile, error) {
	var p Profile
	err := r.db.WithContext(ctx).First(&p, "username = ?", strings.TrimSpace(username)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *repository) UsernameExists(ctx context.Context, username string, exceptUserID uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&Profile{}).Where("username = ?", strings.TrimSpace(username))
	if exceptUserID != uuid.Nil {
		q = q.Where("user_id <> ?", exceptUserID)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *repository) UpsertProfile(ctx context.Context, p *Profile) error {
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.TravelTags == nil {
		p.TravelTags = pq.StringArray{}
	}
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO profile.profiles (
		    user_id, username, bio, avatar_url, cover_image, travel_tags,
		    social_instagram, social_x, social_youtube, social_tiktok,
		    created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
		    username         = EXCLUDED.username,
		    bio              = EXCLUDED.bio,
		    avatar_url       = EXCLUDED.avatar_url,
		    cover_image      = EXCLUDED.cover_image,
		    travel_tags      = EXCLUDED.travel_tags,
		    social_instagram = EXCLUDED.social_instagram,
		    social_x         = EXCLUDED.social_x,
		    social_youtube   = EXCLUDED.social_youtube,
		    social_tiktok    = EXCLUDED.social_tiktok,
		    updated_at       = EXCLUDED.updated_at
	`,
		p.UserID, p.Username, p.Bio, p.AvatarURL, p.CoverImage, p.TravelTags,
		p.SocialInstagram, p.SocialX, p.SocialYoutube, p.SocialTiktok,
		p.CreatedAt, p.UpdatedAt,
	).Error
}

// ---- Follows ----

func (r *repository) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return ErrCannotFollowSelf
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`
			INSERT INTO profile.follows (follower_id, followee_id, created_at)
			VALUES (?, ?, now())
			ON CONFLICT DO NOTHING
		`, followerID, followeeID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrAlreadyFollowing
		}
		if err := bumpFollowCount(tx, followerID, "following_count", +1); err != nil {
			return err
		}
		return bumpFollowCount(tx, followeeID, "followers_count", +1)
	})
}

func (r *repository) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
			Delete(&Follow{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFollowing
		}
		if err := bumpFollowCount(tx, followerID, "following_count", -1); err != nil {
			return err
		}
		return bumpFollowCount(tx, followeeID, "followers_count", -1)
	})
}

func bumpFollowCount(tx *gorm.DB, userID uuid.UUID, col string, delta int) error {
	initFollowers, initFollowing := 0, 0
	switch col {
	case "followers_count":
		if delta > 0 {
			initFollowers = delta
		}
	case "following_count":
		if delta > 0 {
			initFollowing = delta
		}
	default:
		return errors.New("bumpFollowCount: unknown column")
	}
	return tx.Exec(`
		INSERT INTO profile.follow_counts (user_id, followers_count, following_count, updated_at)
		VALUES (?, ?, ?, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    `+col+` = profile.follow_counts.`+col+` + ?,
		    updated_at = now()
	`, userID, initFollowers, initFollowing, delta).Error
}

func (r *repository) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&n).Error
	return n > 0, err
}

func (r *repository) AreFollowing(ctx context.Context, followerID uuid.UUID, targetIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(targetIDs))
	if len(targetIDs) == 0 {
		return out, nil
	}
	var rows []Follow
	err := r.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id IN ?", followerID, targetIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, f := range rows {
		out[f.FolloweeID] = true
	}
	return out, nil
}

func (r *repository) ListFollowers(ctx context.Context, userID uuid.UUID, cursor *FollowCursor, limit int) ([]Follow, error) {
	return r.listFollowsSide(ctx, "followee_id", "follower_id", userID, cursor, limit)
}

func (r *repository) ListFollowing(ctx context.Context, userID uuid.UUID, cursor *FollowCursor, limit int) ([]Follow, error) {
	return r.listFollowsSide(ctx, "follower_id", "followee_id", userID, cursor, limit)
}

func (r *repository) listFollowsSide(ctx context.Context, pivot, other string, userID uuid.UUID, cursor *FollowCursor, limit int) ([]Follow, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&Follow{}).Where(pivot+" = ?", userID)
	if cursor != nil {
		q = q.Where("(created_at, "+other+") < (?, ?)", cursor.CreatedAt, cursor.OtherID)
	}
	var rows []Follow
	err := q.Order("created_at DESC, " + other + " DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) GetCounts(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]FollowCounts, error) {
	out := make(map[uuid.UUID]FollowCounts, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var rows []FollowCounts
	err := r.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, c := range rows {
		out[c.UserID] = c
	}
	return out, nil
}

// ---- Users ----

func (r *repository) FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	out := make(map[uuid.UUID]auth.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []auth.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out, nil
}

// ---- Feed ----

func (r *repository) InsertFeedEntries(ctx context.Context, entries []FeedEntry) error {
	if len(entries) == 0 {
		return nil
	}
	for i := range entries {
		if entries[i].ID == uuid.Nil {
			entries[i].ID = uuid.New()
		}
	}
	return r.db.WithContext(ctx).
		Exec(`INSERT INTO profile.feed AS f
		      (id, user_id, actor_id, event_type, trip_id, published_at)
		      VALUES `+placeholders(len(entries), 6)+`
		      ON CONFLICT (user_id, actor_id, event_type, trip_id) DO NOTHING`,
			flattenFeedArgs(entries)...).Error
}

func placeholders(rows, cols int) string {
	if rows == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < rows; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("(")
		for j := 0; j < cols; j++ {
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString("?")
		}
		b.WriteString(")")
	}
	return b.String()
}

func flattenFeedArgs(entries []FeedEntry) []any {
	args := make([]any, 0, len(entries)*6)
	for _, e := range entries {
		args = append(args, e.ID, e.UserID, e.ActorID, e.EventType, e.TripID, e.PublishedAt)
	}
	return args
}

func (r *repository) ListFeed(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) ([]FeedEntry, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if cursor != nil {
		q = q.Where("(published_at, id) < (?, ?)", cursor.PublishedAt, cursor.ID)
	}
	var rows []FeedEntry
	err := q.Order("published_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// ---- Recommendations ----

func (r *repository) FriendsOfFollowees(ctx context.Context, userID uuid.UUID, limit int) ([]uuid.UUID, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	type row struct {
		ID uuid.UUID
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT f2.followee_id AS id
		FROM profile.follows f1
		JOIN profile.follows f2 ON f2.follower_id = f1.followee_id
		WHERE f1.follower_id = ?
		  AND f2.followee_id <> ?
		  AND NOT EXISTS (
		      SELECT 1 FROM profile.follows me
		      WHERE me.follower_id = ? AND me.followee_id = f2.followee_id
		  )
		GROUP BY f2.followee_id
		ORDER BY COUNT(*) DESC
		LIMIT ?
	`, userID, userID, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// TrendingUsers returns user IDs with the most new followers since `since`,
// excluding the requester. Used as fallback signal when FoF returns too few.
func (r *repository) TrendingUsers(ctx context.Context, since time.Time, excludeUserID uuid.UUID, limit int) ([]uuid.UUID, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	type row struct {
		ID uuid.UUID
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT followee_id AS id
		FROM profile.follows
		WHERE created_at >= ?
		  AND followee_id <> ?
		  AND NOT EXISTS (
		      SELECT 1 FROM profile.follows me
		      WHERE me.follower_id = ? AND me.followee_id = profile.follows.followee_id
		  )
		GROUP BY followee_id
		ORDER BY COUNT(*) DESC
		LIMIT ?
	`, since, excludeUserID, excludeUserID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids, nil
}
