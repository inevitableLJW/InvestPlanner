UPDATE plan_destinations SET execution_mode = 'local', settlement_currency = 'CNY', benchmark_id = '', fx_quota_relevant = false
WHERE name IN ('现金', '债券类基金', '红利低波', 'A股主动基金', '沪深300指数基金', 'A股指数基金（如沪深300）', '中证500指数基金', 'A股');

UPDATE plan_destinations SET execution_mode = 'qdii', settlement_currency = 'CNY', benchmark_id = 'NASDAQ100_TR', fx_quota_relevant = false
WHERE name IN ('纳斯达克100指数（QDII）', '纳指（QDII）');

UPDATE plan_destinations SET execution_mode = 'qdii', settlement_currency = 'CNY', benchmark_id = 'SP500_TR', fx_quota_relevant = false
WHERE name IN ('标普500指数（QDII）', '标普（QDII）');

UPDATE plan_destinations SET execution_mode = 'direct_hk', settlement_currency = 'HKD', benchmark_id = '', fx_quota_relevant = true
WHERE name = '港股';

UPDATE plan_destinations SET execution_mode = 'direct_us', settlement_currency = 'USD', benchmark_id = '', fx_quota_relevant = true
WHERE name IN ('美股', '纳斯达克100指数（美股）', '纳指（美股）', '标普500指数（美股）', '标普（美股）');
