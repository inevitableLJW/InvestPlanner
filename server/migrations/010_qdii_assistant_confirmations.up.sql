CREATE TABLE qdii_assistant_confirmations (
  id char(36) PRIMARY KEY,
  monthly_record_id char(36) NOT NULL,
  destination_id char(36) NOT NULL,
  selected_otc_fund_codes_json json NOT NULL,
  selected_etf_fund_code varchar(32) NOT NULL DEFAULT '',
  confirmed_at datetime(3) NOT NULL,
  created_at datetime(3) NULL,
  updated_at datetime(3) NULL,
  UNIQUE KEY idx_qdii_confirmation (monthly_record_id, destination_id),
  CONSTRAINT fk_qdii_confirmation_month FOREIGN KEY (monthly_record_id) REFERENCES monthly_records(id) ON DELETE CASCADE
);

INSERT INTO schema_migrations (version, applied_at) VALUES (10, UTC_TIMESTAMP(3));
