package database

import "time"

type User struct {
	ID           string `gorm:"type:char(36);primaryKey"`
	Username     string `gorm:"size:320;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        string    `gorm:"type:char(36);primaryKey"`
	UserID    string    `gorm:"type:char(36);not null;index"`
	TokenHash string    `gorm:"type:char(64);uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
	RevokedAt *time.Time
	CreatedAt time.Time
	User      User `gorm:"constraint:OnDelete:CASCADE"`
}

type ExpenseSource struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	UserID    string `gorm:"type:char(36);not null;uniqueIndex:idx_user_source_key"`
	Name      string `gorm:"size:100;not null"`
	NameKey   string `gorm:"size:100;not null;uniqueIndex:idx_user_source_key"`
	SortOrder int    `gorm:"not null"`
	Active    bool   `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	User      User `gorm:"constraint:OnDelete:CASCADE"`
}

type Plan struct {
	ID                     string `gorm:"type:char(36);primaryKey"`
	UserID                 string `gorm:"type:char(36);not null;index"`
	Name                   string `gorm:"size:120;not null"`
	Status                 string `gorm:"size:20;not null;index"`
	DefaultContributionBPS int    `gorm:"not null;default:8000"`
	ReserveCents           int64  `gorm:"not null;default:0"`
	RoundingUnitCents      int64  `gorm:"not null;default:10000"`
	Version                int    `gorm:"not null;default:1"`
	ArchivedAt             *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	User                   User              `gorm:"constraint:OnDelete:CASCADE"`
	Destinations           []PlanDestination `gorm:"constraint:OnDelete:CASCADE"`
}

type PlanDestination struct {
	ID                 string `gorm:"type:char(36);primaryKey"`
	PlanID             string `gorm:"type:char(36);not null;index"`
	Name               string `gorm:"size:100;not null"`
	NameKey            string `gorm:"size:100;not null;index"`
	Active             bool   `gorm:"not null;default:true"`
	AllocationBPS      int    `gorm:"not null;default:0"`
	ExecutionMode      string `gorm:"size:24;not null;default:unknown;index"`
	SettlementCurrency string `gorm:"size:8;not null;default:unknown"`
	BenchmarkID        string `gorm:"size:80;not null;default:'';index"`
	FXQuotaRelevant    bool   `gorm:"not null;default:false"`
	SortOrder          int    `gorm:"not null"`
	ArchivedAt         *time.Time
	Version            int `gorm:"not null;default:1"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Plan               Plan `gorm:"constraint:OnDelete:CASCADE"`
}

type FundShare struct {
	ID            string     `gorm:"type:char(36);primaryKey"`
	FundCode      string     `gorm:"size:32;not null;uniqueIndex:idx_fund_share_code_class"`
	ShareClass    string     `gorm:"size:16;not null;uniqueIndex:idx_fund_share_code_class"`
	Name          string     `gorm:"size:160;not null"`
	Venue         string     `gorm:"size:16;not null;index"`
	Active        bool       `gorm:"not null;default:true;index"`
	EffectiveFrom time.Time  `gorm:"not null;index"`
	EffectiveTo   *time.Time `gorm:"index"`
	CreatedBy     string     `gorm:"type:char(36);not null"`
	UpdatedBy     string     `gorm:"type:char(36);not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BenchmarkMapping struct {
	ID                   string     `gorm:"type:char(36);primaryKey"`
	FundShareID          string     `gorm:"type:char(36);not null;index"`
	BenchmarkID          string     `gorm:"size:80;not null;index"`
	FundReturnBasis      string     `gorm:"size:24;not null"`
	BenchmarkReturnBasis string     `gorm:"size:24;not null"`
	EffectiveFrom        time.Time  `gorm:"not null;index"`
	EffectiveTo          *time.Time `gorm:"index"`
	CreatedBy            string     `gorm:"type:char(36);not null"`
	UpdatedBy            string     `gorm:"type:char(36);not null"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	FundShare            FundShare `gorm:"constraint:OnDelete:CASCADE"`
}

type SalesChannel struct {
	ID            string     `gorm:"type:char(36);primaryKey"`
	Code          string     `gorm:"size:40;not null;uniqueIndex"`
	Name          string     `gorm:"size:80;not null"`
	Provider      string     `gorm:"size:40;not null"`
	Active        bool       `gorm:"not null;default:true"`
	EffectiveFrom time.Time  `gorm:"not null;index"`
	EffectiveTo   *time.Time `gorm:"index"`
	CreatedBy     string     `gorm:"type:char(36);not null"`
	UpdatedBy     string     `gorm:"type:char(36);not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ChannelLimitOverride struct {
	ID                   string `gorm:"type:char(36);primaryKey"`
	UserID               string `gorm:"type:char(36);not null;index"`
	FundShareID          string `gorm:"type:char(36);not null;index"`
	SalesChannelID       string `gorm:"type:char(36);not null;index"`
	DailyLimitCents      *int64
	MinimumPurchaseCents *int64
	PurchaseStatus       string     `gorm:"size:24;not null;default:unknown"`
	Source               string     `gorm:"size:120;not null"`
	EffectiveFrom        time.Time  `gorm:"not null;index"`
	EffectiveTo          *time.Time `gorm:"index"`
	CreatedBy            string     `gorm:"type:char(36);not null"`
	UpdatedBy            string     `gorm:"type:char(36);not null"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type MarketDataSnapshot struct {
	ID             string    `gorm:"type:char(36);primaryKey"`
	DataType       string    `gorm:"size:40;not null;index:idx_snapshot_identity"`
	EntityKey      string    `gorm:"size:120;not null;index:idx_snapshot_identity"`
	Provider       string    `gorm:"size:40;not null"`
	SourceURL      string    `gorm:"size:512;not null"`
	ObservedAt     time.Time `gorm:"not null;index"`
	FetchedAt      time.Time `gorm:"not null;index"`
	Status         string    `gorm:"size:16;not null;index"`
	RawHash        string    `gorm:"type:char(64);not null"`
	ErrorCode      string    `gorm:"size:64;not null;default:''"`
	NormalizedJSON []byte    `gorm:"type:json;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MonthlyRecord struct {
	ID                    string `gorm:"type:char(36);primaryKey"`
	PlanID                string `gorm:"type:char(36);not null;uniqueIndex:idx_plan_month"`
	Month                 string `gorm:"type:char(7);not null;uniqueIndex:idx_plan_month"`
	IncomeCents           int64  `gorm:"not null"`
	ExpenseTotalCents     int64  `gorm:"not null"`
	ReserveCents          int64  `gorm:"not null"`
	SurplusCents          int64  `gorm:"not null"`
	InvestableBaseCents   int64  `gorm:"not null"`
	ContributionBPS       int    `gorm:"not null"`
	RecommendedTotalCents int64  `gorm:"not null"`
	RoundingUnitCents     int64  `gorm:"not null"`
	Status                string `gorm:"size:24;not null"`
	Note                  string `gorm:"type:text"`
	Version               int    `gorm:"not null;default:1"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Plan                  Plan                           `gorm:"constraint:OnDelete:CASCADE"`
	Expenses              []MonthlyExpense               `gorm:"constraint:OnDelete:CASCADE"`
	Allocations           []MonthlyDestinationAllocation `gorm:"constraint:OnDelete:CASCADE"`
	QDIIConfirmations     []QDIIAssistantConfirmation    `gorm:"constraint:OnDelete:CASCADE"`
}

type QDIIAssistantConfirmation struct {
	ID                       string `gorm:"type:char(36);primaryKey"`
	MonthlyRecordID          string `gorm:"type:char(36);not null;uniqueIndex:idx_qdii_confirmation"`
	DestinationID            string `gorm:"type:char(36);not null;uniqueIndex:idx_qdii_confirmation"`
	SelectedOTCFundCodesJSON []byte `gorm:"type:json;not null"`
	SelectedOTCPlansJSON     []byte `gorm:"type:json"`
	SelectedETFFundCode      string `gorm:"size:32;not null;default:''"`
	SelectedETFAmountCents   *int64
	ConfirmedAt              time.Time `gorm:"not null"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
	MonthlyRecord            MonthlyRecord `gorm:"constraint:OnDelete:CASCADE"`
}

type MonthlyExpense struct {
	ID              string  `gorm:"type:char(36);primaryKey"`
	MonthlyRecordID string  `gorm:"type:char(36);not null;index"`
	SourceID        *string `gorm:"type:char(36);index"`
	SourceName      string  `gorm:"size:100;not null"`
	AmountCents     int64   `gorm:"not null"`
	SortOrder       int     `gorm:"not null"`
	CreatedAt       time.Time
	MonthlyRecord   MonthlyRecord `gorm:"constraint:OnDelete:CASCADE"`
}

type MonthlyDestinationAllocation struct {
	ID               string  `gorm:"type:char(36);primaryKey"`
	MonthlyRecordID  string  `gorm:"type:char(36);not null;uniqueIndex:idx_record_destination"`
	DestinationID    *string `gorm:"type:char(36);uniqueIndex:idx_record_destination"`
	DestinationName  string  `gorm:"size:100;not null"`
	SortOrder        int     `gorm:"not null"`
	AllocationBPS    int     `gorm:"not null"`
	RecommendedCents int64   `gorm:"not null"`
	ActualCents      int64   `gorm:"not null;default:0"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	MonthlyRecord    MonthlyRecord `gorm:"constraint:OnDelete:CASCADE"`
}

type SchemaMigration struct {
	Version   int       `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}
