package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"investplanner/server/internal/database"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{DB: db}
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *Store) CreateUser(username, passwordHash string, sourceNames []string) (database.User, error) {
	user := database.User{ID: uuid.NewString(), Username: normalize(username), PasswordHash: passwordHash}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		for index, name := range sourceNames {
			source := database.ExpenseSource{
				ID: uuid.NewString(), UserID: user.ID, Name: name, NameKey: normalize(name), SortOrder: index, Active: true,
			}
			if err := tx.Create(&source).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return user, err
}

func (s *Store) FindUserByUsername(username string) (database.User, error) {
	var user database.User
	if err := s.DB.Where("username = ?", normalize(username)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, ErrNotFound
		}
		return user, err
	}
	return user, nil
}

func (s *Store) CreateSession(userID, tokenHash string, expiresAt time.Time) (database.Session, error) {
	session := database.Session{
		ID: uuid.NewString(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt.UTC(),
	}
	return session, s.DB.Create(&session).Error
}

func (s *Store) FindActiveSession(tokenHash string, now time.Time) (database.Session, error) {
	var session database.Session
	err := s.DB.Preload("User").
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, now.UTC()).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return session, ErrNotFound
	}
	return session, err
}

func (s *Store) RevokeSession(tokenHash string) error {
	now := time.Now().UTC()
	result := s.DB.Model(&database.Session{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", &now)
	return result.Error
}

func (s *Store) ListSources(userID string) ([]database.ExpenseSource, error) {
	var sources []database.ExpenseSource
	err := s.DB.Where("user_id = ?", userID).Order("sort_order ASC, created_at ASC").Find(&sources).Error
	return sources, err
}

func (s *Store) SaveSource(userID string, source database.ExpenseSource) (database.ExpenseSource, error) {
	source.Name = strings.TrimSpace(source.Name)
	source.NameKey = normalize(source.Name)
	if source.ID == "" {
		source.ID = uuid.NewString()
		source.UserID = userID
		return source, s.DB.Create(&source).Error
	}
	result := s.DB.Model(&database.ExpenseSource{}).
		Where("id = ? AND user_id = ?", source.ID, userID).
		Updates(map[string]any{
			"name": source.Name, "name_key": source.NameKey, "sort_order": source.SortOrder, "active": source.Active,
		})
	if result.Error != nil {
		return source, result.Error
	}
	if result.RowsAffected == 0 {
		return source, ErrNotFound
	}
	return source, s.DB.Where("id = ? AND user_id = ?", source.ID, userID).First(&source).Error
}

func (s *Store) CreatePlan(userID, name string, defaults []string) (database.Plan, error) {
	plan := database.Plan{
		ID: uuid.NewString(), UserID: userID, Name: strings.TrimSpace(name), Status: "draft",
		DefaultContributionBPS: 8000, ReserveCents: 0, RoundingUnitCents: 100, Version: 1,
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		for index, destinationName := range defaults {
			destination := database.PlanDestination{
				ID: uuid.NewString(), PlanID: plan.ID, Name: destinationName, NameKey: normalize(destinationName),
				Active: true, AllocationBPS: 0, SortOrder: index, Version: 1,
			}
			if err := tx.Create(&destination).Error; err != nil {
				return err
			}
			plan.Destinations = append(plan.Destinations, destination)
		}
		return nil
	})
	return plan, err
}

func (s *Store) ListPlans(userID string, includeArchived bool) ([]database.Plan, error) {
	var plans []database.Plan
	query := s.DB.Where("user_id = ?", userID)
	if !includeArchived {
		query = query.Where("status <> ?", "archived")
	}
	err := query.Preload("Destinations", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC, created_at ASC")
	}).Order("updated_at DESC").Find(&plans).Error
	return plans, err
}

func (s *Store) GetPlan(userID, planID string) (database.Plan, error) {
	var plan database.Plan
	err := s.DB.Where("id = ? AND user_id = ?", planID, userID).
		Preload("Destinations", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, created_at ASC")
		}).First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return plan, ErrNotFound
	}
	return plan, err
}

func (s *Store) UpdatePlan(userID string, plan database.Plan, destinations []database.PlanDestination, expectedVersion int) (database.Plan, error) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var current database.Plan
		if err := tx.Where("id = ? AND user_id = ?", plan.ID, userID).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if current.Version != expectedVersion {
			return ErrConflict
		}
		var archivedAt *time.Time
		if plan.Status == "archived" {
			now := time.Now().UTC()
			archivedAt = &now
		}
		result := tx.Model(&database.Plan{}).
			Where("id = ? AND user_id = ? AND version = ?", plan.ID, userID, expectedVersion).
			Updates(map[string]any{
				"name":                     strings.TrimSpace(plan.Name),
				"status":                   plan.Status,
				"default_contribution_bps": plan.DefaultContributionBPS,
				"reserve_cents":            plan.ReserveCents,
				"rounding_unit_cents":      plan.RoundingUnitCents,
				"archived_at":              archivedAt,
				"version":                  gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrConflict
		}

		var existing []database.PlanDestination
		if err := tx.Where("plan_id = ?", plan.ID).Find(&existing).Error; err != nil {
			return err
		}
		existingByID := map[string]database.PlanDestination{}
		for _, destination := range existing {
			existingByID[destination.ID] = destination
		}
		seen := map[string]struct{}{}
		for _, destination := range destinations {
			destination.Name = strings.TrimSpace(destination.Name)
			destination.NameKey = normalize(destination.Name)
			destination.PlanID = plan.ID
			if destination.ID == "" {
				destination.ID = uuid.NewString()
				destination.Version = 1
				if err := tx.Create(&destination).Error; err != nil {
					return err
				}
			} else {
				if _, ok := existingByID[destination.ID]; !ok {
					return ErrNotFound
				}
				result := tx.Model(&database.PlanDestination{}).
					Where("id = ? AND plan_id = ?", destination.ID, plan.ID).
					Updates(map[string]any{
						"name": destination.Name, "name_key": destination.NameKey,
						"active": destination.Active, "allocation_bps": destination.AllocationBPS,
						"sort_order": destination.SortOrder, "version": gorm.Expr("version + 1"),
					})
				if result.Error != nil {
					return result.Error
				}
			}
			seen[destination.ID] = struct{}{}
		}
		for _, destination := range existing {
			if _, ok := seen[destination.ID]; ok {
				continue
			}
			var references int64
			if err := tx.Model(&database.MonthlyDestinationAllocation{}).
				Where("destination_id = ?", destination.ID).Count(&references).Error; err != nil {
				return err
			}
			if references > 0 {
				now := time.Now().UTC()
				if err := tx.Model(&database.PlanDestination{}).Where("id = ? AND plan_id = ?", destination.ID, plan.ID).
					Updates(map[string]any{"active": false, "archived_at": &now}).Error; err != nil {
					return err
				}
			} else if err := tx.Where("id = ? AND plan_id = ?", destination.ID, plan.ID).
				Delete(&database.PlanDestination{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return database.Plan{}, err
	}
	return s.GetPlan(userID, plan.ID)
}

func (s *Store) ArchivePlan(userID, planID string, expectedVersion int) error {
	now := time.Now().UTC()
	result := s.DB.Model(&database.Plan{}).
		Where("id = ? AND user_id = ? AND version = ?", planID, userID, expectedVersion).
		Updates(map[string]any{"status": "archived", "archived_at": &now, "version": gorm.Expr("version + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := s.DB.Model(&database.Plan{}).Where("id = ? AND user_id = ?", planID, userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return ErrNotFound
		}
		return ErrConflict
	}
	return nil
}

func (s *Store) GetMonth(userID, planID, month string) (database.MonthlyRecord, error) {
	var record database.MonthlyRecord
	err := s.DB.Joins("JOIN plans ON plans.id = monthly_records.plan_id").
		Where("monthly_records.plan_id = ? AND monthly_records.month = ? AND plans.user_id = ?", planID, month, userID).
		Preload("Expenses", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") }).
		Preload("Allocations", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") }).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return record, ErrNotFound
	}
	return record, err
}

func (s *Store) ListMonths(userID, planID string) ([]database.MonthlyRecord, error) {
	var records []database.MonthlyRecord
	err := s.DB.Joins("JOIN plans ON plans.id = monthly_records.plan_id").
		Where("monthly_records.plan_id = ? AND plans.user_id = ?", planID, userID).
		Preload("Allocations", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") }).
		Order("monthly_records.month DESC").Find(&records).Error
	return records, err
}

func (s *Store) SaveMonth(userID string, record database.MonthlyRecord, expenses []database.MonthlyExpense, allocations []database.MonthlyDestinationAllocation, expectedVersion *int) (database.MonthlyRecord, error) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var plan database.Plan
		if err := tx.Where("id = ? AND user_id = ?", record.PlanID, userID).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var existing database.MonthlyRecord
		find := tx.Where("plan_id = ? AND month = ?", record.PlanID, record.Month).First(&existing)
		if errors.Is(find.Error, gorm.ErrRecordNotFound) {
			if expectedVersion != nil {
				return ErrConflict
			}
			record.ID = uuid.NewString()
			record.Version = 1
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		} else if find.Error != nil {
			return find.Error
		} else {
			if expectedVersion == nil || existing.Version != *expectedVersion {
				return ErrConflict
			}
			record.ID = existing.ID
			result := tx.Model(&database.MonthlyRecord{}).
				Where("id = ? AND plan_id = ? AND version = ?", existing.ID, record.PlanID, *expectedVersion).
				Updates(map[string]any{
					"income_cents": record.IncomeCents, "expense_total_cents": record.ExpenseTotalCents,
					"reserve_cents": record.ReserveCents, "surplus_cents": record.SurplusCents,
					"investable_base_cents": record.InvestableBaseCents, "contribution_bps": record.ContributionBPS,
					"recommended_total_cents": record.RecommendedTotalCents, "rounding_unit_cents": record.RoundingUnitCents,
					"status": record.Status, "note": record.Note, "version": gorm.Expr("version + 1"),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrConflict
			}
			if err := tx.Where("monthly_record_id = ?", existing.ID).Delete(&database.MonthlyExpense{}).Error; err != nil {
				return err
			}
			if err := tx.Where("monthly_record_id = ?", existing.ID).Delete(&database.MonthlyDestinationAllocation{}).Error; err != nil {
				return err
			}
		}
		for i := range expenses {
			expenses[i].ID = uuid.NewString()
			expenses[i].MonthlyRecordID = record.ID
		}
		if len(expenses) > 0 {
			if err := tx.Create(&expenses).Error; err != nil {
				return err
			}
		}
		for i := range allocations {
			allocations[i].ID = uuid.NewString()
			allocations[i].MonthlyRecordID = record.ID
		}
		if len(allocations) > 0 {
			if err := tx.Create(&allocations).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return database.MonthlyRecord{}, err
	}
	return s.GetMonth(userID, record.PlanID, record.Month)
}

func (s *Store) UpdateActuals(userID, planID, month string, actuals map[string]int64, expectedVersion int) (database.MonthlyRecord, error) {
	record, err := s.GetMonth(userID, planID, month)
	if err != nil {
		return record, err
	}
	if record.Version != expectedVersion {
		return record, ErrConflict
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		total := int64(0)
		for allocationID, amount := range actuals {
			result := tx.Model(&database.MonthlyDestinationAllocation{}).
				Where("id = ? AND monthly_record_id = ?", allocationID, record.ID).
				Update("actual_cents", amount)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrNotFound
			}
			total += amount
		}
		status := "complete"
		if record.RecommendedTotalCents == 0 && total == 0 {
			status = "not_required"
		} else if record.RecommendedTotalCents > 0 && total == 0 {
			status = "not_started"
		} else if total < record.RecommendedTotalCents {
			status = "partial"
		}
		result := tx.Model(&database.MonthlyRecord{}).
			Where("id = ? AND plan_id = ? AND version = ?", record.ID, planID, expectedVersion).
			Updates(map[string]any{"status": status, "version": gorm.Expr("version + 1")})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrConflict
		}
		return nil
	})
	if err != nil {
		return database.MonthlyRecord{}, err
	}
	return s.GetMonth(userID, planID, month)
}

func (s *Store) DeleteMonth(userID, planID, month string) error {
	record, err := s.GetMonth(userID, planID, month)
	if err != nil {
		return err
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("monthly_record_id = ?", record.ID).Delete(&database.MonthlyDestinationAllocation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("monthly_record_id = ?", record.ID).Delete(&database.MonthlyExpense{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ? AND plan_id = ?", record.ID, planID).Delete(&database.MonthlyRecord{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *Store) PreviousMonthSources(userID, planID, beforeMonth string) ([]database.MonthlyExpense, error) {
	var record database.MonthlyRecord
	err := s.DB.Joins("JOIN plans ON plans.id = monthly_records.plan_id").
		Where("monthly_records.plan_id = ? AND monthly_records.month < ? AND plans.user_id = ?", planID, beforeMonth, userID).
		Order("monthly_records.month DESC").First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []database.MonthlyExpense{}, nil
	}
	if err != nil {
		return nil, err
	}
	var expenses []database.MonthlyExpense
	if err := s.DB.Where("monthly_record_id = ?", record.ID).Order("sort_order ASC").Find(&expenses).Error; err != nil {
		return nil, err
	}
	return expenses, nil
}

func (s *Store) EnsurePlanOwned(userID, planID string) error {
	var count int64
	if err := s.DB.Model(&database.Plan{}).Where("id = ? AND user_id = ?", planID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: plan", ErrNotFound)
	}
	return nil
}
