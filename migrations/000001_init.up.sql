CREATE SCHEMA crm;

-- Справочник источников лидов (сайт, олимпиада, ...)
CREATE TABLE crm.sources(
    code TEXT PRIMARY KEY CHECK (code ~ '^[a-z_]{2,50}$'),
    name TEXT NOT NULL    CHECK (char_length(name) BETWEEN 1 AND 100)
);

CREATE TABLE crm.employees(
    id         UUID        PRIMARY KEY,
    full_name  TEXT        NOT NULL CHECK(char_length(full_name) BETWEEN 1 AND 255),
    email      TEXT        UNIQUE   CHECK(char_length(email) BETWEEN 3 AND 255),
    role       TEXT        NOT NULL DEFAULT 'manager'
                           CHECK(role IN ('manager', 'head_of_sales', 'analyst')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE crm.leads(
    id            UUID        PRIMARY KEY,
    number        BIGINT      GENERATED ALWAYS AS IDENTITY UNIQUE,
    external_id   TEXT        UNIQUE CHECK (char_length(external_id) BETWEEN 1 AND 100),
    version       BIGINT      NOT NULL DEFAULT 1,

    source_code  TEXT        NOT NULL REFERENCES crm.sources (code),
    owner_id      UUID        NOT NULL REFERENCES crm.employees (id),
    current_stage TEXT        NOT NULL DEFAULT 'new'
                              CHECK(current_stage IN ('new', 'qualified', 'proposal', 'won', 'lost')),
    
    full_name     TEXT        NOT NULL CHECK (char_length(full_name) BETWEEN 1 AND 255),
    phone         TEXT        CHECK(phone ~ '^\+[0-9]{10,15}$'),
    email         TEXT        CHECK (char_length(email) BETWEEN 3 AND 255),
    comment       TEXT        CHECK(char_length(comment) BETWEEN 1 AND 2000),

    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT leads_updated_after_created CHECK (updated_at >= created_at),
    CONSTRAINT leads_contact_required      CHECK (phone IS NOT NULL OR email IS NOT NULL)
);

CREATE TABLE crm.lead_stages(
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lead_id    UUID        NOT NULL REFERENCES crm.leads (id) ON DELETE CASCADE,
    stage      TEXT        NOT NULL
                           CHECK(stage IN ('new', 'qualified', 'proposal', 'won', 'lost')),
    entered_at TIMESTAMPTZ NOT NULL,
    left_at    TIMESTAMPTZ,

    CONSTRAINT lead_stages_left_after_entered CHECK (left_at IS NULL OR left_at >= entered_at)
);

CREATE TABLE crm.import_logs(
    id            BIGINT       GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    file_hash     TEXT         NOT NULL UNIQUE CHECK (file_hash ~ '^[0-9a-f]{64}$'),
    file_name     TEXT         CHECK (char_length(file_name) BETWEEN 1 AND 255),
    rows_imported INTEGER      NOT NULL CHECK(rows_imported>=0),
    imported_at   TIMESTAMPTZ  NOT NULL
);

CREATE INDEX leads_source_code_idx   ON crm.leads (source_code);
CREATE INDEX leads_owner_id_idx       ON crm.leads (owner_id);
CREATE INDEX leads_current_stage_idx  ON crm.leads (current_stage);

CREATE INDEX lead_stages_lead_id_entered_at_idx ON crm.lead_stages (lead_id, entered_at);

CREATE UNIQUE INDEX lead_stages_one_open_per_lead_idx
    ON crm.lead_stages (lead_id)
    WHERE left_at IS NULL;
