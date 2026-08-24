package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"investplanner/server/internal/database"
	"investplanner/server/internal/domain"
)

func (s *Store) SaveBenchmarkMapping(mapping database.BenchmarkMapping) (database.BenchmarkMapping, error) {
	if err := domain.ValidateBenchmarkMapping(mapping.FundReturnBasis, mapping.BenchmarkReturnBasis, mapping.EffectiveFrom, mapping.EffectiveTo); err != nil {
		return mapping, err
	}
	if mapping.ID == "" {
		mapping.ID = uuid.NewString()
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var overlaps int64
		query := tx.Model(&database.BenchmarkMapping{}).
			Where("fund_share_id = ? AND benchmark_id = ? AND id <> ?", mapping.FundShareID, mapping.BenchmarkID, mapping.ID).
			Where("effective_from < COALESCE(?, '9999-12-31') AND COALESCE(effective_to, '9999-12-31') > ?", mapping.EffectiveTo, mapping.EffectiveFrom)
		if err := query.Count(&overlaps).Error; err != nil {
			return err
		}
		if overlaps > 0 {
			return ErrConflict
		}
		return tx.Save(&mapping).Error
	})
	return mapping, err
}

func (s *Store) ActiveFundSharesByBenchmark(benchmarkID string, at time.Time) ([]database.FundShare, error) {
	var shares []database.FundShare
	err := s.DB.Joins("JOIN benchmark_mappings ON benchmark_mappings.fund_share_id = fund_shares.id").
		Where("fund_shares.active = ? AND benchmark_mappings.benchmark_id = ?", true, benchmarkID).
		Where("fund_shares.effective_from <= ? AND (fund_shares.effective_to IS NULL OR fund_shares.effective_to > ?)", at, at).
		Where("benchmark_mappings.effective_from <= ? AND (benchmark_mappings.effective_to IS NULL OR benchmark_mappings.effective_to > ?)", at, at).
		Order("fund_shares.fund_code, fund_shares.share_class").Find(&shares).Error
	return shares, err
}

func (s *Store) SaveSnapshot(snapshot database.MarketDataSnapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = uuid.NewString()
	}
	return s.DB.Create(&snapshot).Error
}

func (s *Store) GetSnapshot(dataType, entityKey string) (database.MarketDataSnapshot, error) {
	var snapshot database.MarketDataSnapshot
	var id string
	err := s.DB.Model(&database.MarketDataSnapshot{}).
		Select("id").
		Where("data_type = ? AND entity_key = ?", dataType, entityKey).
		Order("fetched_at DESC, created_at DESC").
		Limit(1).
		Scan(&id).Error
	if err == nil && id == "" {
		return snapshot, ErrNotFound
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return snapshot, ErrNotFound
	}
	if err != nil {
		return snapshot, err
	}
	err = s.DB.Where("id = ?", id).First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return snapshot, ErrNotFound
	}
	return snapshot, err
}

func (s *Store) ListSnapshots(dataType string) ([]database.MarketDataSnapshot, error) {
	var snapshots []database.MarketDataSnapshot
	err := s.DB.Where("data_type = ?", dataType).Order("entity_key, fetched_at DESC").Find(&snapshots).Error
	return snapshots, err
}

func (s *Store) UpsertDiscoveredFundShare(share database.FundShare) error {
	if share.ID == "" {
		share.ID = uuid.NewString()
	}
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fund_code"}, {Name: "share_class"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "venue", "active", "updated_by", "updated_at"}),
	}).Create(&share).Error
}

func (s *Store) UpsertVerifiedFundShareMapping(share database.FundShare, benchmarkID string, effectiveAt time.Time) error {
	if share.ID == "" {
		share.ID = uuid.NewString()
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "fund_code"}, {Name: "share_class"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "venue", "active", "updated_by", "updated_at"}),
		}).Create(&share).Error; err != nil {
			return err
		}
		var persisted database.FundShare
		if err := tx.Where("fund_code = ? AND share_class = ?", share.FundCode, share.ShareClass).First(&persisted).Error; err != nil {
			return err
		}
		share = persisted
		if err := tx.Model(&database.BenchmarkMapping{}).
			Where("fund_share_id = ? AND effective_to IS NULL AND benchmark_id <> ?", share.ID, benchmarkID).
			Update("effective_to", effectiveAt).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&database.BenchmarkMapping{}).
			Where("fund_share_id = ? AND benchmark_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", share.ID, benchmarkID, effectiveAt, effectiveAt).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		mapping := database.BenchmarkMapping{
			ID: uuid.NewString(), FundShareID: share.ID, BenchmarkID: benchmarkID,
			FundReturnBasis: "adjusted_nav", BenchmarkReturnBasis: "total_return",
			EffectiveFrom: effectiveAt, CreatedBy: share.CreatedBy, UpdatedBy: share.UpdatedBy,
		}
		return tx.Create(&mapping).Error
	})
}

func (s *Store) ListActiveFundShares(at time.Time) ([]database.FundShare, error) {
	var shares []database.FundShare
	err := s.DB.Where("active = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", true, at, at).Order("fund_code").Find(&shares).Error
	return shares, err
}

func (s *Store) ListActiveBenchmarkMappings(at time.Time) ([]database.BenchmarkMapping, error) {
	var mappings []database.BenchmarkMapping
	err := s.DB.Where("effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", at, at).Order("benchmark_id").Find(&mappings).Error
	return mappings, err
}
