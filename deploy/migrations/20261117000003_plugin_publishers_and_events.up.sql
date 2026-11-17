-- Publisher keys are explicit trust roots for plugin package signatures.
CREATE TABLE IF NOT EXISTS public.plugin_publishers (
    signer_key_id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    public_key BYTEA NOT NULL CHECK (octet_length(public_key) = 32),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_by TEXT NOT NULL DEFAULT '',
    revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.plugin_catalog_events (
    id BIGSERIAL PRIMARY KEY,
    plugin_id TEXT NOT NULL,
    version TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('submitted', 'approved', 'rejected', 'disabled')),
    actor TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS plugin_catalog_events_lookup_idx
    ON public.plugin_catalog_events (plugin_id, version, created_at DESC);
