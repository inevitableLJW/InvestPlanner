-- Destructive rollback. Back up first; only run after disabling both assistants.
DROP TABLE IF EXISTS market_data_snapshots;
DROP TABLE IF EXISTS channel_limit_overrides;
DROP TABLE IF EXISTS sales_channels;
DROP TABLE IF EXISTS benchmark_mappings;
DROP TABLE IF EXISTS fund_shares;
ALTER TABLE plan_destinations
  DROP COLUMN fx_quota_relevant,
  DROP COLUMN benchmark_id,
  DROP COLUMN settlement_currency,
  DROP COLUMN execution_mode;
DELETE FROM schema_migrations WHERE version = 6;
