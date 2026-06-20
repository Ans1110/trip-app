package event

import (
	"time"

	"github.com/google/uuid"
)

const (
	// Auth events
	EventUserRegistered    = "auth.registered"
	EventUserVerified      = "auth.verified"
	EventUserLoggedIn      = "auth.logged_in"
	EventUserLoggedOut     = "auth.logged_out"
	EventPasswordReset     = "auth.password_reset"
	EventPasswordChanged   = "auth.password_changed"
	EventUserDeactivated   = "auth.deactivated"
	EventUserDeleted       = "auth.deleted"
	EventUserBlocked       = "auth.blocked"

	// Trip events
	EventTripCreated       = "trip.created"
	EventTripUpdated       = "trip.updated"
	EventTripDeleted       = "trip.deleted"
	EventActivityCreated   = "trip.activity_created"
	EventPackingItemPacked = "trip.packing_item_packed"
	EventTodoUpdated       = "trip.todo_updated"
	EventTodoReminder      = "todo.reminder"

	// Room (collaboration space) events. Friend and Room are decoupled, so
	// membership changes are scoped to room, not trip.
	EventRoomMemberJoined    = "room.member_joined"
	EventRoomMemberLeft      = "room.member_left"
	EventRoomMemberRemoved   = "room.member_removed"
	EventRoomCodeRegenerated = "room.code_regenerated"
	EventRoomInviteCreated   = "room.invite_created"
	EventRoomInviteRevoked   = "room.invite_revoked"

	// Calendar events
	EventCalendarEventCreated = "calendar.event_created"
	EventCalendarEventRSVP    = "calendar.event_rsvp"

	// Chat events
	EventChatMessage         = "chat.message"
	EventChatMessageReaction = "chat.message_reaction"
	EventChatMessageMention  = "chat.message_mention"

	// Vote events
	EventVoteCreated = "vote.created"
	EventVoteCast    = "vote.cast"
	EventVoteClosed  = "vote.closed"

	// Finance events
	EventFinanceAdded        = "finance.added"
	EventFinanceSettled      = "finance.settled"
	EventBudgetExceeded      = "finance.budget_exceeded"
	EventSettlementConfirmed = "finance.settlement_confirmed"

	// Article events
	EventArticleLiked      = "article.liked"
	EventArticleCommented  = "article.commented"
	EventArticlePublished  = "article.published"
	EventArticleBookmarked = "article.bookmarked"

	// Profile / social events
	EventUserFollowed       = "profile.followed"
	EventUserProfileUpdated = "profile.updated"
	EventFriendRequestSent      = "friend.request_sent"
	EventFriendRequestAccepted  = "friend.request_accepted"
	EventFriendRequestDeclined  = "friend.request_declined"
	EventFriendRequestCancelled = "friend.request_cancelled"
	EventFriendRemoved          = "friend.removed"
	EventFriendBlocked          = "friend.blocked"
	EventFriendUnblocked        = "friend.unblocked"

	// Location events
	EventUserCheckedIn = "location.checked_in"
	EventWeatherAlert  = "weather.alert"

	// Media events
	EventMediaUploaded  = "media.uploaded"
	EventMediaProcessed = "media.processed"

	// Notification events
	EventNotificationSent = "notification.sent"

	// Moderation events
	EventContentReported = "content.reported"
	EventReportResolved  = "report.resolved"

	// Recap / AI events
	EventRecapGenerate = "recap.generate"
	EventRecapReady    = "recap.ready"
)

type Event struct {
	Type      string
	Payload   any
	UserID    uuid.UUID
	TargetID  uuid.UUID
	Timestamp time.Time
}
