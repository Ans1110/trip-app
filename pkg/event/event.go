package event

import (
	"time"

	"github.com/google/uuid"
)

const (
	// Auth events
	EventUserRegistered  = "auth.registered"
	EventUserVerified    = "auth.verified"
	EventPasswordReset   = "auth.password_reset"
	EventUserDeactivated = "auth.deactivated"
	EventUserBlocked     = "auth.blocked"

	// Trip events
	EventTripUpdated       = "trip.updated"
	EventTripDeleted       = "trip.deleted"
	EventMemberJoined      = "trip.member_joined"
	EventMemberLeft        = "trip.member_left"
	EventActivityCreated   = "trip.activity_created"
	EventPackingItemPacked = "trip.packing_item_packed"
	EventTodoUpdated       = "trip.todo_updated"
	EventTodoReminder      = "todo.reminder"

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
	EventFriendRequest      = "friend.request"
	EventFriendAccepted     = "friend.accepted"

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
