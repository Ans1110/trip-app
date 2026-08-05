package stream

const (
	StreamNotification = "events:notification"
	GroupNotification  = "notification-group"

	StreamAnalytics = "events:analytics"
	GroupAnalytics  = "analytics-group"

	StreamMediaUploaded = "media:uploaded"
	GroupMedia          = "thumbnail-processor-group"

	StreamModeration = "events:moderation"
	GroupModeration  = "moderation-group"

	StreamRecap = "events:recap_generate"
	GroupRecap  = "recap-generator-group"

	StreamTripEvents = "events:trip"
	GroupTripEvents  = "trip-events-group"

	StreamProfileEvents = "events:profile"
	GroupProfileFanout  = "profile-fanout-group"

	StreamPostEvents = "events:post"
)
