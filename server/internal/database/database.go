package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		return nil
	})
}
