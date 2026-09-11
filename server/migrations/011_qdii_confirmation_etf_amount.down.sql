ALTER TABLE qdii_assistant_confirmations
  DROP COLUMN selected_etf_amount_cents;

DELETE FROM schema_migrations WHERE version = 11;
