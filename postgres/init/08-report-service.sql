-- =============================================================
-- report_db — report-service
-- =============================================================
-- Not present in ServiceHub.sql. architecture-design.md §1 has
-- report-service own "Daily rollups per hub" — one row per
-- dealership per day, built by the end-of-day job in §6 of that
-- document from the accumulated event log (ServiceCompleted,
-- MaterialsConsumed, PaymentReceived).
-- =============================================================

\connect report_db

CREATE TABLE daily_hub_report (
    id                     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dealership_id          BIGINT NOT NULL,   -- owned by dealership-service, no FK
    report_date            date NOT NULL,
    revenue                numeric(14,2) NOT NULL DEFAULT 0,
    bookings_completed     int NOT NULL DEFAULT 0,
    material_cost          numeric(14,2) NOT NULL DEFAULT 0,
    technician_utilization jsonb,             -- per-technician booked-hours snapshot
    generated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (dealership_id, report_date)
);
