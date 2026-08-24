DELETE benchmark_mappings
FROM benchmark_mappings
JOIN fund_shares ON fund_shares.id = benchmark_mappings.fund_share_id
WHERE benchmark_mappings.created_by = '00000000-0000-0000-0000-000000000000'
  AND fund_shares.share_class = 'ETF'
  AND fund_shares.fund_code IN ('513100', '159941', '513300', '513500', '159612', '159655');
