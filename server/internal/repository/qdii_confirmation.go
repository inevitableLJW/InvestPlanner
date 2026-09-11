package repository

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"investplanner/server/internal/database"
)

func (s *Store) GetQDIIConfirmation(userID, planID, month, destinationID string) (database.QDIIAssistantConfirmation, error) {
	var confirmation database.QDIIAssistantConfirmation
	err := s.DB.Model(&database.QDIIAssistantConfirmation{}).
		Joins("JOIN monthly_records ON monthly_records.id = qdii_assistant_confirmations.monthly_record_id").
		Joins("JOIN plans ON plans.id = monthly_records.plan_id").
		Where("plans.user_id = ? AND plans.id = ? AND monthly_records.month = ? AND qdii_assistant_confirmations.destination_id = ?", userID, planID, month, destinationID).
		First(&confirmation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return confirmation, ErrNotFound
	}
	return confirmation, err
}

func (s *Store) ListQDIIConfirmations(userID, planID, month string) ([]database.QDIIAssistantConfirmation, error) {
	var confirmations []database.QDIIAssistantConfirmation
	err := s.DB.Model(&database.QDIIAssistantConfirmation{}).
		Joins("JOIN monthly_records ON monthly_records.id = qdii_assistant_confirmations.monthly_record_id").
		Joins("JOIN plans ON plans.id = monthly_records.plan_id").
		Where("plans.user_id = ? AND plans.id = ? AND monthly_records.month = ?", userID, planID, month).
		Order("qdii_assistant_confirmations.confirmed_at DESC").
		Find(&confirmations).Error
	return confirmations, err
}

func (s *Store) FundSharesByCodes(codes []string) ([]database.FundShare, error) {
	if len(codes) == 0 {
		return []database.FundShare{}, nil
	}
	var shares []database.FundShare
	err := s.DB.Where("fund_code IN ?", codes).
		Order("updated_at DESC, created_at DESC").
		Find(&shares).Error
	return shares, err
}

func (s *Store) SaveQDIIConfirmation(userID, planID, month, destinationID string, selectedOTCJSON, selectedOTCPlansJSON []byte, selectedETF string, selectedETFAmountCents *int64) (database.QDIIAssistantConfirmation, error) {
	var confirmation database.QDIIAssistantConfirmation
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var record database.MonthlyRecord
		if err := tx.Model(&database.MonthlyRecord{}).
			Joins("JOIN plans ON plans.id = monthly_records.plan_id").
			Where("plans.user_id = ? AND plans.id = ? AND monthly_records.month = ?", userID, planID, month).
			First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var allocationCount int64
		if err := tx.Model(&database.MonthlyDestinationAllocation{}).
			Where("monthly_record_id = ? AND destination_id = ?", record.ID, destinationID).
			Count(&allocationCount).Error; err != nil {
			return err
		}
		if allocationCount == 0 {
			return ErrNotFound
		}
		lookupErr := tx.Where("monthly_record_id = ? AND destination_id = ?", record.ID, destinationID).First(&confirmation).Error
		confirmedAt := s.DB.NowFunc().UTC()
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			confirmation = database.QDIIAssistantConfirmation{
				ID: uuid.NewString(), MonthlyRecordID: record.ID, DestinationID: destinationID,
				SelectedOTCFundCodesJSON: selectedOTCJSON, SelectedOTCPlansJSON: selectedOTCPlansJSON,
				SelectedETFFundCode: selectedETF, SelectedETFAmountCents: selectedETFAmountCents,
				ConfirmedAt: confirmedAt,
			}
			return tx.Create(&confirmation).Error
		}
		if lookupErr != nil {
			return lookupErr
		}
		if err := tx.Model(&confirmation).Updates(map[string]any{
			"selected_otc_fund_codes_json": selectedOTCJSON,
			"selected_otc_plans_json":      selectedOTCPlansJSON,
			"selected_etf_fund_code":       selectedETF,
			"selected_etf_amount_cents":    selectedETFAmountCents,
			"confirmed_at":                 confirmedAt,
		}).Error; err != nil {
			return err
		}
		confirmation.SelectedOTCFundCodesJSON = selectedOTCJSON
		confirmation.SelectedOTCPlansJSON = selectedOTCPlansJSON
		confirmation.SelectedETFFundCode = selectedETF
		confirmation.SelectedETFAmountCents = selectedETFAmountCents
		confirmation.ConfirmedAt = confirmedAt
		return nil
	})
	return confirmation, err
}
