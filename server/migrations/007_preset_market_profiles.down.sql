UPDATE plan_destinations SET execution_mode = 'unknown', settlement_currency = 'unknown', benchmark_id = '', fx_quota_relevant = false
WHERE name IN (
  '现金', '债券类基金', '纳斯达克100指数（QDII）', '纳指（QDII）', '标普500指数（QDII）', '标普（QDII）',
  '红利低波', 'A股主动基金', '沪深300指数基金', 'A股指数基金（如沪深300）', '中证500指数基金', 'A股', '港股', '美股',
  '纳斯达克100指数（美股）', '纳指（美股）', '标普500指数（美股）', '标普（美股）'
);
