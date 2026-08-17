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
	ID            string `gorm:"type:char(36);primaryKey"`
	PlanID        string `gorm:"type:char(36);not null;index"`
	Name          string `gorm:"size:100;not null"`
	NameKey       string `gorm:"size:100;not null;index"`
	Active        bool   `gorm:"not null;default:true"`
	AllocationBPS int    `gorm:"not null;default:0"`
	SortOrder     int    `gorm:"not null"`
	ArchivedAt    *time.Time
	Version       int `gorm:"not null;default:1"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Plan          Plan `gorm:"constraint:OnDelete:CASCADE"`
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
