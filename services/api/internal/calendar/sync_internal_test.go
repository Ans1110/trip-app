package calendar

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTripDayBounds_UTCWhenTZEmpty(t *testing.T) {
	start := time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC)
	end := time.Date(2026, 7, 5, 8, 0, 0, 0, time.UTC)
	gotStart, gotEnd := tripDayBounds(start, end, "")

	assert.Equal(t, time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), gotStart)
	// End is the *exclusive* day-after so the last calendar day is included.
	assert.Equal(t, time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC), gotEnd)
}

func TestTripDayBounds_UsesIANAZoneForWallClock(t *testing.T) {
	tz := "Asia/Taipei" // UTC+8, no DST
	loc, err := time.LoadLocation(tz)
	require.NoError(t, err)

	start := time.Date(2026, 7, 3, 0, 0, 0, 0, loc)
	end := time.Date(2026, 7, 3, 0, 0, 0, 0, loc)

	gotStart, gotEnd := tripDayBounds(start, end, tz)

	// Start of 2026-07-03 in Taipei == 2026-07-02T16:00:00Z
	assert.Equal(t, time.Date(2026, 7, 2, 16, 0, 0, 0, time.UTC), gotStart)
	// End is start-of-next-day in Taipei == 2026-07-03T16:00:00Z
	assert.Equal(t, time.Date(2026, 7, 3, 16, 0, 0, 0, time.UTC), gotEnd)
}

func TestTripDayBounds_UnknownZoneFallsBackToUTC(t *testing.T) {
	start := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	gotStart, gotEnd := tripDayBounds(start, end, "Not/A_Real_Zone")
	assert.Equal(t, time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), gotStart)
	assert.Equal(t, time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC), gotEnd)
}

func TestBuildMemberRows_OwnerFirstAndUnique(t *testing.T) {
	eventID := uuid.New()
	owner := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()

	// Owner appears both in owner slot and in memberIDs -> only one row.
	// m1 is duplicated -> deduped.
	rows := buildMemberRows(eventID, owner, []uuid.UUID{owner, m1, m1, m2})
	require.Len(t, rows, 3)

	assert.Equal(t, owner, rows[0].UserID)
	assert.Equal(t, MemberRoleOwner, rows[0].Role)
	assert.Equal(t, eventID, rows[0].EventID)

	// The remaining slots are the non-owner members in insertion order.
	got := map[uuid.UUID]MemberRole{rows[1].UserID: rows[1].Role, rows[2].UserID: rows[2].Role}
	assert.Equal(t, MemberRoleMember, got[m1])
	assert.Equal(t, MemberRoleMember, got[m2])
}

func TestWireOpFor(t *testing.T) {
	assert.Equal(t, "CALENDAR_CREATED", wireOpFor(AuditEventCreated))
	assert.Equal(t, "CALENDAR_UPDATED", wireOpFor(AuditEventUpdated))
	assert.Equal(t, "CALENDAR_DELETED", wireOpFor(AuditEventDeleted))
	assert.Equal(t, "", wireOpFor("something_unknown"))
}
