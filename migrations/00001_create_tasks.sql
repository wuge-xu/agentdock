-- +goose Up

CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    name VARCHAR(128) NOT NULL,
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    max_attempts SMALLINT NOT NULL DEFAULT 1,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT tasks_tenant_idempotency_key_unique
        UNIQUE (tenant_id, idempotency_key),

    CONSTRAINT tasks_idempotency_key_not_blank
        CHECK (btrim(idempotency_key) <> ''),

    CONSTRAINT tasks_name_not_blank
        CHECK (btrim(name) <> ''),

    CONSTRAINT tasks_input_object
        CHECK (jsonb_typeof(input) = 'object'),

    CONSTRAINT tasks_status_valid
        CHECK (
            status IN (
                'created',
                'queued',
                'running',
                'succeeded',
                'failed',
                'cancelled'
            )
        ),

    CONSTRAINT tasks_max_attempts_valid
        CHECK (max_attempts BETWEEN 1 AND 10),

    CONSTRAINT tasks_version_positive
        CHECK (version >= 1),

    CONSTRAINT tasks_updated_at_not_before_created_at
        CHECK (updated_at >= created_at)
);

-- +goose Down

DROP TABLE IF EXISTS tasks;