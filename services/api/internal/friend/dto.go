package friend

import "time"

type SendRequestPayload struct {
	ReceiverID string `json:"receiver_id" binding:"required,uuid"`
	Message    string `json:"message" binding:"max=500"`
}

type BlockPayload struct {
	TargetID string `json:"target_id" binding:"required,uuid"`
}

type CreateInvitePayload struct {
	// MaxUses defaults to 1 (single-use). 0 means "use the default".
	MaxUses int `json:"max_uses" binding:"min=0,max=100"`
	// TTLSeconds defaults to 7 days. 0 means "use the default".
	TTLSeconds int `json:"ttl_seconds" binding:"min=0,max=2592000"`
}

type AcceptInvitePayload struct {
	Token string `json:"token" binding:"required,min=8,max=128"`
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

type InviteResponse struct {
	ID        string    `json:"id"`
	Token     string    `json:"token,omitempty"`
	URL       string    `json:"url,omitempty"`
	MaxUses   int       `json:"max_uses"`
	Uses      int       `json:"uses"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
