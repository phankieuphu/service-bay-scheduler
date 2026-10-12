-- =============================================================
-- Database-per-service bootstrap
-- =============================================================
-- Runs once, against the default bootstrap database (POSTGRES_DB),
-- when the postgres container's data volume is empty. Creates one
-- database per bounded context from architecture-design.md §1.
--
-- Same Postgres instance for all of them is fine at this scale
-- (architecture-design.md §7) — split to separate instances only
-- once one service's load actually demands it. What matters now is
-- that no two services' tables live in the same database, so a
-- cross-schema foreign key is never even possible by accident.
--
-- identity-, customer- and vehicle-service own their tables through
-- migrations (<service>/migrations, applied on startup), so their
-- databases are only created here. The 04-08 scripts are schemas for
-- services that aren't built yet; each moves into its service's
-- migrations when that service is.
-- =============================================================

CREATE DATABASE identity_db;
CREATE DATABASE customer_db;
CREATE DATABASE vehicle_db;
CREATE DATABASE dealership_db;
CREATE DATABASE scheduler_db;
CREATE DATABASE billing_db;
CREATE DATABASE notification_db;
CREATE DATABASE report_db;
