INSERT INTO fund_shares
  (id, fund_code, share_class, name, venue, active, effective_from, created_by, updated_by, created_at, updated_at)
VALUES
  (UUID(), '270042', 'A', '广发纳斯达克100ETF联接人民币(QDII)A', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
  (UUID(), '006479', 'C', '广发纳斯达克100ETF联接人民币(QDII)C', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
  (UUID(), '050025', 'A', '博时标普500ETF联接A', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)),
  (UUID(), '006075', 'C', '博时标普500ETF联接C', 'otc', true, '2000-01-01', '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3))
ON DUPLICATE KEY UPDATE name = VALUES(name), venue = VALUES(venue), active = true, updated_at = UTC_TIMESTAMP(3);

INSERT INTO benchmark_mappings
  (id, fund_share_id, benchmark_id, fund_return_basis, benchmark_return_basis, effective_from, created_by, updated_by, created_at, updated_at)
SELECT UUID(), id,
  CASE WHEN fund_code IN ('270042', '006479') THEN 'NASDAQ100_TR' ELSE 'SP500_TR' END,
  'adjusted_nav', 'total_return', '2000-01-01',
  '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000', UTC_TIMESTAMP(3), UTC_TIMESTAMP(3)
FROM fund_shares
WHERE (fund_code, share_class) IN (('270042', 'A'), ('006479', 'C'), ('050025', 'A'), ('006075', 'C'));
