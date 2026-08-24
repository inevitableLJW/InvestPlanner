ALTER TABLE plan_destinations
  ADD COLUMN execution_mode varchar(24) NOT NULL DEFAULT 'unknown',
  ADD COLUMN settlement_currency varchar(8) NOT NULL DEFAULT 'unknown',
  ADD COLUMN benchmark_id varchar(80) NOT NULL DEFAULT '',
  ADD COLUMN fx_quota_relevant boolean NOT NULL DEFAULT false;

-- Existing rows deliberately remain unknown. Fund, mapping, channel, override,
-- and snapshot tables are created by the version 6 GORM migration.

