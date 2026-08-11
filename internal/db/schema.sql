-- Service Desk — PostgreSQL schema.
-- Run this once against a fresh database before starting the server with
-- DATABASE_URL set. Slice/struct fields (skills, line items, notes, etc.)
-- are stored as JSONB rather than normalized into join tables — a
-- deliberate simplification to keep the schema close to the in-memory
-- model it replaces.

-- One sequence per entity prefix, used to build the same human-readable
-- IDs the in-memory stores generated (e.g. "TCK-0001").
CREATE SEQUENCE IF NOT EXISTS seq_ticket;
CREATE SEQUENCE IF NOT EXISTS seq_user;
CREATE SEQUENCE IF NOT EXISTS seq_engineer;
CREATE SEQUENCE IF NOT EXISTS seq_attendance;
CREATE SEQUENCE IF NOT EXISTS seq_leave;
CREATE SEQUENCE IF NOT EXISTS seq_timesheet;
CREATE SEQUENCE IF NOT EXISTS seq_entity;
CREATE SEQUENCE IF NOT EXISTS seq_invoice;
CREATE SEQUENCE IF NOT EXISTS seq_vendor_bill;
CREATE SEQUENCE IF NOT EXISTS seq_adjustment;
CREATE SEQUENCE IF NOT EXISTS seq_applicant;
CREATE SEQUENCE IF NOT EXISTS seq_assignment;
CREATE SEQUENCE IF NOT EXISTS seq_contract;

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    role          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tickets (
    id                      TEXT PRIMARY KEY,
    title                   TEXT NOT NULL,
    description             TEXT NOT NULL DEFAULT '',
    client_name             TEXT NOT NULL,
    project_name            TEXT NOT NULL DEFAULT '',
    project_type            TEXT NOT NULL DEFAULT '',
    domain                  TEXT NOT NULL DEFAULT '',
    site_address            TEXT NOT NULL DEFAULT '',
    priority                TEXT NOT NULL,
    sla_due_at              TIMESTAMPTZ,
    source                  TEXT NOT NULL,
    status                  TEXT NOT NULL,
    assigned_engineer_id    TEXT NOT NULL DEFAULT '',
    assigned_engineer_name  TEXT NOT NULL DEFAULT '',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS engineers (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    email             TEXT NOT NULL,
    phone             TEXT NOT NULL DEFAULT '',
    skills            JSONB NOT NULL DEFAULT '[]',
    location          TEXT NOT NULL DEFAULT '',
    hourly_rate       DOUBLE PRECISION NOT NULL DEFAULT 0,
    day_rate          DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL DEFAULT '',
    approved_projects JSONB NOT NULL DEFAULT '[]',
    available         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS attendance_records (
    id           TEXT PRIMARY KEY,
    engineer_id  TEXT NOT NULL,
    date         TEXT NOT NULL,
    status       TEXT NOT NULL,
    check_in_at  TIMESTAMPTZ NOT NULL,
    check_out_at TIMESTAMPTZ,
    location     TEXT NOT NULL DEFAULT '',
    notes        TEXT NOT NULL DEFAULT '',
    UNIQUE (engineer_id, date)
);

-- Safe to run against an already-existing database: adds the column if
-- an earlier deploy created this table before check-out existed.
ALTER TABLE attendance_records ADD COLUMN IF NOT EXISTS check_out_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS leave_requests (
    id          TEXT PRIMARY KEY,
    engineer_id TEXT NOT NULL,
    from_date   TEXT NOT NULL,
    to_date     TEXT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    decided_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS timesheets (
    id            TEXT PRIMARY KEY,
    ticket_id     TEXT NOT NULL,
    engineer_id   TEXT NOT NULL,
    check_in_at   TIMESTAMPTZ NOT NULL,
    check_out_at  TIMESTAMPTZ NOT NULL,
    job_type      TEXT NOT NULL,
    file_url      TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL,
    billed_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency      TEXT NOT NULL DEFAULT '',
    invoice_id    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS entities (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    country          TEXT NOT NULL DEFAULT '',
    default_currency TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS invoices (
    id           TEXT PRIMARY KEY,
    entity_id    TEXT NOT NULL,
    client_name  TEXT NOT NULL,
    currency     TEXT NOT NULL,
    line_items   JSONB NOT NULL DEFAULT '[]',
    total        DOUBLE PRECISION NOT NULL DEFAULT 0,
    status       TEXT NOT NULL,
    amount_paid  DOUBLE PRECISION NOT NULL DEFAULT 0,
    due_date     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at      TIMESTAMPTZ,
    paid_at      TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS vendor_bills (
    id          TEXT PRIMARY KEY,
    entity_id   TEXT NOT NULL,
    vendor_name TEXT NOT NULL,
    ticket_id   TEXT NOT NULL DEFAULT '',
    amount      DOUBLE PRECISION NOT NULL,
    currency    TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS adjustment_notes (
    id         TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    type       TEXT NOT NULL,
    amount     DOUBLE PRECISION NOT NULL,
    reason     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS applicants (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    email        TEXT NOT NULL,
    phone        TEXT NOT NULL DEFAULT '',
    job_title    TEXT NOT NULL,
    resume_url   TEXT NOT NULL DEFAULT '',
    stage        TEXT NOT NULL,
    recruiter_id TEXT NOT NULL DEFAULT '',
    notes        JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assignments (
    id               TEXT PRIMARY KEY,
    engineer_id      TEXT NOT NULL,
    project_name     TEXT NOT NULL,
    client_name      TEXT NOT NULL,
    hourly_rate      DOUBLE PRECISION NOT NULL DEFAULT 0,
    day_rate         DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency         TEXT NOT NULL DEFAULT '',
    travel_allowance DOUBLE PRECISION NOT NULL DEFAULT 0,
    tools_allowance  DOUBLE PRECISION NOT NULL DEFAULT 0,
    start_date       TEXT NOT NULL,
    end_date         TEXT NOT NULL DEFAULT '',
    active           BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS contracts (
    id            TEXT PRIMARY KEY,
    client_name   TEXT NOT NULL,
    entity_id     TEXT NOT NULL DEFAULT '',
    project_scope TEXT NOT NULL DEFAULT '',
    billing_terms TEXT NOT NULL DEFAULT '',
    start_date    TEXT NOT NULL,
    expiry_date   TEXT NOT NULL,
    auto_renew    BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tickets_client ON tickets (client_name);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets (status);
CREATE INDEX IF NOT EXISTS idx_timesheets_ticket ON timesheets (ticket_id);
CREATE INDEX IF NOT EXISTS idx_timesheets_engineer ON timesheets (engineer_id);
CREATE INDEX IF NOT EXISTS idx_vendor_bills_ticket ON vendor_bills (ticket_id);
CREATE INDEX IF NOT EXISTS idx_applicants_job_title ON applicants (job_title);
CREATE INDEX IF NOT EXISTS idx_assignments_engineer ON assignments (engineer_id);
