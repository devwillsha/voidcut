BEGIN;

CREATE TABLE activity_events (
    event_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    version TEXT NOT NULL,
    event_timestamp TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT activity_events_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX activity_events_session_timestamp_idx
    ON activity_events (session_id, event_timestamp);

CREATE INDEX activity_events_trace_id_idx
    ON activity_events (trace_id);

CREATE INDEX activity_events_user_id_idx
    ON activity_events (user_id);

CREATE TABLE jobs (
    job_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    input_url TEXT,
    output_url TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT jobs_state_check CHECK (
        state IN ('pending', 'queued', 'running', 'succeeded', 'failed', 'cancelled')
    )
);

CREATE INDEX jobs_user_id_created_at_idx
    ON jobs (user_id, created_at DESC);

CREATE INDEX jobs_session_id_idx
    ON jobs (session_id);

CREATE INDEX jobs_state_idx
    ON jobs (state);

COMMIT;
