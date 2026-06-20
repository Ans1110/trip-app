package friend

import "time"

type SendRequestPayload struct {
	ReceiverID string `json:"receiver_id" binding:"required,uuid"`
	Message    string `json:"message" binding:"max=500"`
}

type BlockPayload struct {
	TargetID string `json:"target_id" binding:"required,uuid"`
}

type UserSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type FriendResponse struct {
	User      UserSummary `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
}

type RequestResponse struct {
	ID        string      `json:"id"`
	Status    string      `json:"status"`
	Message   string      `json:"message,omitempty"`
	Sender    UserSummary `json:"sender"`
	Receiver  UserSummary `json:"receiver"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type BlockResponse struct {
	User      UserSummary `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
}
