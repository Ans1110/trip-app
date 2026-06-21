package trip

import "time"

type UserSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email,omitempty"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// ---- Trip ----

type CreateTripPayload struct {
	Title       string `json:"title" binding:"required,min=1,max=120"`
	Description string `json:"description" binding:"max=2000"`
	CoverImage  string `json:"cover_image" binding:"max=500"`
	StartDate   string `json:"start_date" binding:"required"` // YYYY-MM-DD
	EndDate     string `json:"end_date" binding:"required"`   // YYYY-MM-DD
}

type UpdateTripPayload struct {
	Title       *string `json:"title" binding:"omitempty,min=1,max=120"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	CoverImage  *string `json:"cover_image" binding:"omitempty,max=500"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	Status      *string `json:"status" binding:"omitempty,oneof=planning ongoing completed"`
}

type TripResponse struct {
	ID          string      `json:"id"`
	Owner       UserSummary `json:"owner"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	CoverImage  string      `json:"cover_image,omitempty"`
	StartDate   string      `json:"start_date"`
	EndDate     string      `json:"end_date"`
	Status      string      `json:"status"`
	MemberCount int         `json:"member_count"`
	MyRole      string      `json:"my_role,omitempty"`
	Room        *RoomBrief  `json:"room,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type RoomBrief struct {
	ID       string `json:"id"`
	RoomCode string `json:"room_code"`
}

// ListTripsQuery captures GET /trips query string options.
type ListTripsQuery struct {
	Status string // planning|ongoing|completed (empty = all)
	Q      string // case-insensitive title search
	Role   string // owner|member|all (empty = all)
	Limit  int
	Offset int
}

// ---- Room ----

type RoomMemberResponse struct {
	User     UserSummary `json:"user"`
	Role     string      `json:"role"`
	JoinedAt time.Time   `json:"joined_at"`
}

type RoomResponse struct {
	ID        string               `json:"id"`
	TripID    string               `json:"trip_id"`
	RoomCode  string               `json:"room_code"`
	Members   []RoomMemberResponse `json:"members"`
	CreatedAt time.Time            `json:"created_at"`
}

// RoomPreview is what GET /rooms/by-code returns before the user commits to
// joining. Sensitive bits (member list, owner email) are intentionally omitted.
type RoomPreview struct {
	TripID      string      `json:"trip_id"`
	RoomID      string      `json:"room_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	CoverImage  string      `json:"cover_image,omitempty"`
	StartDate   string      `json:"start_date"`
	EndDate     string      `json:"end_date"`
	Status      string      `json:"status"`
	Owner       UserSummary `json:"owner"`
	MemberCount int         `json:"member_count"`
	AlreadyJoined bool      `json:"already_joined"`
}

// ---- Join ----

type JoinRoomPayload struct {
	Code string `json:"code" binding:"required"`
}

type JoinRoomResponse struct {
	TripID        string `json:"trip_id"`
	RoomID        string `json:"room_id"`
	AlreadyMember bool   `json:"already_member"`
}

// ---- Itinerary ----

type CreateItineraryPayload struct {
	Day         int        `json:"day" binding:"required,min=1"`
	Title       string     `json:"title" binding:"max=200"`
	Description string     `json:"description" binding:"max=2000"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	Location    string     `json:"location" binding:"max=200"`
	Latitude    *float64   `json:"latitude" binding:"omitempty,min=-90,max=90"`
	Longitude   *float64   `json:"longitude" binding:"omitempty,min=-180,max=180"`
	SortOrder   int        `json:"sort_order"`
}

type UpdateItineraryPayload struct {
	Day         *int       `json:"day" binding:"omitempty,min=1"`
	Title       *string    `json:"title" binding:"omitempty,max=200"`
	Description *string    `json:"description" binding:"omitempty,max=2000"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	Location    *string    `json:"location" binding:"omitempty,max=200"`
	Latitude    *float64   `json:"latitude" binding:"omitempty,min=-90,max=90"`
	Longitude   *float64   `json:"longitude" binding:"omitempty,min=-180,max=180"`
	SortOrder   *int       `json:"sort_order"`
}

type ReorderItineraryPayload struct {
	ItemIDs []string `json:"item_ids" binding:"required,dive,uuid"`
}

type ItineraryResponse struct {
	ID          string     `json:"id"`
	TripID      string     `json:"trip_id"`
	Day         int        `json:"day"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Location    string     `json:"location"`
	Latitude    *float64   `json:"latitude,omitempty"`
	Longitude   *float64   `json:"longitude,omitempty"`
	SortOrder   int        `json:"sort_order"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ---- Todo ----

type CreateTodoPayload struct {
	Title      string     `json:"title" binding:"required,min=1,max=200"`
	AssigneeID *string    `json:"assignee_id" binding:"omitempty,uuid"`
	DueDate    *time.Time `json:"due_date"`
	Priority   *string    `json:"priority" binding:"omitempty,oneof=low normal high"`
	Tags       []string   `json:"tags" binding:"omitempty,max=10,dive,min=1,max=30"`
}

type UpdateTodoPayload struct {
	Title       *string    `json:"title" binding:"omitempty,min=1,max=200"`
	AssigneeID  *string    `json:"assignee_id" binding:"omitempty,uuid"`
	IsCompleted *bool      `json:"is_completed"`
	DueDate     *time.Time `json:"due_date"`
	Priority    *string    `json:"priority" binding:"omitempty,oneof=low normal high"`
	Tags        *[]string  `json:"tags" binding:"omitempty,max=10,dive,min=1,max=30"`
	SortOrder   *int       `json:"sort_order"`
}

type ReorderTodosPayload struct {
	TodoIDs []string `json:"todo_ids" binding:"required,dive,uuid"`
}

type TodoResponse struct {
	ID          string     `json:"id"`
	TripID      string     `json:"trip_id"`
	Title       string     `json:"title"`
	IsCompleted bool       `json:"is_completed"`
	AssigneeID  *string    `json:"assignee_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    string     `json:"priority"`
	Tags        []string   `json:"tags"`
	SortOrder   int        `json:"sort_order"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
