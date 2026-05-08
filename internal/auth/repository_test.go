package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	runMigrations(t, db)
	return db
}

func runMigrations(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqls := []string{
		`CREATE SCHEMA IF NOT EXISTS auth`,
		`CREATE TABLE IF NOT EXISTS auth.users (
			id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email                 TEXT UNIQUE NOT NULL,
			name                  TEXT NOT NULL DEFAULT '',
			avatar_url            TEXT NOT NULL DEFAULT '',
			password_hash         TEXT,
			is_blocked            BOOLEAN NOT NULL DEFAULT false,
			is_verified           BOOLEAN NOT NULL DEFAULT false,
			status                VARCHAR(20) NOT NULL DEFAULT 'active',
			deactivated_at        TIMESTAMPTZ,
			deletion_scheduled_at TIMESTAMPTZ,
			created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS auth.providers (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id     UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
			provider    TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(provider, provider_id)
		)`,
		`CREATE TABLE IF NOT EXISTS auth.email_verifications (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at    TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS auth.password_resets (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at    TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS auth.mfa_configs (
			user_id     UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
			totp_secret TEXT NOT NULL,
			is_enabled  BOOLEAN NOT NULL DEFAULT false,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS auth.user_sessions (
			id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id            UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
			device_name        TEXT,
			device_type        VARCHAR(20),
			ip_address         TEXT,
			user_agent         TEXT,
			refresh_token_hash TEXT NOT NULL UNIQUE,
			last_active_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at         TIMESTAMPTZ NOT NULL,
			revoked_at         TIMESTAMPTZ,
			created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}
	for _, sql := range sqls {
		require.NoError(t, db.Exec(sql).Error)
	}
}

func newUser(t *testing.T) *auth.User {
	t.Helper()
	h := fmt.Sprintf("hash-%s", uuid.New())
	return &auth.User{
		ID:           uuid.New(),
		Email:        fmt.Sprintf("%s@example.com", uuid.New()),
		Name:         "Test User",
		PasswordHash: &h,
	}
}

func TestUserCRUD(t *testing.T) {
	repo := auth.NewRepository(setupDB(t))
	ctx := context.Background()

	t.Run("CreateAndFindByEmail", func(t *testing.T) {
		u := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u))

		got, err := repo.FindUserByEmail(ctx, u.Email)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, u.ID, got.ID)
	})

	t.Run("FindByEmail_Normalizes_Input", func(t *testing.T) {
		u := newUser(t)
		u.Email = "normalized@example.com"
		require.NoError(t, repo.CreateUser(ctx, u))

		got, err := repo.FindUserByEmail(ctx, "  NORMALIZED@EXAMPLE.COM  ")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, u.ID, got.ID)
	})

	t.Run("FindByEmail_NotFound_ReturnsNil", func(t *testing.T) {
		got, err := repo.FindUserByEmail(ctx, "nobody@example.com")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("FindByID", func(t *testing.T) {
		u := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u))

		got, err := repo.FindUserByID(ctx, u.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, u.ID, got.ID)
	})

	t.Run("FindByID_NotFound_ReturnsNil", func(t *testing.T) {
		got, err := repo.FindUserByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("FindByIDs", func(t *testing.T) {
		u1, u2 := newUser(t), newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u1))
		require.NoError(t, repo.CreateUser(ctx, u2))

		got, err := repo.FindUserByIDs(ctx, []uuid.UUID{u1.ID, u2.ID})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("UpdateUserFields", func(t *testing.T) {
		u := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u))

		require.NoError(t, repo.UpdateUserFields(ctx, u.ID, map[string]any{
			"name":        "Updated Name",
			"is_verified": true,
			"status":      auth.UserStatusActive,
		}))

		got, err := repo.FindUserByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", got.Name)
		assert.True(t, got.IsVerified)
	})

}

func TestProviderRepository(t *testing.T) {
	repo := auth.NewRepository(setupDB(t))
	ctx := context.Background()

	u := newUser(t)
	require.NoError(t, repo.CreateUser(ctx, u))

	t.Run("CreateAndFind", func(t *testing.T) {
		p := &auth.Provider{
			ID:         uuid.New(),
			UserID:     u.ID,
			Provider:   "google",
			ProviderID: uuid.NewString(),
		}
		require.NoError(t, repo.CreateProvider(ctx, p))

		got, err := repo.FindProviderByProviderID(ctx, p.Provider, p.ProviderID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, p.ID, got.ID)
	})

	t.Run("Find_NotFound_ReturnsNil", func(t *testing.T) {
		got, err := repo.FindProviderByProviderID(ctx, "github", "no-such-id")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestEmailVerificationRepository(t *testing.T) {
	repo := auth.NewRepository(setupDB(t))
	ctx := context.Background()

	u := newUser(t)
	require.NoError(t, repo.CreateUser(ctx, u))

	t.Run("CreateAndFind", func(t *testing.T) {
		ev := &auth.EmailVerification{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: uuid.NewString(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, repo.CreateEmailVerification(ctx, ev))

		got, err := repo.FindEmailVerificationByTokenHash(ctx, ev.TokenHash)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, ev.ID, got.ID)
	})

	t.Run("Find_NotFound_ReturnsNil", func(t *testing.T) {
		got, err := repo.FindEmailVerificationByTokenHash(ctx, "no-such-hash")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Find_AfterMarkUsed_ReturnsNil", func(t *testing.T) {
		ev := &auth.EmailVerification{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: uuid.NewString(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, repo.CreateEmailVerification(ctx, ev))
		require.NoError(t, repo.MarkEmailVerificationUsed(ctx, ev.ID))

		got, err := repo.FindEmailVerificationByTokenHash(ctx, ev.TokenHash)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Find_Expired_ReturnsNil", func(t *testing.T) {
		ev := &auth.EmailVerification{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: uuid.NewString(),
			ExpiresAt: time.Now().Add(-time.Minute),
		}
		require.NoError(t, repo.CreateEmailVerification(ctx, ev))

		got, err := repo.FindEmailVerificationByTokenHash(ctx, ev.TokenHash)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("MarkUsed_SetsUsedAt", func(t *testing.T) {
		ev := &auth.EmailVerification{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: uuid.NewString(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, repo.CreateEmailVerification(ctx, ev))
		require.NoError(t, repo.MarkEmailVerificationUsed(ctx, ev.ID))

		var stored auth.EmailVerification
		require.NoError(t, repo.(*auth.RepositoryForTest).DB().Where("id = ?", ev.ID).First(&stored).Error)
		require.NotNil(t, stored.UsedAt)
	})
}

func TestPasswordResetRepository(t *testing.T) {
	repo := auth.NewRepository(setupDB(t))
	ctx := context.Background()

	u := newUser(t)
	require.NoError(t, repo.CreateUser(ctx, u))

	t.Run("CreateAndFind", func(t *testing.T) {
		pr := &auth.PasswordReset{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: uuid.NewString(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, repo.CreatePasswordReset(ctx, pr))

		got, err := repo.FindPasswordResetByTokenHash(ctx, pr.TokenHash)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, pr.ID, got.ID)
	})

	t.Run("Find_NotFound_ReturnsNil", func(t *testing.T) {
		got, err := repo.FindPasswordResetByTokenHash(ctx, "no-such-hash")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Find_AfterMarkUsed_ReturnsNil", func(t *testing.T) {
		pr := &auth.PasswordReset{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: uuid.NewString(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, repo.CreatePasswordReset(ctx, pr))
		require.NoError(t, repo.MarkPasswordResetUsed(ctx, pr.ID))

		got, err := repo.FindPasswordResetByTokenHash(ctx, pr.TokenHash)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Find_Expired_ReturnsNil", func(t *testing.T) {
		pr := &auth.PasswordReset{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: uuid.NewString(),
			ExpiresAt: time.Now().Add(-time.Minute),
		}
		require.NoError(t, repo.CreatePasswordReset(ctx, pr))

		got, err := repo.FindPasswordResetByTokenHash(ctx, pr.TokenHash)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestMFAConfigRepository(t *testing.T) {
	repo := auth.NewRepository(setupDB(t))
	ctx := context.Background()

	u := newUser(t)
	require.NoError(t, repo.CreateUser(ctx, u))

	t.Run("UpsertInsert", func(t *testing.T) {
		cfg := &auth.MFAConfig{
			UserID:     u.ID,
			TOTPSecret: "secret-abc",
			IsEnabled:  false,
		}
		require.NoError(t, repo.UpsertMFAConfig(ctx, cfg))

		got, err := repo.FindMFAConfig(ctx, u.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "secret-abc", got.TOTPSecret)
		assert.False(t, got.IsEnabled)
	})

	t.Run("UpsertUpdate", func(t *testing.T) {
		cfg := &auth.MFAConfig{
			UserID:     u.ID,
			TOTPSecret: "secret-xyz",
			IsEnabled:  true,
		}
		require.NoError(t, repo.UpsertMFAConfig(ctx, cfg))

		got, err := repo.FindMFAConfig(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, "secret-xyz", got.TOTPSecret)
		assert.True(t, got.IsEnabled)
	})

	t.Run("Find_NotFound_ReturnsNil", func(t *testing.T) {
		got, err := repo.FindMFAConfig(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestUserSessionRepository(t *testing.T) {
	repo := auth.NewRepository(setupDB(t))
	ctx := context.Background()

	u := newUser(t)
	require.NoError(t, repo.CreateUser(ctx, u))

	newSession := func(t *testing.T, expiresIn time.Duration) *auth.UserSession {
		t.Helper()
		return &auth.UserSession{
			ID:               uuid.New(),
			UserID:           u.ID,
			DeviceType:       auth.DeviceWeb,
			RefreshTokenHash: uuid.NewString(),
			LastActiveAt:     time.Now(),
			ExpiresAt:        time.Now().Add(expiresIn),
		}
	}

	t.Run("CreateAndFind", func(t *testing.T) {
		s := newSession(t, time.Hour)
		require.NoError(t, repo.CreateUserSession(ctx, s))

		got, err := repo.FindUserSessionByRefreshTokenHash(ctx, s.RefreshTokenHash)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, s.ID, got.ID)
	})

	t.Run("Find_NotFound_ReturnsNil", func(t *testing.T) {
		got, err := repo.FindUserSessionByRefreshTokenHash(ctx, "no-such-hash")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Find_Revoked_ReturnsNil", func(t *testing.T) {
		s := newSession(t, time.Hour)
		require.NoError(t, repo.CreateUserSession(ctx, s))
		require.NoError(t, repo.RevokeUserSession(ctx, s.ID))

		got, err := repo.FindUserSessionByRefreshTokenHash(ctx, s.RefreshTokenHash)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Find_Expired_ReturnsNil", func(t *testing.T) {
		s := newSession(t, -time.Minute)
		require.NoError(t, repo.CreateUserSession(ctx, s))

		got, err := repo.FindUserSessionByRefreshTokenHash(ctx, s.RefreshTokenHash)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("RevokeSession_SetsRevokedAt", func(t *testing.T) {
		s := newSession(t, time.Hour)
		require.NoError(t, repo.CreateUserSession(ctx, s))
		require.NoError(t, repo.RevokeUserSession(ctx, s.ID))

		var stored auth.UserSession
		require.NoError(t, repo.(*auth.RepositoryForTest).DB().Where("id = ?", s.ID).First(&stored).Error)
		require.NotNil(t, stored.RevokedAt)
	})

	t.Run("ListSessionByUserID_OnlyActiveAndUnexpired", func(t *testing.T) {
		u2 := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u2))

		active := &auth.UserSession{
			ID: uuid.New(), UserID: u2.ID, DeviceType: auth.DeviceWeb,
			RefreshTokenHash: uuid.NewString(), LastActiveAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		revoked := &auth.UserSession{
			ID: uuid.New(), UserID: u2.ID, DeviceType: auth.DeviceIOS,
			RefreshTokenHash: uuid.NewString(), LastActiveAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		expired := &auth.UserSession{
			ID: uuid.New(), UserID: u2.ID, DeviceType: auth.DeviceAndroid,
			RefreshTokenHash: uuid.NewString(), LastActiveAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-time.Minute),
		}

		require.NoError(t, repo.CreateUserSession(ctx, active))
		require.NoError(t, repo.CreateUserSession(ctx, revoked))
		require.NoError(t, repo.CreateUserSession(ctx, expired))
		require.NoError(t, repo.RevokeUserSession(ctx, revoked.ID))

		sessions, err := repo.ListSessionByUserID(ctx, u2.ID)
		require.NoError(t, err)
		require.Len(t, sessions, 1)
		assert.Equal(t, active.ID, (sessions)[0].ID)
	})

	t.Run("DeleteUserSession_SoftRevokes", func(t *testing.T) {
		s := newSession(t, time.Hour)
		require.NoError(t, repo.CreateUserSession(ctx, s))
		require.NoError(t, repo.DeleteUserSession(ctx, s.ID))

		// row still exists but revoked_at is set
		var stored auth.UserSession
		require.NoError(t, repo.(*auth.RepositoryForTest).DB().Where("id = ?", s.ID).First(&stored).Error)
		require.NotNil(t, stored.RevokedAt)

		// FindBy returns nil for revoked session
		got, err := repo.FindUserSessionByRefreshTokenHash(ctx, s.RefreshTokenHash)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("DeleteAllSessions_SoftRevokesAll", func(t *testing.T) {
		u3 := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u3))

		for range 3 {
			s := &auth.UserSession{
				ID:               uuid.New(),
				UserID:           u3.ID,
				RefreshTokenHash: uuid.NewString(),
				LastActiveAt:     time.Now(),
				ExpiresAt:        time.Now().Add(time.Hour),
			}
			require.NoError(t, repo.CreateUserSession(ctx, s))
		}

		require.NoError(t, repo.DeleteAllSessions(ctx, u3.ID))

		// ListSession filters out revoked; should be empty
		sessions, err := repo.ListSessionByUserID(ctx, u3.ID)
		require.NoError(t, err)
		assert.Empty(t, sessions)

		// Rows still exist in DB
		var count int64
		require.NoError(t, repo.(*auth.RepositoryForTest).DB().Model(&auth.UserSession{}).
			Where("user_id = ? AND revoked_at IS NOT NULL", u3.ID).Count(&count).Error)
		assert.EqualValues(t, 3, count)
	})
}

func TestUserLifecycle(t *testing.T) {
	repo := auth.NewRepository(setupDB(t))
	ctx := context.Background()

	t.Run("DeactivateUser", func(t *testing.T) {
		u := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u))
		require.NoError(t, repo.DeactivateUser(ctx, u.ID))

		got, err := repo.FindUserByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, auth.UserStatusDeactivated, got.Status)
		require.NotNil(t, got.DeactivatedAt)
	})

	t.Run("MarkUserDeleted", func(t *testing.T) {
		u := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u))
		require.NoError(t, repo.MarkUserDeleted(ctx, u.ID))

		got, err := repo.FindUserByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, auth.UserStatusDeleted, got.Status)
		require.NotNil(t, got.DeletionScheduledAt)
	})

	t.Run("BlockUser", func(t *testing.T) {
		u := newUser(t)
		require.NoError(t, repo.CreateUser(ctx, u))
		require.NoError(t, repo.BlockUser(ctx, u.ID))

		got, err := repo.FindUserByID(ctx, u.ID)
		require.NoError(t, err)
		assert.True(t, got.IsBlocked)
	})
}
