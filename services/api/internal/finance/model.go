package finance

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type SplitStrategy string

const (
	SplitEqual      SplitStrategy = "equal"
	SplitCustom     SplitStrategy = "custom"
	SplitPercentage SplitStrategy = "percentage"
)

type SettlementStatus string

const (
	SettlementProposed  SettlementStatus = "proposed"
	SettlementConfirmed SettlementStatus = "confirmed"
	SettlementCancelled SettlementStatus = "cancelled"
)

type Expense struct {
	ID             uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID         uuid.UUID           `gorm:"type:uuid;column:trip_id;not null;index"`
	PaidBy         uuid.UUID           `gorm:"type:uuid;column:paid_by;not null"`
	Amount         decimal.Decimal     `gorm:"column:amount;type:numeric(18,4);not null"`
	Currency       string              `gorm:"column:currency;type:char(3);not null"`
	AmountBase     decimal.Decimal     `gorm:"column:amount_base;type:numeric(18,4);not null"`
	RateToBase     decimal.NullDecimal `gorm:"column:rate_to_base;type:numeric(20,10)"`
	Description    string              `gorm:"column:description;not null;default:''"`
	Category       string              `gorm:"column:category;not null;default:'other'"`
	SplitStrategy  SplitStrategy       `gorm:"column:split_strategy;not null;default:'equal'"`
	OccurredAt     time.Time           `gorm:"column:occurred_at;not null"`
	ReceiptAssetID *uuid.UUID          `gorm:"type:uuid;column:receipt_asset_id"`
	CreatedBy      uuid.UUID           `gorm:"type:uuid;column:created_by;not null"`
	CreatedAt      time.Time           `gorm:"column:created_at"`
	UpdatedAt      time.Time           `gorm:"column:updated_at"`
	DeletedAt      gorm.DeletedAt      `gorm:"column:deleted_at;index"`
}

func (Expense) TableName() string { return "finance.expense" }

type ExpenseShare struct {
	ExpenseID  uuid.UUID           `gorm:"type:uuid;primaryKey;column:expense_id"`
	UserID     uuid.UUID           `gorm:"type:uuid;primaryKey;column:user_id"`
	Amount     decimal.Decimal     `gorm:"column:amount;type:numeric(18,4);not null"`
	AmountBase decimal.Decimal     `gorm:"column:amount_base;type:numeric(18,4);not null"`
	SharePct   decimal.NullDecimal `gorm:"column:share_pct;type:numeric(9,6)"`
	PaidAt    *time.Time `gorm:"column:paid_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (ExpenseShare) TableName() string { return "finance.expense_share" }

type FxSnapshot struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Base      string          `gorm:"column:base;type:char(3);not null"`
	Quote     string          `gorm:"column:quote;type:char(3);not null"`
	Rate      decimal.Decimal `gorm:"column:rate;type:numeric(20,10);not null"`
	AsOf      time.Time       `gorm:"column:as_of;type:date;not null"`
	Source    string          `gorm:"column:source;not null;default:'manual'"`
	CreatedAt time.Time       `gorm:"column:created_at"`
}

func (FxSnapshot) TableName() string { return "finance.fx_snapshot" }

type Budget struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID    uuid.UUID       `gorm:"type:uuid;column:trip_id;not null;index"`
	Category  string          `gorm:"column:category;not null"`
	Amount    decimal.Decimal `gorm:"column:amount;type:numeric(18,4);not null"`
	Currency  string          `gorm:"column:currency;type:char(3);not null"`
	CreatedBy uuid.UUID       `gorm:"type:uuid;column:created_by;not null"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`
}

func (Budget) TableName() string { return "finance.budget" }

type Settlement struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID      uuid.UUID        `gorm:"type:uuid;column:trip_id;not null;index"`
	PayerID     uuid.UUID        `gorm:"type:uuid;column:payer_id;not null"`
	PayeeID     uuid.UUID        `gorm:"type:uuid;column:payee_id;not null"`
	Amount      decimal.Decimal  `gorm:"column:amount;type:numeric(18,4);not null"`
	Currency    string           `gorm:"column:currency;type:char(3);not null"`
	Status      SettlementStatus `gorm:"column:status;not null;default:'proposed'"`
	Note        string           `gorm:"column:note;not null;default:''"`
	CreatedBy   uuid.UUID        `gorm:"type:uuid;column:created_by;not null"`
	CreatedAt   time.Time        `gorm:"column:created_at"`
	ConfirmedAt *time.Time       `gorm:"column:confirmed_at"`
	CancelledAt *time.Time       `gorm:"column:cancelled_at"`
}

func (Settlement) TableName() string { return "finance.settlement" }
