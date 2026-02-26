-- 000008_phase8_affiliates.down.sql
-- Drop all 19 affiliate tables in reverse order

DROP TABLE IF EXISTS affiliate_activity_log;
DROP TABLE IF EXISTS affiliate_sub_affiliates;
DROP TABLE IF EXISTS affiliate_media;
DROP TABLE IF EXISTS affiliate_notifications;
DROP TABLE IF EXISTS affiliate_reports_cache;
DROP TABLE IF EXISTS affiliate_payments;
DROP TABLE IF EXISTS affiliate_invoice_lines;
DROP TABLE IF EXISTS affiliate_invoices;
DROP TABLE IF EXISTS affiliate_commissions;
DROP TABLE IF EXISTS affiliate_fees;
DROP TABLE IF EXISTS affiliate_player_refs;
DROP TABLE IF EXISTS affiliate_clicks;
DROP TABLE IF EXISTS affiliate_links;
DROP TABLE IF EXISTS affiliate_deals;
DROP TABLE IF EXISTS affiliate_plan_tiers;
DROP TABLE IF EXISTS affiliate_plans;
DROP TABLE IF EXISTS affiliate_tokens;
DROP TABLE IF EXISTS affiliates;
DROP TABLE IF EXISTS affiliate_brands;
