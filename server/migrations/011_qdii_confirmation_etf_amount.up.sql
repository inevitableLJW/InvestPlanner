ALTER TABLE qdii_assistant_confirmations
  ADD COLUMN selected_etf_amount_cents bigint NULL AFTER selected_etf_fund_code;

INSERT INTO schema_migrations (version, applied_at) VALUES (11, UTC_TIMESTAMP(3));
