-- 000006_phase6_gamification.down.sql

DROP INDEX IF EXISTS idx_event_outbox_unpublished;

ALTER TABLE event_outbox DROP COLUMN "publishedAt";

DROP TABLE IF EXISTS reward_grants;
DROP TABLE IF EXISTS player_engagement;
DROP TABLE IF EXISTS player_quest_progress;
DROP TABLE IF EXISTS quests;
