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
-- =============================================================

CREATE DATABASE identity_db;
CREATE DATABASE customer_db;
CREATE DATABASE vehicle_db;
CREATE DATABASE dealership_db;
CREATE DATABASE scheduler_db;
CREATE DATABASE billing_db;
CREATE DATABASE notification_db;
CREATE DATABASE report_db;
