package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"investplanner/server/internal/domain"
)

func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 1).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.AutoMigrate(
				&User{}, &Session{}, &ExpenseSource{}, &Plan{}, &PlanDestination{},
				&MonthlyRecord{}, &MonthlyExpense{}, &MonthlyDestinationAllocation{},
			); err != nil {
				return fmt.Errorf("migration 1: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 1, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 2).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if tx.Migrator().HasColumn("users", "email") {
				if !tx.Migrator().HasColumn("users", "username") {
					if err := tx.Exec("ALTER TABLE users ADD COLUMN username varchar(320) NULL").Error; err != nil {
						return fmt.Errorf("migration 2 add username: %w", err)
					}
				}
				if err := tx.Exec("UPDATE users SET username = LOWER(TRIM(email)) WHERE username IS NULL OR username = ''").Error; err != nil {
					return fmt.Errorf("migration 2 copy usernames: %w", err)
				}
				if err := tx.Exec("ALTER TABLE users MODIFY COLUMN username varchar(320) NOT NULL").Error; err != nil {
					return fmt.Errorf("migration 2 require username: %w", err)
				}
				if !tx.Migrator().HasIndex("users", "idx_users_username") {
					if err := tx.Exec("CREATE UNIQUE INDEX idx_users_username ON users (username)").Error; err != nil {
						return fmt.Errorf("migration 2 index username: %w", err)
					}
				}
				if err := tx.Migrator().DropColumn("users", "email"); err != nil {
					return fmt.Errorf("migration 2 drop email: %w", err)
				}
			} else if err := tx.AutoMigrate(&User{}); err != nil {
				return fmt.Errorf("migration 2 users: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 2, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 3).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec(`
				UPDATE expense_sources AS source
				JOIN (
					SELECT user_id, MIN(sort_order) AS douyin_order
					FROM expense_sources WHERE name_key = '抖音' GROUP BY user_id
				) AS douyin ON douyin.user_id = source.user_id
				LEFT JOIN expense_sources AS jd ON jd.user_id = source.user_id AND jd.name_key = '京东'
				SET source.sort_order = source.sort_order + 1
				WHERE source.sort_order >= douyin.douyin_order AND jd.id IS NULL
			`).Error; err != nil {
				return fmt.Errorf("migration 3 move sources: %w", err)
			}
			if err := tx.Exec(`
				INSERT INTO expense_sources (id, user_id, name, name_key, sort_order, active, created_at, updated_at)
				SELECT UUID(), user.id, '京东', '京东', COALESCE(douyin.douyin_order - 1, 3), TRUE, NOW(3), NOW(3)
				FROM users AS user
				LEFT JOIN (
					SELECT user_id, MIN(sort_order) AS douyin_order
					FROM expense_sources WHERE name_key = '抖音' GROUP BY user_id
				) AS douyin ON douyin.user_id = user.id
				LEFT JOIN expense_sources AS jd ON jd.user_id = user.id AND jd.name_key = '京东'
				WHERE jd.id IS NULL
			`).Error; err != nil {
				return fmt.Errorf("migration 3 add jd: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 3, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 4).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec(`
				UPDATE expense_sources AS jd
				JOIN (
					SELECT user_id, MIN(sort_order) AS douyin_order
					FROM expense_sources WHERE name_key = '抖音' GROUP BY user_id
				) AS douyin ON douyin.user_id = jd.user_id
				SET jd.sort_order = douyin.douyin_order - 1
				WHERE jd.name_key = '京东' AND jd.sort_order = douyin.douyin_order
			`).Error; err != nil {
				return fmt.Errorf("migration 4 correct jd ordering: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 4, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 5).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec("ALTER TABLE plans MODIFY COLUMN rounding_unit_cents bigint NOT NULL DEFAULT 10000").Error; err != nil {
				return fmt.Errorf("migration 5 update plan rounding default: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 5, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 6).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.AutoMigrate(
				&PlanDestination{}, &FundShare{}, &BenchmarkMapping{}, &SalesChannel{},
				&ChannelLimitOverride{}, &MarketDataSnapshot{},
			); err != nil {
				return fmt.Errorf("migration 6 market data models: %w", err)
			}
			// Existing destinations are intentionally left unclassified. Product
			// semantics forbid inferring execution mode from their display names.
			if err := tx.Model(&PlanDestination{}).Where("execution_mode = '' OR execution_mode IS NULL").
				Updates(map[string]any{"execution_mode": "unknown", "settlement_currency": "unknown", "fx_quota_relevant": false}).Error; err != nil {
				return fmt.Errorf("migration 6 classify existing destinations: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 6, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 7).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			for _, preset := range domain.SystemPresets() {
				names := append([]string{preset.Name}, preset.Aliases...)
				if err := tx.Model(&PlanDestination{}).Where("name IN ?", names).Updates(map[string]any{
					"execution_mode": preset.Profile.ExecutionMode, "settlement_currency": preset.Profile.SettlementCurrency,
					"benchmark_id": preset.Profile.BenchmarkID, "fx_quota_relevant": preset.Profile.FXQuotaRelevant,
				}).Error; err != nil {
					return fmt.Errorf("migration 7 preset market profiles: %w", err)
				}
			}
			if err := tx.Create(&SchemaMigration{Version: 7, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 8).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec(`
				INSERT INTO fund_shares
					(id, fund_code, share_class, name, venue, active, effective_from, created_by, updated_by, created_at, updated_at)
				VALUES
					(UUID(), '270042', 'A', '广发纳斯达克100ETF联接人民币(QDII)A', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '006479', 'C', '广发纳斯达克100ETF联接人民币(QDII)C', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '050025', 'A', '博时标普500ETF联接A', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '006075', 'C', '博时标普500ETF联接C', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3))
				ON DUPLICATE KEY UPDATE name = VALUES(name), venue = VALUES(venue), active = true, updated_at = UTC_TIMESTAMP(3)
			`).Error; err != nil {
				return fmt.Errorf("migration 8 seed QDII fund shares: %w", err)
			}
			if err := tx.Exec(`
				INSERT INTO benchmark_mappings
					(id, fund_share_id, benchmark_id, fund_return_basis, benchmark_return_basis, effective_from, created_by, updated_by, created_at, updated_at)
				SELECT UUID(), id,
					CASE WHEN fund_code IN ('270042', '006479') THEN 'NASDAQ100_TR' ELSE 'SP500_TR' END,
					'adjusted_nav', 'total_return', '2000-01-01',
					'00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)
				FROM fund_shares
				WHERE (fund_code, share_class) IN (('270042', 'A'), ('006479', 'C'), ('050025', 'A'), ('006075', 'C'))
			`).Error; err != nil {
				return fmt.Errorf("migration 8 seed exact QDII benchmark mappings: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 8, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 9).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec(`
				INSERT INTO fund_shares
					(id, fund_code, share_class, name, venue, active, effective_from, created_by, updated_by, created_at, updated_at)
				VALUES
					(UUID(), '513100', 'ETF', '纳指ETF国泰', 'etf', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '159941', 'ETF', '纳指ETF广发', 'etf', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '513300', 'ETF', '纳斯达克ETF华夏', 'etf', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '513500', 'ETF', '标普500ETF博时', 'etf', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '159612', 'ETF', '标普500ETF国泰', 'etf', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
					(UUID(), '159655', 'ETF', '标普500ETF华夏', 'etf', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3))
				ON DUPLICATE KEY UPDATE name = VALUES(name), venue = VALUES(venue), active = true, updated_at = UTC_TIMESTAMP(3)
			`).Error; err != nil {
				return fmt.Errorf("migration 9 seed QDII ETFs: %w", err)
			}
			if err := tx.Exec(`
				INSERT INTO benchmark_mappings
					(id, fund_share_id, benchmark_id, fund_return_basis, benchmark_return_basis, effective_from, created_by, updated_by, created_at, updated_at)
				SELECT UUID(), id,
					CASE WHEN fund_code IN ('513100', '159941', '513300') THEN 'NASDAQ100_TR' ELSE 'SP500_TR' END,
					'adjusted_nav', 'total_return', '2000-01-01',
					'00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)
				FROM fund_shares
				WHERE share_class = 'ETF' AND fund_code IN ('513100', '159941', '513300', '513500', '159612', '159655')
			`).Error; err != nil {
				return fmt.Errorf("migration 9 seed exact QDII ETF mappings: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 9, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 10).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.AutoMigrate(&QDIIAssistantConfirmation{}); err != nil {
				return fmt.Errorf("migration 10 QDII assistant confirmations: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 10, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 11).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.AutoMigrate(&QDIIAssistantConfirmation{}); err != nil {
				return fmt.Errorf("migration 11 QDII ETF confirmation amount: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 11, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}

		count = 0
		if err := tx.Model(&SchemaMigration{}).Where("version = ?", 12).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.AutoMigrate(&QDIIAssistantConfirmation{}); err != nil {
				return fmt.Errorf("migration 12 QDII OTC confirmation amounts: %w", err)
			}
			if err := tx.Create(&SchemaMigration{Version: 12, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
