-- Signed plugin registry. Packages remain inert bytes: this migration does not
-- grant execution permission or activate a runtime. A later review/activation
-- workflow must explicitly approve an entry before any client/server loads it.
CREATE TABLE IF NOT EXISTS public.plugin_catalog (
    plugin_id       TEXT NOT NULL,
    version         TEXT NOT NULL,
    name            TEXT NOT NULL,
    author          TEXT NOT NULL,
    description     TEXT NOT NULL,
    target          TEXT NOT NULL CHECK (target IN ('android', 'server')),
    entrypoint      TEXT NOT NULL,
    permissions     JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(permissions) = 'array'),
    package_sha256  CHAR(64) NOT NULL CHECK (package_sha256 ~ '^[a-f0-9]{64}$'),
    signer_key_id   TEXT NOT NULL,
    signature       TEXT NOT NULL,
    package_data    BYTEA NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'disabled', 'rejected')),
    submitted_by    TEXT NOT NULL DEFAULT '',
    reviewed_by     TEXT NOT NULL DEFAULT '',
    review_note     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at     TIMESTAMPTZ,
    PRIMARY KEY (plugin_id, version)
);

CREATE INDEX IF NOT EXISTS plugin_catalog_status_updated_idx
    ON public.plugin_catalog (status, updated_at DESC);

CREATE INDEX IF NOT EXISTS plugin_catalog_target_status_idx
    ON public.plugin_catalog (target, status, name);
