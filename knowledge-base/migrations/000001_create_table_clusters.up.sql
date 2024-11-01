CREATE TABLE IF NOT EXISTS clusters (
    id UUID PRIMARY KEY,
    type TEXT NOT NULL,
    role TEXT NOT NULL,
    raft_id INT,
    ip INET NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
