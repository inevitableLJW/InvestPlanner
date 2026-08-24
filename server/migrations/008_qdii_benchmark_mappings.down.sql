DELETE benchmark_mappings
FROM benchmark_mappings
JOIN fund_shares ON fund_shares.id = benchmark_mappings.fund_share_id
WHERE benchmark_mappings.created_by = '00000000-0000-0000-0000-000000000000'
  AND (fund_shares.fund_code, fund_shares.share_class) IN (('270042', 'A'), ('006479', 'C'), ('050025', 'A'), ('006075', 'C'));
