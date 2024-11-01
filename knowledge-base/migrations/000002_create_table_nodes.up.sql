CREATE TABLE IF NOT EXISTS nodes (
    cluster_id uuid NOT NULL,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL, -- TZ identifier
    cpu INT NOT NULL,
    cpu_arch VARCHAR(255) NOT NULL,
    memory FLOAT NOT NULL, -- bytes
    network_bandwidth FLOAT NOT NULL, -- bps
    ephemeral_storage FLOAT NOT NULL, -- bytes
    energy FLOAT NOT NULL, -- kWh
    pricing FLOAT NOT NULL, -- EUR/h
    is_ready BOOLEAN NOT NULL,
    is_schedulable BOOLEAN NOT NULL,
    is_pid_pressure_exists BOOLEAN NOT NULL,
    is_memory_pressure_exists BOOLEAN NOT NULL,
    is_disk_pressure_exists BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (cluster_id, name),
    FOREIGN KEY (cluster_id) REFERENCES clusters(id)
);
