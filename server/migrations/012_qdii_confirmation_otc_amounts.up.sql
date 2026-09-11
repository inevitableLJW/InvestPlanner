ALTER TABLE qdii_assistant_confirmations
    ADD COLUMN selected_otc_plans_json JSON NULL AFTER selected_otc_fund_codes_json;
